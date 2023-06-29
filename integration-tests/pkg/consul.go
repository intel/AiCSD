/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package pkg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// consulAPIResponse is meant to capture the Consul Response for /v1/agent/services.
// This is necessary to ensure our Consul test dependencies are set up and ready to go for integration testing.
type consulAPIResponse struct {
	AsFileReceiverOem     []interface{} `json:"app-file-receiver-oem"`
	AsFileSenderGateway   []interface{} `json:"app-file-sender-gateway"`
	AsPipelineSim         []interface{} `json:"app-pipeline-sim"`
	AsTaskLauncher        []interface{} `json:"app-task-launcher"`
	Consul                []interface{} `json:"consul"`
	MsDataOrganizer       []interface{} `json:"app-data-organizer"`
	MsFileReceiverGateway []interface{} `json:"app-file-receiver-gateway"`
	MsFileSenderOem       []interface{} `json:"app-file-sender-oem"`
	MsFileWatcher         []interface{} `json:"app-file-watcher"`
	MsJobRepository       []interface{} `json:"app-job-repository"`
}

// checkResponse is the helper that checks the consul response contains all services.
func checkResponse(services consulAPIResponse) bool {
	if services.AsFileReceiverOem == nil ||
		services.AsFileSenderGateway == nil ||
		services.AsPipelineSim == nil ||
		services.AsTaskLauncher == nil ||
		services.Consul == nil ||
		services.MsDataOrganizer == nil ||
		services.MsFileReceiverGateway == nil ||
		services.MsFileSenderOem == nil ||
		services.MsFileWatcher == nil ||
		services.MsJobRepository == nil {
		return false
	}
	return true
}

// consulResponse is a helper func that verifies all services are registered with consul.
// This is used as part of the Consul wait.Strategy for the service setup.
func ConsulResponse(body io.Reader) bool {
	data, err := io.ReadAll(body)
	if err != nil {
		fmt.Printf("data %s", data)
		return false
	}
	var services consulAPIResponse
	err = json.Unmarshal(data, &services)
	if err != nil {
		fmt.Printf("data %s response %s", data, services)

		return false
	}
	return checkResponse(services)
}

// ChangeConsulKeyValue will create a request to change a key value stored in Consul
func ChangeConsulKeyValue(t *testing.T, url string, value string, expectedResponse bool) {
	t.Helper()
	request, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer([]byte(value)))
	require.NoError(t, err)
	token := GetConsulACLToken(t)
	request.Header.Set(ConsulHeaderKey, fmt.Sprintf(ConsulTokenFmt, token))
	testclient := NewTestClient()

	response, err := testclient.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, http.StatusOK, response.StatusCode)
	responseBody, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	var body bool
	err = json.Unmarshal(responseBody, &body)
	require.NoError(t, err)
	require.Equal(t, expectedResponse, body)
}

func GetConsulACLToken(t *testing.T) string {
	t.Helper()
	cmdStr := "docker container ls | grep consul | awk '{ print $1 }'"
	out, err := exec.Command("/bin/sh", "-c", cmdStr).Output()
	require.NoError(t, err)
	containerId := strings.TrimSpace(string(out))
	cmdStr = "docker exec -i " + string(containerId) + " /bin/sh -c 'cat /tmp/edgex/secrets/consul-acl-token/mgmt_token.json | jq -r '.SecretID''"
	out, err = exec.Command("/bin/sh", "-c", cmdStr).Output()
	require.NoError(t, err)
	return strings.TrimSpace(string(out))
}
