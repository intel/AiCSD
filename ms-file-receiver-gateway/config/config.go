/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package config

import (
	"errors"
	"fmt"
	"os"

	"aicsd/pkg/helpers"

	"github.com/edgexfoundry/app-functions-sdk-go/v3/pkg/interfaces"
)

type Configuration struct {
	BaseFileFolder      string
	JobRepoBaseUrl      string
	TaskLauncherBaseUrl string
	FileHostname        string
}

func New(service interfaces.ApplicationService) (*Configuration, error) {
	config := Configuration{}
	var err error
	protocol := "http"

	config.BaseFileFolder, err = helpers.GetAppSetting(service, "BaseFileFolder", false)
	if err != nil {
		return nil, err
	}
	if len(config.BaseFileFolder) == 0 {
		return nil, errors.New("must specify at least one folder for output files. None specified")
	}
	if _, err := os.Stat(config.BaseFileFolder); os.IsNotExist(err) {
		return nil, fmt.Errorf("Base File Folder Directory Not Found: %s", config.BaseFileFolder)
	}
	// TODO: fix the versioning to use a constant for the version
	config.JobRepoBaseUrl, err = helpers.GetUrlFromAppSetting(service, "JobRepo", protocol, false)
	if err != nil {
		return nil, err
	}

	config.TaskLauncherBaseUrl, err = helpers.GetUrlFromAppSetting(service, "TaskLauncher", protocol, false)
	if err != nil {
		return nil, err
	}

	config.FileHostname, err = helpers.GetAppSetting(service, "FileHostname", false)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
