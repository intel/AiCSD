/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package file_receiver

import (
	"aicsd/pkg"
	"aicsd/pkg/auth"
	"aicsd/pkg/types"
	"aicsd/pkg/werrors"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type ReceiverClient struct {
	baseUrl     string
	httpTimeout time.Duration
	// retryAttempts represents the number of times file should be attempted to be read, if file-read operation fails
	retryAttempts int
	// retryWaitTime represents the sleep time in between each retry file-read operation
	retryWaitTime time.Duration
	jwtInfo       *auth.JWTInfo
}

// NewClient is used like a constructor to create a receiver client
func NewClient(baseUrl string, httpTimeout time.Duration, retryAttempts int, retryWaitTime time.Duration, info *auth.JWTInfo) Client {
	client := ReceiverClient{
		baseUrl:       baseUrl,
		httpTimeout:   httpTimeout,
		retryAttempts: retryAttempts,
		retryWaitTime: retryWaitTime,
		jwtInfo:       info,
	}
	return &client
}

// TransmitJob is used to create and send POST request using job as message body
func (c *ReceiverClient) TransmitJob(entry types.Job) error {
	transmitJobUrl := fmt.Sprintf("%s%s", c.baseUrl, pkg.EndpointTransmitJob)
	body, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, transmitJobUrl, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("could not create request to transmit job: %s", err.Error())
	}
	err = c.jwtInfo.AddAuthHeader(req)
	if err != nil {
		return werrors.WrapErr(err, pkg.ErrAuthHeader)
	}
	client := &http.Client{
		Timeout: c.httpTimeout,
	}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("TransmitJob API status not OK: %s", response.Status)
	}
	return nil
}

// TransmitFile is used to create and send POST request using multipart form
func (c *ReceiverClient) TransmitFile(id string, entry types.FileInfo) (int, error) {
	transmitFileUrl := fmt.Sprintf("%s%s", c.baseUrl, pkg.EndpointTransmitFile)
	//load file data
	fullFileName := filepath.Join(entry.DirName, entry.Name)
	var fileContents []byte
	attempts := 0
	var err error
	for attempts <= c.retryAttempts {
		fileContents, err = os.ReadFile(fullFileName)
		if err == nil {
			break
		}
		var badPathErr *os.PathError
		if !errors.As(err, &badPathErr) {
			break
		}
		time.Sleep(c.retryWaitTime)
		attempts++
	}

	if err != nil {
		return attempts, err
	}

	// create a new request with custom header (id)
	req, err := http.NewRequest(http.MethodPost, transmitFileUrl, bytes.NewReader(fileContents))
	if err != nil {
		return attempts, fmt.Errorf("failed to create http request: %s", err.Error())
	}

	req.Header.Set(pkg.FilenameKey, entry.Name)
	req.Header.Set(pkg.JobIdKey, id)
	err = c.jwtInfo.AddAuthHeader(req)
	if err != nil {
		return attempts, werrors.WrapErr(err, pkg.ErrAuthHeader)
	}

	// make the request
	client := &http.Client{
		Timeout: c.httpTimeout,
	}
	_, err = client.Do(req)
	if err != nil {
		// return just the error so its type can be parsed
		return attempts, err
	}

	return attempts, nil
}
