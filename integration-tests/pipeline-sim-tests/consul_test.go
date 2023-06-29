/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/
package pipeline_sim_tests

import (
	"aicsd/integration-tests/pkg"
	"bytes"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/require"
)

func TestServicesOnConsul(t *testing.T) {

	serviceNames := PipelineSimFactory.GetApplicationServiceNames()

	token := pkg.GetConsulACLToken(t)
	require.Eventually(t, func() bool {
		e := httpexpect.Default(t, pkg.ConsulUrl)
		obj := e.GET("/v1/agent/services").
			WithHeader(pkg.ConsulHeaderKey, fmt.Sprintf(pkg.ConsulTokenFmt, token)).
			WithMaxRetries(pkg.MaxRetries).
			Expect().
			Status(http.StatusOK).JSON().Object()

		if len(obj.Raw()) != len(serviceNames) {
			fmt.Printf("Expecting %d services but recieved %d\n", len(serviceNames), len(obj.Raw()))
			return false
		}

		for _, service := range serviceNames {
			if _, ok := obj.Raw()[service]; !ok {
				fmt.Printf("Service %s not found in Consul\n", service)
				return false
			}
		}

		return true
	}, pkg.ContainerWait, time.Second)
}

func TestLogLevelChange(t *testing.T) {
	logLevel := "Test"
	token := pkg.GetConsulACLToken(t)
	for _, service := range PipelineSimFactory.GetDockerServiceNames() {

		request, err := http.NewRequest(http.MethodPut, fmt.Sprintf(pkg.ConsulChangeLogLevelUrl, service), bytes.NewBuffer([]byte(logLevel)))
		require.NoError(t, err)
		request.Header.Set(pkg.ConsulHeaderKey, fmt.Sprintf(pkg.ConsulTokenFmt, token))

		testclient := pkg.NewTestClient()
		response, err := testclient.Do(request)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, response.StatusCode)
	}

}
