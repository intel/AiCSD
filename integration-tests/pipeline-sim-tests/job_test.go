/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/
package pipeline_sim_tests

import (
	integrationtests "aicsd/integration-tests/pkg"
	"aicsd/pkg"
	"aicsd/pkg/helpers"
	"net/http"
	"testing"

	"github.com/gavv/httpexpect/v2"
)

func TestCRUDJobRepo(t *testing.T) {
	jobField := helpers.CreateTestJob("string", "string")
	jobField.Id = ""
	e := httpexpect.Default(t, integrationtests.JobRepositoryUrl)

	jobID := e.POST(pkg.EndpointJob).WithJSON(jobField).
		WithMaxRetries(integrationtests.MaxRetries).
		WithRetryPolicy(httpexpect.RetryAllErrors).
		Expect().
		Status(http.StatusCreated)

	// Duplicate post
	e.POST(pkg.EndpointJob).WithJSON(jobField).
		WithMaxRetries(integrationtests.MaxRetries).
		WithRetryPolicy(httpexpect.RetryAllErrors).
		Expect().
		Status(http.StatusConflict)

	e.GET(pkg.EndpointJob).
		WithMaxRetries(integrationtests.MaxRetries).
		WithRetryPolicy(httpexpect.RetryAllErrors).
		Expect().
		Status(http.StatusOK)

	jobField.Owner = "New Owner"

	e.PUT(pkg.EndpointJob + "/" + jobID.Text().Raw()).WithJSON(jobField).
		WithMaxRetries(integrationtests.MaxRetries).
		WithRetryPolicy(httpexpect.RetryAllErrors).
		Expect().
		Status(http.StatusOK)

	e.DELETE(pkg.EndpointJob + "/" + jobID.Text().Raw()).
		WithMaxRetries(integrationtests.MaxRetries).
		WithRetryPolicy(httpexpect.RetryAllErrors).
		Expect().
		Status(http.StatusOK)

	// Duplicate delete
	e.DELETE(pkg.EndpointJob + "/" + jobID.Text().Raw()).
		Expect().
		Status(http.StatusInternalServerError)
}
