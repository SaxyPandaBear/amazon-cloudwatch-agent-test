// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package awsservice

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

const allowedRetries = 5

// TODO: Refactor Structure and Interface for more easier follow that shares the same session
var (
	ctx context.Context
	cwl *cloudwatchlogs.Client
)

// DeleteLogGroupAndStream cleans up a log group and stream by name. This gracefully handles
// ResourceNotFoundException errors from calling the APIs
func DeleteLogGroupAndStream(logGroupName, logStreamName string) {
	DeleteLogStream(logGroupName, logStreamName)
	DeleteLogGroup(logGroupName)
}

// DeleteLogStream cleans up log stream by name
func DeleteLogStream(logGroupName, logStreamName string) {
	cwlClient, clientContext, err := getCloudWatchLogsClient()
	if err != nil {
		log.Printf("Error occurred while creating CloudWatch Logs SDK client: %v", err)
		return // terminate gracefully so this alone doesn't cause integration test failures
	}

	// catch ResourceNotFoundException when deleting the log group and log stream, as these
	// are not useful exceptions to log errors on during cleanup
	var rnf *types.ResourceNotFoundException

	_, err = cwlClient.DeleteLogStream(*clientContext, &cloudwatchlogs.DeleteLogStreamInput{
		LogGroupName:  aws.String(logGroupName),
		LogStreamName: aws.String(logStreamName),
	})
	if err != nil && !errors.As(err, &rnf) {
		log.Printf("Error occurred while deleting log stream %s: %v", logStreamName, err)
	}
}

// DeleteLogGroup cleans up log group by name
func DeleteLogGroup(logGroupName string) {
	cwlClient, clientContext, err := getCloudWatchLogsClient()
	if err != nil {
		log.Printf("Error occurred while creating CloudWatch Logs SDK client: %v", err)
		return // terminate gracefully so this alone doesn't cause integration test failures
	}

	// catch ResourceNotFoundException when deleting the log group and log stream, as these
	// are not useful exceptions to log errors on during cleanup
	var rnf *types.ResourceNotFoundException

	_, err = cwlClient.DeleteLogGroup(*clientContext, &cloudwatchlogs.DeleteLogGroupInput{
		LogGroupName: aws.String(logGroupName),
	})
	if err != nil && !errors.As(err, &rnf) {
		log.Printf("Error occurred while deleting log group %s: %v", logGroupName, err)
	}
}

// ValidateLogs queries a given LogGroup/LogStream combination given the start and end times, and executes an
// arbitrary validator function on the found logs.
func ValidateLogs(logGroup, logStream string, since, until *time.Time, validator func(logs []string) bool) (bool, error) {
	log.Printf("Checking %s/%s since %s", logGroup, logStream, since.UTC().Format(time.RFC3339))

	foundLogs, err := getLogsSince(logGroup, logStream, since, until)
	if err != nil {
		return false, err
	}

	return validator(foundLogs), nil
}

// getLogsSince makes GetLogEvents API calls, paginates through the results for the given time frame, and returns
// the raw log strings
func getLogsSince(logGroup, logStream string, since, until *time.Time) ([]string, error) {
	foundLogs := make([]string, 0)

	cwlClient, clientContext, err := getCloudWatchLogsClient()
	if err != nil {
		return foundLogs, err
	}

	// https://docs.aws.amazon.com/AmazonCloudWatchLogs/latest/APIReference/API_GetLogEvents.html
	// GetLogEvents can return an empty result while still having more log events on a subsequent page,
	// so rather than expecting all the events to show up in one GetLogEvents API call, we need to paginate.
	params := &cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  aws.String(logGroup),
		LogStreamName: aws.String(logStream),
		StartFromHead: aws.Bool(true), // read from the beginning
	}

	var sinceMs int64
	if since != nil {
		sinceMs = since.UnixNano() / 1e6 // convert to millisecond timestamp
		params.StartTime = aws.Int64(sinceMs)
	}

	var untilMs int64
	if until != nil {
		untilMs = until.UnixNano() / 1e6
		params.EndTime = aws.Int64(untilMs)
	}

	var output *cloudwatchlogs.GetLogEventsOutput
	var nextToken *string
	attempts := 0

	for {
		if nextToken != nil {
			params.NextToken = nextToken
		}
		output, err = cwlClient.GetLogEvents(*clientContext, params)

		attempts += 1

		if err != nil {
			var rnf *types.ResourceNotFoundException
			if errors.As(err, &rnf) && attempts <= allowedRetries {
				// The log group/stream hasn't been created yet, so wait and retry
				time.Sleep(time.Minute)
				continue
			}

			// if the error is not a ResourceNotFoundException, we should fail here.
			return foundLogs, err
		}

		for _, e := range output.Events {
			foundLogs = append(foundLogs, *e.Message)
		}

		if nextToken != nil && output.NextForwardToken != nil && *output.NextForwardToken == *nextToken {
			// From the docs: If you have reached the end of the stream, it returns the same token you passed in.
			log.Printf("Done paginating log events for %s/%s and found %d logs", logGroup, logStream, len(foundLogs))
			break
		}

		nextToken = output.NextForwardToken
	}
	return foundLogs, nil
}

// IsLogGroupExists confirms whether the logGroupName exists or not
func IsLogGroupExists(logGroupName string) bool {
	cwlClient, clientContext, err := getCloudWatchLogsClient()
	if err != nil {
		log.Println("error occurred while creating CWL client", err)
		return false
	}

	describeLogGroupInput := cloudwatchlogs.DescribeLogGroupsInput{
		LogGroupNamePrefix: aws.String(logGroupName),
	}

	describeLogGroupOutput, err := cwlClient.DescribeLogGroups(*clientContext, &describeLogGroupInput)

	if err != nil {
		log.Println("error occurred while calling DescribeLogGroups", err)
		return false
	}

	return len(describeLogGroupOutput.LogGroups) > 0
}

// getCloudWatchLogsClient returns a singleton SDK client for interfacing with CloudWatch Logs
func getCloudWatchLogsClient() (*cloudwatchlogs.Client, *context.Context, error) {
	if cwl == nil {
		ctx = context.Background()
		c, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return nil, nil, err
		}

		cwl = cloudwatchlogs.NewFromConfig(c)
	}
	return cwl, &ctx, nil
}
