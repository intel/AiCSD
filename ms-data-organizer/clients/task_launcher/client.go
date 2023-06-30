/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package task_launcher

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
	"strconv"
	"time"
)

type TaskLauncherClient struct {
	baseUrl     string
	httpTimeout time.Duration
	jwtInfo     *auth.JWTInfo
}

func NewClient(baseUrl string, httpTimeout time.Duration, info *auth.JWTInfo) Client {
	client := TaskLauncherClient{
		baseUrl:     baseUrl,
		httpTimeout: httpTimeout,
		jwtInfo:     info,
	}
	return &client
}

func (c *TaskLauncherClient) MatchTask(job types.Job) (bool, error) {
	matchTaskUrl := fmt.Sprintf("%s%s", c.baseUrl, pkg.EndpointMatchTask)
	body, err := json.Marshal(job)
	if err != nil {
		return false, fmt.Errorf("could not marshall job: %s", err.Error())
	}
	req, err := http.NewRequest(http.MethodPost, matchTaskUrl, bytes.NewBuffer(body))
	if err != nil {
		return false, werrors.WrapMsg(err, "could not create request to handle job")
	}
	err = c.jwtInfo.AddAuthHeader(req)
	if err != nil {
		return false, werrors.WrapErr(err, pkg.ErrAuthHeader)
	}
	client := &http.Client{
		Timeout: c.httpTimeout,
	}
	response, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("could not make request to match task: %s", err.Error())
	}
	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		return false, fmt.Errorf("could not read response body: %s", err.Error())
	}
	if response.StatusCode != http.StatusOK {
		return false, fmt.Errorf("match task returned not ok status %s: %s", response.Status, respBody)
	}
	isMatch, err := strconv.ParseBool(string(bytes.TrimSpace(respBody)))
	if err != nil {
		return false, fmt.Errorf("could not parse bool from match task got: %s", respBody)
	}
	return isMatch, nil
}

// RetrieveById is used to query for a task by id.
// It returns a task which matches the id and an error if http request fails.
func (c *TaskLauncherClient) RetrieveById(id string) (types.Task, error) {
	var task types.Task
	taskIdUrl := fmt.Sprintf("%s/api/v1/task/%s", c.baseUrl, id)

	resp, err := http.Get(taskIdUrl)
	if err != nil {
		return task, fmt.Errorf("task launcher retrieve by id get error: %s", err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return task, fmt.Errorf("task launcher retrieve task status not OK for id %s: %s", id, resp.Status)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return task, fmt.Errorf("task launcher retrieve by id read error: %s", err.Error())
	}

	err = json.Unmarshal(respBody, &task)
	if err != nil {
		return task, fmt.Errorf("task launcher retrieve by id unmarshal error: %s", err.Error())
	}

	return task, nil
}
