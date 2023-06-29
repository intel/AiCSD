/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package data_organizer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"

	"aicsd/ms-file-watcher/config"
	"aicsd/pkg"
	"aicsd/pkg/types"
)

type ClientImpl struct {
	baseUrl      string // this is the base url to the endpoint
	fileHostname string
	fileJob      map[string]string
}

func NewClient(config *config.Configuration) Client {
	return &ClientImpl{
		baseUrl:      config.DataOrgBaseUrl,
		fileHostname: config.FileHostname,
		fileJob:      config.FileJob,
	}
}

func (c *ClientImpl) NotifyNewFile(filename string) error {
	notifyNewFileUrl := fmt.Sprintf("%s%s", c.baseUrl, pkg.EndpointNotifyNewFile)

	path, name := filepath.Split(filename)
	ext := filepath.Ext(filename)

	// build up the job object
	jobEntry := types.Job{
		InputFile: types.FileInfo{
			Hostname:   c.fileHostname,
			DirName:    path,
			Name:       name,
			Extension:  ext,
			Attributes: c.fileJob,
		},
	}

	body, err := json.Marshal(jobEntry)
	if err != nil {
		return err
	}
	response, err := http.Post(notifyNewFileUrl, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusAlreadyReported {
		return fmt.Errorf("TransmitJob API status not OK: %s", response.Status)
	}
	return nil
}
