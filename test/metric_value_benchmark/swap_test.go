// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

//go:build linux && integration
// +build linux,integration

package metric_value_benchmark

import (
	"github.com/aws/amazon-cloudwatch-agent-test/test/metric"
	"github.com/aws/amazon-cloudwatch-agent-test/test/metric/dimension"
	"github.com/aws/amazon-cloudwatch-agent-test/test/status"
	"github.com/aws/amazon-cloudwatch-agent-test/test/test_runner"
	"log"
)

type SwapTestRunner struct {
	test_runner.BaseTestRunner
}

var _ test_runner.ITestRunner = (*SwapTestRunner)(nil)

func (t *SwapTestRunner) Validate() status.TestGroupResult {
	metricsToFetch := t.GetMeasuredMetrics()
	testResults := make([]status.TestResult, 0, len(metricsToFetch))
	for metricName := range metricsToFetch {
		testResults = append(testResults, t.validateSwapMetric(metricName))
	}

	return status.TestGroupResult{
		Name:        t.GetTestName(),
		TestResults: testResults,
	}
}

func (t *SwapTestRunner) GetTestName() string {
	return "Swap"
}

func (t *SwapTestRunner) GetAgentConfigFileName() string {
	return "swap_config.json"
}

func (t *SwapTestRunner) GetMeasuredMetrics() map[string]*metric.Bounds {
	return map[string]*metric.Bounds{
		"swap_free":         nil,
		"swap_used":         nil,
		"swap_used_percent": nil,
	}
}

func (t *SwapTestRunner) validateSwapMetric(metricName string) status.TestResult {
	testResult := status.TestResult{
		Name:   metricName,
		Status: status.FAILED,
	}

	dims, failed := t.DimensionFactory.GetDimensions([]dimension.Instruction{
		{
			Key:   "InstanceId",
			Value: dimension.UnknownDimensionValue(),
		},
	})

	if len(failed) > 0 {
		return testResult
	}

	fetcher := metric.MetricValueFetcher{}
	values, err := fetcher.Fetch(namespace, metricName, dims, metric.AVERAGE)
	log.Printf("metric values are %v", values)
	if err != nil {
		return testResult
	}

	if !isAllValuesGreaterThanOrEqualToZero(metricName, values) {
		return testResult
	}

	testResult.Status = status.SUCCESSFUL
	return testResult
}
