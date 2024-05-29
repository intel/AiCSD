/*********************************************************************
 * Copyright (c) Intel Corporation 2024
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package config

import (
	"github.com/edgexfoundry/app-functions-sdk-go/v2/pkg/interfaces"
	"strings"

	"aicsd/pkg/helpers"
)

type Configuration struct {
	OvmsUrl          string
	OutputStreamHost string
}

func New(service interfaces.ApplicationService) (*Configuration, error) {
	var err error
	config := Configuration{}

	config.OvmsUrl, err = helpers.GetUrlFromAppSetting(service, "OVMSGrpc", "http", false)
	config.OvmsUrl = strings.Replace(config.OvmsUrl, "http://", "", 1)
	config.OutputStreamHost, err = helpers.GetUrlFromAppSetting(service, "OutputStream", "http", false)
	config.OutputStreamHost = strings.Replace(config.OvmsUrl, "http://", "", 1)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
