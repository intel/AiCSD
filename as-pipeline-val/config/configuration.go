/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package config

import (
	"fmt"

	"aicsd/pkg/helpers"

	"github.com/edgexfoundry/app-functions-sdk-go/v3/pkg/interfaces"
)

type Configuration struct {
	SimBaseUrl            string
	PipelineUrl           string
	DeviceProfileLocation string
	DeviceProfileName     string
	DeviceName            string
	ResourceName          string
}

func New(service interfaces.ApplicationService) (*Configuration, error) {
	var err error
	config := Configuration{}
	protocol := "http"

	config.SimBaseUrl, err = helpers.GetUrlFromAppSetting(service, "Sim", protocol, false)
	if err != nil {
		return nil, err
	}

	pipelineBaseUrl, err := helpers.GetUrlFromAppSetting(service, "Pipeline", protocol, false)
	if err != nil {
		return nil, err
	}
	pipelineEndpoint, err := helpers.GetAppSetting(service, "PipelineEndpoint", false)
	if err != nil {
		return nil, err
	}
	config.PipelineUrl = fmt.Sprintf("%s%s", pipelineBaseUrl, pipelineEndpoint)

	config.DeviceProfileName, err = helpers.GetAppSetting(service, "DeviceProfileName", false)
	if err != nil {
		return nil, err
	}

	config.DeviceName, err = helpers.GetAppSetting(service, "DeviceName", false)
	if err != nil {
		return nil, err
	}

	config.ResourceName, err = helpers.GetAppSetting(service, "ResourceName", false)
	if err != nil {
		return nil, err
	}
	return &config, nil
}
