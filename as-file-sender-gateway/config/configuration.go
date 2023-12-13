/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/edgexfoundry/app-functions-sdk-go/v3/pkg/interfaces"

	"aicsd/pkg/helpers"
)

type Configuration struct {
	JobRepoBaseUrl      string
	TaskLauncherBaseUrl string
	FileHostname        string
	ArchiveFolder       string
	RejectFolder        string
}

func New(service interfaces.ApplicationService) (*Configuration, error) {
	var err error
	config := Configuration{}
	protocol := "http"
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

	config.ArchiveFolder, err = helpers.GetAppSetting(service, "ArchiveFolder", false)
	if err != nil {
		return nil, err
	}
	if len(config.ArchiveFolder) == 0 {
		return nil, errors.New("must specify at least one folder for output files. None specified")
	}
	if _, err := os.Stat(config.ArchiveFolder); os.IsNotExist(err) {
		return nil, fmt.Errorf("Archive Folder Not Found: %s", config.ArchiveFolder)
	}

	config.RejectFolder, err = helpers.GetAppSetting(service, "RejectFolder", false)
	if err != nil {
		return nil, err
	}
	if len(config.RejectFolder) == 0 {
		return nil, errors.New("must specify at least one folder for reject files. None specified")
	}
	if _, err := os.Stat(config.RejectFolder); os.IsNotExist(err) {
		return nil, fmt.Errorf("RejectFolder Folder Not Found: %s", config.RejectFolder)
	}
	return &config, nil
}
