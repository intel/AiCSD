/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package config

import (
	"aicsd/pkg/helpers"
	"fmt"
	"github.com/edgexfoundry/app-functions-sdk-go/v2/pkg/interfaces"
	"os"
)

type Configuration struct {
	RedisHost         string
	RedisPort         string
	LocalizationFiles []string
}

func New(service interfaces.ApplicationService) (*Configuration, error) {
	config := Configuration{}
	var err error

	config.RedisHost, err = helpers.GetAppSetting(service, "RedisHost", false)
	if err != nil {
		return nil, err
	}
	config.RedisPort, err = helpers.GetAppSetting(service, "RedisPort", false)
	if err != nil {
		return nil, err
	}

	config.LocalizationFiles, err = service.GetAppSettingStrings("LocalizationFiles")
	if len(config.LocalizationFiles) == 0 {
		return nil, fmt.Errorf("must specify at least one localization file. None specified")
	}
	for _, file := range config.LocalizationFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			return nil, fmt.Errorf("localization file named %s not found", file)
		}
	}
	return &config, nil
}
