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
)

type MemTestRunner struct {
	test_runner.BaseTestRunner
}

var _ test_runner.ITestRunner = (*MemTestRunner)(nil)

func (m *MemTestRunner) Validate() status.TestGroupResult {
	metricsToFetch := m.GetMeasuredMetrics()
	testResults := make([]status.TestResult, 0, len(metricsToFetch))
	for name := range metricsToFetch {
		testResults = append(testResults, m.validateMemMetric(name))
	}

	return status.TestGroupResult{
		Name:        m.GetTestName(),
		TestResults: testResults,
	}
}

func (m *MemTestRunner) GetTestName() string {
	return "Mem"
}

func (m *MemTestRunner) GetAgentConfigFileName() string {
	return "mem_config.json"
}

func (m *MemTestRunner) GetMeasuredMetrics() map[string]*metric.Bounds {
	return map[string]*metric.Bounds{
		"mem_active":            nil,
		"mem_available":         nil,
		"mem_available_percent": nil,
		"mem_buffered":          nil,
		"mem_cached":            nil,
		"mem_free":              nil,
		"mem_inactive":          nil,
		"mem_total":             nil,
		"mem_used":              nil,
		"mem_used_percent":      nil,
	}
}

func (m *MemTestRunner) validateMemMetric(metricName string) status.TestResult {
	testResult := status.TestResult{
		Name:   metricName,
		Status: status.FAILED,
	}

	dims, failed := m.DimensionFactory.GetDimensions([]dimension.Instruction{
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
	if err != nil {
		return testResult
	}

	if !isAllValuesGreaterThanOrEqualToZero(metricName, values) {
		return testResult
	}

	testResult.Status = status.SUCCESSFUL
	return testResult
}
