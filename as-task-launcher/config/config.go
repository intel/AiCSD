/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package config

import (
	"fmt"
	"time"

	"github.com/edgexfoundry/app-functions-sdk-go/v3/pkg/interfaces"

	"aicsd/pkg/helpers"
)

// TODO: Define your structured custom configuration types. Must be wrapped with an outer struct with
//
//	single element that matches the top level custom configuration element in your configuration.toml file,
//	'AppCustom' in this example. Replace this example with your configuration structure or
//	remove this file if not using structured custom configuration.
type ServiceConfig struct {
	AppCustom AppCustomConfig
}

// AppCustomConfig is example of service's custom structured configuration that is specified in the service's
// configuration.toml file and Configuration Provider (aka Consul), if enabled.
type AppCustomConfig struct {
	ResourceNames string
	SomeService   HostInfo
}

// HostInfo is example struct for defining connection information for external service
type HostInfo struct {
	Host     string
	Port     int
	Protocol string
}

type Configuration struct {
	JobRepoBaseUrl        string
	FileSenderBaseUrl     string
	PipelineStatusBaseUrl string
	DeviceProfileLocation string
	DeviceProfileName     string
	DeviceName            string
	ResourceName          string
	RetryTimeout          int64
	FileHostname          string
	RedisHost             string
	RedisPort             string
}

func New(service interfaces.ApplicationService) (*Configuration, error) {
	var err error
	config := Configuration{}
	protocol := "http"
	config.JobRepoBaseUrl, err = helpers.GetUrlFromAppSetting(service, "JobRepo", protocol, false)
	if err != nil {
		return nil, err
	}

	config.FileSenderBaseUrl, err = helpers.GetUrlFromAppSetting(service, "FileSender", protocol, false)
	if err != nil {
		return nil, err
	}

	config.PipelineStatusBaseUrl, err = helpers.GetUrlFromAppSetting(service, "PipelineStatus", protocol, false)
	if err != nil {
		return nil, err
	}

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

	retryWindow, err := helpers.GetAppSetting(service, "RetryWindow", false)
	if err != nil {
		return nil, err
	}
	retryDuration, err := time.ParseDuration(retryWindow)
	if err != nil {
		return nil, fmt.Errorf("could not parse integer for retry window, got %s: %s", retryWindow, err.Error())
	}
	config.RetryTimeout = retryDuration.Nanoseconds()

	config.RedisHost, err = helpers.GetAppSetting(service, "RedisHost", false)
	if err != nil {
		return nil, err
	}
	config.RedisPort, err = helpers.GetAppSetting(service, "RedisPort", false)
	if err != nil {
		return nil, err
	}

	config.FileHostname, err = helpers.GetAppSetting(service, "FileHostname", false)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
