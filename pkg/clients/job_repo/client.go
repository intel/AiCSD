/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package job_repo

import (
	"aicsd/pkg/auth"
	"aicsd/pkg/werrors"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"aicsd/pkg"
	"aicsd/pkg/types"
)

type RepoClient struct {
	baseUrl     string
	jobUrl      string
	byOwnerUrl  string
	httpTimeout time.Duration
	jwtInfo     *auth.JWTInfo
}

func NewClient(baseUrl string, httpTimeout time.Duration, info *auth.JWTInfo) Client {
	client := RepoClient{
		baseUrl:     baseUrl,
		jobUrl:      fmt.Sprintf("%s%s", baseUrl, pkg.EndpointJob),
		byOwnerUrl:  fmt.Sprintf("%s%s/%s", baseUrl, pkg.EndpointJob, pkg.OwnerKey),
		httpTimeout: httpTimeout,
		jwtInfo:     info,
	}
	return &client
}

// Create is used to add a job to the job repo.
// It returns the id as a string; IsNew bool indicating
// if the id is new (true) or already existed (false); and
// error if owner does not match what was sent.
func (c *RepoClient) Create(job types.Job) (string, bool, error) {
	var id string
	isNew := true // set default, assume new
	body, err := json.Marshal(job)
	if err != nil {
		return id, isNew, fmt.Errorf("job repo create job marshal error: %s", err.Error())
	}

	req, err := http.NewRequest(http.MethodPost, c.jobUrl, bytes.NewBuffer(body))
	if err != nil {
		return id, isNew, fmt.Errorf("could not create request to handle job: %s", err.Error())
	}
	err = c.jwtInfo.AddAuthHeader(req)
	if err != nil {
		return id, isNew, werrors.WrapErr(err, pkg.ErrAuthHeader)
	}
	client := &http.Client{
		Timeout: c.httpTimeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		return id, isNew, fmt.Errorf("job repo create job post error: %s", err.Error())
	}
	defer resp.Body.Close()
	//    -- response code: 201 if created + id is returned, update local copy
	//    --  				409 if duplicate job + id is returned
	// 	  -- 				40X handle other errors
	if resp.StatusCode == http.StatusConflict {
		isNew = false
	} else if resp.StatusCode != http.StatusCreated {
		return id, isNew, fmt.Errorf("job repo create job status not expected: %s", resp.Status)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return id, isNew, fmt.Errorf("job repo create job read error: %s", err.Error())
	}
	id = string(respBody)

	return id, isNew, nil
}

