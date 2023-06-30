/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package config

import (
	"aicsd/pkg/wait"
	"strconv"
	"time"

	"github.com/edgexfoundry/app-functions-sdk-go/v2/pkg/interfaces"

	"aicsd/pkg/helpers"
)

const (
	// these keys need to correspond to job values in the configuration.toml
	jobKeys = "LabName,LabEquipment,Operator"
)

type Configuration struct {
	JobRepoBaseUrl      string
	FileReceiverBaseUrl string
	FileHostname        string
	RetryAttempts       int
	RetryWaitTime       time.Duration
	PrivateKeyPath      string
	JWTKeyPath          string
	JWTAlgorithm        string
	JWTDuration         string
	DependentServices   wait.Services
}

func New(service interfaces.ApplicationService) (*Configuration, error) {
	config := Configuration{}
	var err error

	config.PrivateKeyPath, err = helpers.GetAppSetting(service, "PrivateKeyPath", true)
	if err != nil {
		return nil, err
	}
	config.JWTKeyPath, err = helpers.GetAppSetting(service, "JWTKeyPath", true)
	if err != nil {
		return nil, err
	}
	config.JWTAlgorithm, err = helpers.GetAppSetting(service, "JWTAlgorithm", true)
	if err != nil {
		return nil, err
	}
	config.JWTDuration, err = helpers.GetAppSetting(service, "JWTDuration", true)
	if err != nil {
		return nil, err
	}

	protocol := "http"
	config.DependentServices = wait.Services{wait.ServiceConsul, wait.ServiceJobRepo}
	if len(config.JWTAlgorithm) > 0 && len(config.PrivateKeyPath) > 0 && len(config.JWTKeyPath) > 0 {
		protocol = "https"
		config.DependentServices = wait.Services{}
	}

	// TODO: fix the versioning to use a constant for the version
	config.JobRepoBaseUrl, err = helpers.GetUrlFromAppSetting(service, "JobRepo", protocol, false)
	if err != nil {
		return nil, err
	}

	config.FileReceiverBaseUrl, err = helpers.GetUrlFromAppSetting(service, "FileReceiver", protocol, false)
	if err != nil {
		return nil, err
	}

	config.FileHostname, err = helpers.GetAppSetting(service, "FileHostname", false)
	if err != nil {
		return nil, err
	}

	retryAttemptsValue, err := helpers.GetAppSetting(service, "RetryAttempts", false)
	if err != nil {
		return nil, err
	}
	config.RetryAttempts, err = strconv.Atoi(retryAttemptsValue)
	if err != nil {
		return nil, err
	}

	retryWaitTimeValue, err := helpers.GetAppSetting(service, "RetryWaitTime", false)
	if err != nil {
		return nil, err
	}
	config.RetryWaitTime, err = time.ParseDuration(retryWaitTimeValue)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
