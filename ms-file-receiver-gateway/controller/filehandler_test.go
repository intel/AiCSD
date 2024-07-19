/*********************************************************************
 * Copyright (c) Intel Corporation 2024
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"aicsd/ms-file-receiver-gateway/config"
	"aicsd/pkg"
	jobHandlerMocks "aicsd/pkg/clients/job_handler/mocks"
	jobRepoMocks "aicsd/pkg/clients/job_repo/mocks"
	"aicsd/pkg/helpers"
	"aicsd/pkg/types"

	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const fileHostname = "gateway"

func TestFileHandler_RetryOnStartup(t *testing.T) {
	configuration := config.Configuration{
		BaseFileFolder: ".",
		FileHostname:   fileHostname,
	}
	testJobs := []types.Job{helpers.CreateTestJob(pkg.OwnerFileRecvGateway, fileHostname),
		helpers.CreateTestJob(pkg.OwnerFileRecvGateway, fileHostname)}
	tests := []struct {
		Name                      string
		Jobs                      *[]types.Job
		Hostname                  string
		RepoMockRetrieveReturn    error
		TaskLauncherMockDTHReturn error
		ExpectedError             error
	}{
		{"happy path", &testJobs, fileHostname, nil, nil, nil},
		{"retrieve failed", &testJobs, fileHostname, errors.New("retrieve failed"), nil,
			fmt.Errorf("could not retrieve %s data: retrieve failed", pkg.OwnerFileRecvGateway)},
		{"fileHostname mismatch", &testJobs, "", nil, nil, fmt.Errorf("1 error occurred:\n\t* fileHostname does not match: got %s, expected %s\n\n", "oem", fileHostname)},
		{"DTH failed", &testJobs, fileHostname, nil, errors.New("call failed"), errors.New("HandleJob call failed: call failed")},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			repoMock := jobRepoMocks.Client{}
			taskLaunchMock := jobHandlerMocks.Client{}
			fileHandler := New(logger.MockLogger{}, &repoMock, &taskLaunchMock, &configuration)
			if len(test.Hostname) == 0 {
				(*test.Jobs)[0].InputFile.Hostname = "oem"
			}
			repoMock.On("RetrieveAllByOwner", pkg.OwnerFileRecvGateway).Return(*test.Jobs, test.RepoMockRetrieveReturn)
			taskLaunchMock.On("HandleJob", mock.Anything).Return(test.TaskLauncherMockDTHReturn)
			err := fileHandler.RetryOnStartup()

			if test.ExpectedError == nil {
				require.Nil(t, err)
			} else {
				require.Contains(t, err.Error(), test.ExpectedError.Error())
			}
		})
	}

}

func TestFileHandler_TransmitJob(t *testing.T) {
	configuration := config.Configuration{
		BaseFileFolder: ".",
		FileHostname:   fileHostname,
	}
	fileHandler := New(logger.MockLogger{}, nil, nil, &configuration)
	expected := helpers.CreateTestJob(pkg.OwnerFileRecvGateway, fileHostname)
	badId := expected
	badId.Id = ""
	tests := []struct {
		Name               string
		LocalData          *types.Job
		ExpectedStatusCode int
		ExpectError        bool
	}{
		{"happy path", &expected, http.StatusOK, false},
		{"no job", nil, http.StatusBadRequest, true},
		{"bad id", &badId, http.StatusBadRequest, true},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var requestBody []byte
			var err error
			if test.LocalData != nil {
				requestBody, err = json.Marshal(test.LocalData)
				require.NoError(t, err)
			} else {
				requestBody = nil
			}
			req := httptest.NewRequest("POST", "http://localhost", bytes.NewReader(requestBody))
			w := httptest.NewRecorder()
			fileHandler.TransmitJob(w, req)
			resp := w.Result()
			defer func() { _ = resp.Body.Close() }()

			require.Equal(t, test.ExpectedStatusCode, resp.StatusCode, "invalid status code")
			if test.ExpectError {
				return
			}
			require.NotNil(t, fileHandler.jobMap[test.LocalData.Id])
			assert.Equal(t, expected, fileHandler.jobMap[test.LocalData.Id])
		})
	}
}

// FuzzTransmitJob tests the TransmitJob method with random JSON data.
func FuzzTransmitJob(f *testing.F) {
	// Seed corpus entries with known inputs to start with.
	// These should be valid JSON representations of types.Job objects.
	expected := helpers.CreateTestJob(pkg.OwnerFileRecvGateway, fileHostname)
	val, err := json.Marshal(&expected)
	require.NoError(f, err)

	badJob1_obj := expected
	badJob1_obj.Id = ""
	badJob1_json, err := json.Marshal(&badJob1_obj)
	require.NoError(f, err)

	badJob2_obj := expected
	badJob2_obj.InputFile.Name = "@#%!&*()"
	badJob2_json, err := json.Marshal(&badJob2_obj)
	require.NoError(f, err)

	f.Add(string(val))          // known good input
	f.Add(string(badJob1_json)) // known bad input
	f.Add(string(badJob2_json)) // known edge case input

	// Setup mocks and dependencies as needed.
	configuration := config.Configuration{
		BaseFileFolder: ".",
		FileHostname:   fileHostname,
	}
	fileHandler := New(logger.MockLogger{}, nil, nil, &configuration)

	// Define the fuzzing function.
	f.Fuzz(func(t *testing.T, data string) {
		requestBody := []byte(data)

		req := httptest.NewRequest("POST", "http://localhost", bytes.NewReader(requestBody))
		w := httptest.NewRecorder()
		fileHandler.TransmitJob(w, req)
		resp := w.Result()
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest {
			t.Errorf("unexpected status code: got %v, want %v or %v", resp.StatusCode, http.StatusOK, http.StatusBadRequest)
		}
	})
}

func TestFileHandler_TransmitFile(t *testing.T) {
	configuration := config.Configuration{
		BaseFileFolder: ".",
		FileHostname:   fileHostname,
	}
	expected := types.Job{
		Id:    "1",
		Owner: "Input File Sender",
		InputFile: types.FileInfo{
			Hostname:  "oemsys1",
			DirName:   filepath.Join(".", "test"),
			Name:      "test-image.tiff",
			Extension: "tiff",
		},
		LastUpdated: time.Now().UTC().UnixNano(),
	}
	expectedNested := expected
	expectedNested.InputFile.Name = "test-nested-file.tiff"
	expectedNested.InputFile.DirName = filepath.Join(expected.InputFile.DirName, "input", "test", "u1")
	badWinPath := expectedNested
	badWinPath.InputFile.DirName = "C:\\Users\\test\\data"

	tests := []struct {
		Name                      string
		Filename                  string
		Id                        string
		Job                       *types.Job
		RequestBody               []byte
		RepoMockUpdateReturn      error
		TaskLauncherMockDTHReturn error
		ExpectedStatusCode        int
		ExpectError               bool
		ExpectedErrorMsg          string
	}{
		{"happy path", expected.InputFile.Name, expected.Id, &expected, []byte("body"), nil, nil, http.StatusOK, false, ""},
		{"happy nested path", expectedNested.InputFile.Name, expectedNested.Id, &expectedNested, []byte("body"), nil, nil, http.StatusOK, false, ""},
		{"missing windows keyword path", badWinPath.InputFile.Name, badWinPath.Id, &badWinPath, []byte("body"), nil, nil, http.StatusInternalServerError, true, "expected valid Windows filepath keyword \"\\oem-files\\\", got"},
		{"bad id", expected.InputFile.Name, mock.Anything, &expected, []byte("body"), nil, nil, http.StatusInternalServerError, true, "did not receive job mapping to id"},
		{"wrong filename", "image2.tiff", expected.Id, &expected, []byte("body"), nil, nil, http.StatusInternalServerError, true, "received job input file does not match"},
		{"job repo update failed", expected.InputFile.Name, expected.Id, &expected, []byte("body"), errors.New("Update Failed"), nil, http.StatusInternalServerError, true, "job repo update failed"},
		{"task launcher DTH failed", expected.InputFile.Name, expected.Id, &expected, []byte("body"), nil, errors.New("404: Not Found"), http.StatusOK, false, ""},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			repoMock := jobRepoMocks.Client{}
			taskLaunchMock := jobHandlerMocks.Client{}
			fileHandler := New(logger.MockLogger{}, &repoMock, &taskLaunchMock, &configuration)
			fileHandler.jobMap[expected.Id] = *test.Job
			req := httptest.NewRequest("POST", "http://localhost", bytes.NewReader(test.RequestBody))
			req.Header.Add(pkg.FilenameKey, test.Filename)
			req.Header.Add(pkg.JobIdKey, test.Id)
			w := httptest.NewRecorder()
			// set the actual expected value into the update
			repoMock.On("Update", mock.Anything, mock.Anything).Return(types.Job{}, test.RepoMockUpdateReturn)
			taskLaunchMock.On("HandleJob", mock.Anything).Return(test.TaskLauncherMockDTHReturn)
			fileHandler.TransmitFile(w, req)
			resp := w.Result()
			defer func() { _ = resp.Body.Close() }()

			require.Equal(t, test.ExpectedStatusCode, resp.StatusCode, "invalid status code")
			if test.ExpectError {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				require.Contains(t, string(body), test.ExpectedErrorMsg)
				return
			}
			// check that job was removed from the map
			_, ok := fileHandler.jobMap[test.Id]
			require.False(t, ok)
			taskLaunchMock.AssertCalled(t, "HandleJob", mock.Anything)
			// clean up the file
			_ = os.Remove(filepath.Join(".", "test-image.tiff"))
			// clean up nested folder-structure
			_ = os.RemoveAll(filepath.Join(".", "test", "u1"))
			// clean up test-image
			_ = os.WriteFile("./test/test-image.tiff", []byte{}, pkg.FilePermissions)
		})
	}
}