// RetrieveAll is used to query for all jobs.
// It returns jobs which match the owner and error if http request fails.
func (c *RepoClient) RetrieveAll(headers map[string]string) ([]types.Job, error) {
	req, err := http.NewRequest(http.MethodGet, c.jobUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("job repo retrieve all new request error: %s", err.Error())
	}

	client := &http.Client{
		Timeout: c.httpTimeout,
	}

	// Note: Set appropriate headers as needed
	// This was added to allow for headers to support internationalization.
	if headers != nil {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}
	err = c.jwtInfo.AddAuthHeader(req)
	if err != nil {
		return nil, werrors.WrapErr(err, pkg.ErrAuthHeader)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("job repo retrieve all do request error: %s", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("job repo retrieve all by owner status not OK: %s", resp.Status)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("job repo retrieve all by owner read error: %s", err.Error())
	}

	var jobs []types.Job
	err = json.Unmarshal(respBody, &jobs)
	if err != nil {
		return nil, fmt.Errorf("job repo retrieve all by owner unmarshal error: %s", err.Error())
	}

	return jobs, nil
}

// RetrieveAllByOwner is used to query for all jobs by owner.
// It returns jobs which match the owner and error if http request fails.
func (c *RepoClient) RetrieveAllByOwner(owner string) ([]types.Job, error) {
	ownerUrl := fmt.Sprintf("%s/%s", c.byOwnerUrl, owner)
	req, err := http.NewRequest(http.MethodGet, ownerUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("job repo retrieve all by owner new request error: %s", err.Error())
	}
	err = c.jwtInfo.AddAuthHeader(req)
	if err != nil {
		return nil, werrors.WrapErr(err, pkg.ErrAuthHeader)
	}
	client := &http.Client{
		Timeout: c.httpTimeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("job repo retrieve all by owner do request error: %s", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("job repo retrieve all by owner status not OK for owner %s: %s", owner, resp.Status)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("job repo retrieve all by owner read error: %s", err.Error())
	}

	var jobs []types.Job
	err = json.Unmarshal(respBody, &jobs)
	if err != nil {
		return nil, fmt.Errorf("job repo retrieve all by owner unmarshal error: %s", err.Error())
	}

	return jobs, nil
}

// RetrieveById is used to query for a job by id.
// It returns job which matches the id and error if http request fails.
func (c *RepoClient) RetrieveById(id string) (types.Job, error) {
	var job types.Job
	jobIdUrl := fmt.Sprintf("%s/%s", c.jobUrl, id)

	req, err := http.NewRequest(http.MethodGet, jobIdUrl, nil)
	if err != nil {
		return job, fmt.Errorf("could not create request to handle job: %s", err.Error())
	}
	err = c.jwtInfo.AddAuthHeader(req)
	if err != nil {
		return job, werrors.WrapErr(err, pkg.ErrAuthHeader)
	}
	client := &http.Client{
		Timeout: c.httpTimeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		return job, fmt.Errorf("job repo retrieve by id get error: %s", err.Error())
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return job, fmt.Errorf("job repo retrieve job status not OK for id %s: %s", id, resp.Status)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return job, fmt.Errorf("job repo retrieve by id read error: %s", err.Error())
	}

	err = json.Unmarshal(respBody, &job)
	if err != nil {
		return job, fmt.Errorf("job repo retrieve by id unmarshal error: %s", err.Error())
	}

	return job, nil
}

// Update is used to put updated values for a job.
// It returns an error if the http request fails.
func (c *RepoClient) Update(id string, jobFields map[string]interface{}) (types.Job, error) {
	var job types.Job
	jobIdUrl := fmt.Sprintf("%s/%s", c.jobUrl, id)

	body, err := json.Marshal(jobFields)
	if err != nil {
		return job, fmt.Errorf("job repo update job marshal error: %s", err.Error())
	}
	req, err := http.NewRequest(http.MethodPut, jobIdUrl, bytes.NewBuffer(body))
	if err != nil {
		return job, fmt.Errorf("job repo update job new request error: %s", err.Error())
	}

	err = c.jwtInfo.AddAuthHeader(req)
	if err != nil {
		return job, werrors.WrapErr(err, pkg.ErrAuthHeader)
	}
	client := &http.Client{
		Timeout: c.httpTimeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		return job, fmt.Errorf("job repo update job do request error: %s", err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return job, fmt.Errorf("job repo update job status not OK for id %s: %s", id, resp.Status)
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return job, fmt.Errorf("job repo update read error: %s", err.Error())
	}

	err = json.Unmarshal(respBody, &job)
	if err != nil {
		return job, fmt.Errorf("job repo update unmarshal error: %s", err.Error())
	}

	return job, nil
}

// Delete is used to remove a job by id.
// It returns an error if the http request fails.
func (c *RepoClient) Delete(id string) error {
	jobIdUrl := fmt.Sprintf("%s/%s", c.jobUrl, id)
	req, err := http.NewRequest(http.MethodDelete, jobIdUrl, nil)
	if err != nil {
		return fmt.Errorf("job repo delete job new request error: %s", err.Error())
	}
	err = c.jwtInfo.AddAuthHeader(req)
	if err != nil {
		return werrors.WrapErr(err, pkg.ErrAuthHeader)
	}
	client := &http.Client{
		Timeout: c.httpTimeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("job repo delete job do request error: %s", err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("job repo delete job status not OK for id %s: %s", id, resp.Status)
	}
	return nil
}
