/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package job_handler

import (
	"aicsd/pkg"
	"aicsd/pkg/auth"
	"aicsd/pkg/types"
	"aicsd/pkg/werrors"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type JobHandlerClient struct {
	baseUrl     string
	httpTimeout time.Duration
	jwtInfo     *auth.JWTInfo
}

// NewClient builds a new job handler client using the passed in base url
func NewClient(baseUrl string, httpTimeout time.Duration, info *auth.JWTInfo) Client {
	client := JobHandlerClient{
		baseUrl:     baseUrl,
		httpTimeout: httpTimeout,
		jwtInfo:     info,
	}
	return &client
}

// DataToHandle marshals the job and sends it as the request body to the DataToHandle endpoint
func (c *JobHandlerClient) HandleJob(job types.Job) error {
	dataToHandleUrl := fmt.Sprintf("%s%s", c.baseUrl, pkg.EndpointDataToHandle)
	body, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("could not marshall job: %s", err.Error())
	}
	req, err := http.NewRequest(http.MethodPost, dataToHandleUrl, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("could not create request to handle job: %s", err.Error())
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
		return fmt.Errorf("could not do request to handle job: %s", err.Error())
	}
	defer response.Body.Close()
	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("could not read response body: %s", err.Error())
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("match task returned not ok status %s: %s", response.Status, respBody)
	}
	return nil
}
