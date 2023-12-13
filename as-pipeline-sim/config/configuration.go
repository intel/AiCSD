/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package config

import (
	"github.com/edgexfoundry/app-functions-sdk-go/v3/pkg/interfaces"

	"aicsd/pkg/helpers"
)

type Configuration struct {
	GetiUrl string
}

func New(service interfaces.ApplicationService) (*Configuration, error) {
	var err error
	config := Configuration{}

	config.GetiUrl, err = helpers.GetUrlFromAppSetting(service, "Geti", "http", false)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
