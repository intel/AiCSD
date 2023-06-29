/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package clients

import (
	"aicsd/as-pipeline-val/types"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type PipelineClient struct {
	pipelineUrl string
	httpTimeout time.Duration
}

func NewClient(pipelineUrl string, httpTimeout time.Duration) Client {
	client := PipelineClient{
		pipelineUrl: pipelineUrl,
		httpTimeout: httpTimeout,
	}
	return &client
}

// GetPipelines is used retrieve all pipelines with their associated information from the pipeline server.
// It returns a slice of pipeline parameters or an error if the endpoint is not correct.
func (c *PipelineClient) GetPipelines() ([]types.PipelineParams, error) {

	resp, err := http.Get(c.pipelineUrl)
	if err != nil {
		return nil, fmt.Errorf("get pipelines %s request error: %s", c.pipelineUrl, err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get pipelines %s status not OK: %s", c.pipelineUrl, resp.Status)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("get pipelines %s read error: %s", c.pipelineUrl, err.Error())
	}

	var pipelineInfo []types.PipelineParams
	err = json.Unmarshal(respBody, &pipelineInfo)
	if err != nil {
		return nil, fmt.Errorf("get pipelines %s unmarshal error: %s", c.pipelineUrl, err.Error())
	}

	return pipelineInfo, nil
}
