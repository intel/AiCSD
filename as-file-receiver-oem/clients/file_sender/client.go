/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package file_sender

import (
	"aicsd/pkg/auth"
	"aicsd/pkg/werrors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"aicsd/pkg"
)

type SenderClient struct {
	baseUrl     string
	httpTimeout time.Duration
	jwtInfo     *auth.JWTInfo
}

// NewClient is used like a constructor to create a receiver client
func NewClient(baseUrl string, httpTimeout time.Duration, info *auth.JWTInfo) Client {
	client := SenderClient{
		baseUrl:     baseUrl,
		httpTimeout: httpTimeout,
		jwtInfo:     info,
	}
	return &client
}

// TransmitFile takes a Job ID and File ID and makes a get request to the TransmitFile API to get the corresponding job output file
func (c *SenderClient) TransmitFile(jobId, fileId string) ([]byte, error) {
	transmitFileUrl := fmt.Sprintf("%s%s", c.baseUrl, pkg.EndpointTransmitFileJobId)
	transmitFileUrl = strings.Replace(transmitFileUrl, "{"+pkg.JobIdKey+"}", jobId, -1)
	transmitFileUrl = strings.Replace(transmitFileUrl, "{"+pkg.FileIdKey+"}", fileId, -1)

	req, err := http.NewRequest(http.MethodGet, transmitFileUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %s", err.Error())
	}
	err = c.jwtInfo.AddAuthHeader(req)
	if err != nil {
		return nil, werrors.WrapErr(err, pkg.ErrAuthHeader)
	}

	client := &http.Client{
		Timeout: c.httpTimeout,
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to TransmitFile: %s", err.Error())
	}
	defer response.Body.Close()

	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %s", err.Error())
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TransmitFile returned not ok status %s: %s", response.Status, respBody)
	}

	return respBody, nil
}

// ArchiveFile takes a Job ID and makes a POST request to archive the file on the gateway
func (c *SenderClient) ArchiveFile(jobId string) error {
	archiveFileUrl := fmt.Sprintf("%s%s", c.baseUrl, pkg.EndpointArchiveFile)
	archiveFileUrl = strings.Replace(archiveFileUrl, "{"+pkg.JobIdKey+"}", jobId, -1)
	req, err := http.NewRequest(http.MethodPost, archiveFileUrl, nil)
	if err != nil {
		return fmt.Errorf("failed to create http request: %s", err.Error())
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
		return fmt.Errorf("failed to ArchiveFile: %s", err.Error())
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("ArchiveFile returned not ok status: %s", response.Status)
	}

	return nil
}

// Retry logic that makes a POST request to the file-sender-gw retry endpoint to refresh the jobs for reprocessing
func (c *SenderClient) Retry() error {
	retryEndpoint := fmt.Sprintf("%s%s", c.baseUrl, pkg.EndpointRetry)
	req, err := http.NewRequest(http.MethodPost, retryEndpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create http request: %s", err.Error())
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
		return fmt.Errorf("failed to Retry: %s", err.Error())
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("Retry returned not ok status: %s", response.Status)
	}

	return nil
}
