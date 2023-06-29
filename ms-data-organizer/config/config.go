/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package config

import (
	"aicsd/pkg/helpers"
	"aicsd/pkg/types"
	"aicsd/pkg/wait"
	"github.com/edgexfoundry/app-functions-sdk-go/v2/pkg/interfaces"
)

type Configuration struct {
	JobRepoBaseUrl      string
	TaskLauncherBaseUrl string
	FileSenderBaseUrl   string
	FilenameDecoder     types.FilenameDecoder
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

	config.JobRepoBaseUrl, err = helpers.GetUrlFromAppSetting(service, "JobRepo", protocol, false)
	if err != nil {
		return nil, err
	}
	config.TaskLauncherBaseUrl, err = helpers.GetUrlFromAppSetting(service, "TaskLauncher", protocol, false)
	if err != nil {
		return nil, err
	}
	config.FileSenderBaseUrl, err = helpers.GetUrlFromAppSetting(service, "FileSender", "http", false)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
