/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package config

import (
	"aicsd/pkg/helpers"
	"aicsd/pkg/wait"
	"errors"
	"fmt"
	"github.com/edgexfoundry/app-functions-sdk-go/v2/pkg/interfaces"
	"os"
)

type Configuration struct {
	FileHostname      string
	OutputFolder      string
	JobRepoBaseUrl    string
	FileSenderBaseUrl string
	PrivateKeyPath    string
	JWTKeyPath        string
	JWTAlgorithm      string
	JWTDuration       string
	DependentServices wait.Services
}

func New(service interfaces.ApplicationService) (*Configuration, error) {
	var err error
	config := Configuration{}

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
	config.DependentServices = wait.Services{wait.ServiceConsul, wait.ServiceRedis, wait.ServiceJobRepo}
	if len(config.JWTAlgorithm) > 0 && len(config.PrivateKeyPath) > 0 && len(config.JWTKeyPath) > 0 {
		protocol = "https"
		config.DependentServices = wait.Services{}
	}

	config.FileHostname, err = helpers.GetAppSetting(service, "FileHostname", false)
	if err != nil {
		return nil, err
	}

	config.OutputFolder, err = helpers.GetAppSetting(service, "OutputFolder", false)
	if err != nil {
		return nil, err
	}
	if len(config.OutputFolder) == 0 {
		return nil, errors.New("must specify a folder for output files. None specified")
	}
	if _, err := os.Stat(config.OutputFolder); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory not found for OutputFolder: %s", config.OutputFolder)
	}

	config.JobRepoBaseUrl, err = helpers.GetUrlFromAppSetting(service, "JobRepo", protocol, false)
	if err != nil {
		return nil, err
	}

	config.FileSenderBaseUrl, err = helpers.GetUrlFromAppSetting(service, "FileSender", protocol, false)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
