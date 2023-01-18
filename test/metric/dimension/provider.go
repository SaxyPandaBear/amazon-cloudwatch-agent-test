// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

//go:build !windows
// +build !windows

package dimension

import (
	"github.com/aws/amazon-cloudwatch-agent-test/environment"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"log"
)

type ExpectedDimensionValue struct {
	Value *string
}

func (d *ExpectedDimensionValue) IsKnown() bool {
	return d.Value == nil
}

func UnknownDimensionValue() ExpectedDimensionValue {
	return ExpectedDimensionValue{}
}

func GetDimensionFactory(env environment.MetaData) Factory {
	allDimensionProviders := []IProvider{
		&ContainerInsightsDimensionProvider{Provider: Provider{env: env}},
		&HostDimensionProvider{Provider: Provider{env: env}},
		&LocalInstanceIdDimensionProvider{Provider: Provider{env: env}},
		&ECSInstanceIdDimensionProvider{Provider: Provider{env: env}},
		&CustomDimensionProvider{Provider: Provider{env: env}},
	}

	applicableDimensionProviders := []IProvider{}

	for _, provider := range allDimensionProviders {
		if provider.IsApplicable() {
			applicableDimensionProviders = append(applicableDimensionProviders, provider)
		}
	}

	return Factory{applicableDimensionProviders}
}

type Instruction struct {
	Key   string
	Value ExpectedDimensionValue
}

type Factory struct {
	providers []IProvider
}

func (f *Factory) GetDimensions(instructions []Instruction) ([]types.Dimension, []Instruction) {
	resultDimensions := []types.Dimension{}
	unfulfilledInstructions := []Instruction{}
	for _, instruction := range instructions {
		dim := f.executeInstruction(instruction)
		if (dim != types.Dimension{}) {
			resultDimensions = append(resultDimensions, dim)
			log.Printf("Result dim is : %s, %s", *dim.Name, *dim.Value)
		} else {
			unfulfilledInstructions = append(unfulfilledInstructions, instruction)
			log.Printf("unfulfilled dim is : %s, %s", *dim.Name, *dim.Value)
		}
	}

	return resultDimensions, unfulfilledInstructions
}

func (f *Factory) executeInstruction(instruction Instruction) types.Dimension {
	for _, provider := range f.providers {
		dim := provider.GetDimension(instruction)
		if (dim != types.Dimension{}) {
			return dim
		}
	}
	return types.Dimension{}
}

type IProvider interface {
	IsApplicable() bool
	GetDimension(Instruction) types.Dimension
}

type Provider struct {
	env environment.MetaData
}
