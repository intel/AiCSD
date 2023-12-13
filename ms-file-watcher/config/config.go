/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"aicsd/pkg/helpers"

	"github.com/edgexfoundry/app-functions-sdk-go/v3/pkg/interfaces"
)

const (
	// these keys need to correspond to job values in the configuration.toml
	jobKeys = "LabName,LabEquipment,Operator"
)

type Configuration struct {
	FoldersToWatch    []string
	DataOrgBaseUrl    string
	FileJob           map[string]string
	FileHostname      string
	FileExclusionList []string
	App               App
}
type App struct {
	UpdatableSettings UpdatableSettings
}
type UpdatableSettings struct {
	WatchSubfolders   bool
	FileExclusionList string
}

func New(service interfaces.ApplicationService) (*Configuration, error) {
	config := Configuration{}
	var err error
	config.FoldersToWatch, err = service.GetAppSettingStrings("FoldersToWatch")
	if err != nil {
		return nil, err
	}
	if len(config.FoldersToWatch) == 0 {
		return nil, errors.New("must specify at least one folder to watch. None specified")
	}
	for _, dir := range config.FoldersToWatch {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return nil, fmt.Errorf("watch Folder Directory Not Found: %s", dir)
		}
	}

	config.DataOrgBaseUrl, err = helpers.GetUrlFromAppSetting(service, "DataOrg", "http", false)
	if err != nil {
		return nil, err
	}

	//  set up the job from the config file
	config.FileJob = make(map[string]string)

	jobKeysList := strings.Split(jobKeys, ",")
	for _, key := range jobKeysList {
		config.FileJob[key], err = helpers.GetAppSetting(service, key, false)
		if err != nil {
			return nil, err
		}
	}

	config.FileHostname, err = helpers.GetAppSetting(service, "FileHostname", false)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func (c *App) UpdateFromRaw(rawConfig interface{}) bool {
	configuration, ok := rawConfig.(*App)
	if !ok {
		return false
	}

	*c = *configuration

	return true
}
