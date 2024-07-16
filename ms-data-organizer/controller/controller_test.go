/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package controller

import (
	"aicsd/pkg/wait"
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/logger"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	taskLauncherMocks "aicsd/ms-data-organizer/clients/task_launcher/mocks"
	"aicsd/pkg"
	fileSenderMocks "aicsd/pkg/clients/job_handler/mocks"
	jobRepoMocks "aicsd/pkg/clients/job_repo/mocks"
	"aicsd/pkg/helpers"
	"aicsd/pkg/types"
)

const fileHostname = "oem"

var dependentServices = wait.Services{wait.ServiceConsul, wait.ServiceJobRepo}

func TestDataOrgController_RetryOnStartup(t *testing.T) {
	expected := []types.Job{helpers.CreateTestJob(pkg.OwnerDataOrg, fileHostname)}
	emptyJobs := []types.Job{}
	tests := []struct {
		Name                   string
		Jobs                   *[]types.Job
		RepoMockRetrieveError  error
		LauncherMatchTaskBool  bool
		LauncherMatchTaskError error
		RepoUpdateError        error
		SenderMockError        error
		ExpectedError          error
	}{
		{"happy path", &expected, nil, true,
			nil, nil, nil, nil},
		{"empty jobs", &emptyJobs, pkg.ErrRetrieving, true,
			nil, nil, nil, pkg.ErrRetrieving},
		{"no matching tasks", &expected, nil, false,
			nil, nil, nil, nil},
		{"matchTask call failed", &expected, nil, false,
			errors.New("matchTask failed"), nil, nil, errors.New("call failed for matchTasks on file (test-image.tiff): matchTask failed")},
		{"update failed", &expected, nil, false,
			nil, pkg.ErrUpdating, nil, pkg.ErrUpdating},
		{"sender failed", &expected, nil, true,
			nil, nil, pkg.ErrHandleJob, pkg.ErrHandleJob},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			repoMock := jobRepoMocks.Client{}
			taskLaunchMock := taskLauncherMocks.Client{}
			senderMock := fileSenderMocks.Client{}
			attributeParser := make(map[string]types.AttributeInfo)
			fileHandler := New(logger.MockLogger{}, &repoMock, &senderMock, &taskLaunchMock, attributeParser, dependentServices)

			repoMock.On("RetrieveAllByOwner", pkg.OwnerDataOrg).Return(*test.Jobs, test.RepoMockRetrieveError)
			taskLaunchMock.On("MatchTask", mock.Anything).Return(test.LauncherMatchTaskBool, test.LauncherMatchTaskError)
			if !test.LauncherMatchTaskBool {
				repoMock.On("Update", mock.Anything, mock.Anything).Return(types.Job{}, test.RepoUpdateError)
			}
			senderMock.On("HandleJob", mock.Anything).Return(test.SenderMockError)

			err := fileHandler.RetryOnStartup()

			if test.ExpectedError != nil {
				require.Contains(t, err.Error(), test.ExpectedError.Error())
			}
		})
	}
}

func TestDataOrgController_NotifyNewFileHandler(t *testing.T) {
	expectedId := "1"
	expected := helpers.CreateTestJob(pkg.OwnerDataOrg, fileHostname)
	var err error
	badFile := expected
	badFile.Owner = "none"
	fileExistsJob := helpers.CreateTestJob(pkg.OwnerTaskLauncher, fileHostname)

	tests := []struct {
		Name                   string
		LocalData              *types.Job
		RepoCreateId           string
		RepoCreateIsNew        bool
		RepoCreateError        error
		RepoRetByIdEntry       types.Job
		RepoRetByIdError       error
		LauncherMatchTaskBool  bool
		LauncherMatchTaskError error
		RepoUpdateError        error
		SenderMockError        error
		ExpectedStatusCode     int
		ExpectError            bool
	}{
		{"happy path new id", &expected, expectedId, true, nil,
			types.Job{}, nil, true, nil, nil,
			nil, http.StatusOK, false},
		{"happy path existing id data org owned", &expected, expectedId, false, nil,
			expected, nil, true, nil, nil,
			nil, http.StatusOK, false},
		{"happy path existing id task launcher owned", &expected, expectedId, false, nil,
			fileExistsJob, nil, true, nil, nil,
			nil, http.StatusAlreadyReported, false},
		{"unmarshall failure", nil, expectedId, true, nil,
			expected, nil, true, nil, nil,
			nil, http.StatusInternalServerError, true},
		{"repo create error", &expected, expectedId, true, pkg.ErrJobCreation,
			expected, nil, true, nil, nil,
			nil, http.StatusBadRequest, true},
		{"repo retrieve error", &expected, expectedId, false, nil,
			types.Job{}, pkg.ErrRetrieving, true, nil, nil,
			nil, http.StatusBadRequest, true},
		{"repo no matching tasks", &expected, expectedId, false, nil,
			expected, nil, false, nil, nil,
			nil, http.StatusNoContent, false},
		{"repo match task call failed", &expected, expectedId, false, nil,
			expected, nil, false, errors.New("match task call failed"), nil,
			nil, http.StatusBadRequest, true},
		{"repo fail update", &expected, expectedId, false, nil,
			expected, nil, false, nil, pkg.ErrUpdating,
			nil, http.StatusInternalServerError, true},
		{"repo sender error", &expected, expectedId, false, nil,
			expected, nil, true, nil, nil,
			errors.New("sender error"), http.StatusInternalServerError, true},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			repoMock := jobRepoMocks.Client{}
			senderMock := fileSenderMocks.Client{}
			taskLaunchMock := taskLauncherMocks.Client{}
			attributeParser := make(map[string]types.AttributeInfo)
			dataOrgController := New(logger.MockLogger{}, &repoMock, &senderMock, &taskLaunchMock, attributeParser, dependentServices)
			var requestBody []byte
			if test.LocalData != nil {
				requestBody, err = json.Marshal(expected)
				require.NoError(t, err)
			} else {
				requestBody = nil
			}

			req := httptest.NewRequest("POST", "http://localhost", bytes.NewReader(requestBody))
			w := httptest.NewRecorder()

			repoMock.On("Create", mock.Anything).Return(test.RepoCreateId, test.RepoCreateIsNew, test.RepoCreateError)
			repoMock.On("RetrieveById", mock.Anything).Return(test.RepoRetByIdEntry, test.RepoRetByIdError)
			taskLaunchMock.On("MatchTask", mock.Anything).Return(test.LauncherMatchTaskBool, test.LauncherMatchTaskError)
			if !test.LauncherMatchTaskBool && test.LauncherMatchTaskError == nil {
				repoMock.On("Update", mock.Anything, mock.Anything).Return(test.RepoRetByIdEntry, test.RepoUpdateError)
			}
			senderMock.On("HandleJob", mock.Anything).Return(test.SenderMockError)

			dataOrgController.NotifyNewFileHandler(w, req)
			resp := w.Result()
			defer resp.Body.Close()

			require.Equal(t, test.ExpectedStatusCode, resp.StatusCode)

			if !test.RepoCreateIsNew {
				repoMock.AssertCalled(t, "RetrieveById", mock.Anything)
			}
			if !test.LauncherMatchTaskBool && test.LauncherMatchTaskError == nil {
				repoMock.AssertCalled(t, "Update", mock.Anything, mock.Anything)
			}
		})
	}
}
func FuzzNotifyNewFileHandler(f *testing.F) {

	expectedId := "1"
	expected := helpers.CreateTestJob(pkg.OwnerDataOrg, fileHostname)
	expectedJson, err := json.Marshal(expected)
	require.NoError(f, err)

	badFile := expected
	badFile.Owner = "none"
	badFileJson, err := json.Marshal(badFile)
	require.NoError(f, err)

	fileExistsJob := helpers.CreateTestJob(pkg.OwnerTaskLauncher, fileHostname)
	fileExistsJobJson, err := json.Marshal(fileExistsJob)
	require.NoError(f, err)

	// Seed corpus with a valid JSON job representation if available
	f.Add(expectedJson)
	f.Add(badFileJson)
	f.Add(fileExistsJobJson)

	f.Fuzz(func(t *testing.T, data []byte) {
		// Mock dependencies
		repoMock := jobRepoMocks.Client{}
		senderMock := fileSenderMocks.Client{}
		taskLaunchMock := taskLauncherMocks.Client{}
		attributeParser := make(map[string]types.AttributeInfo)
		dependentServices := wait.Services{wait.ServiceConsul, wait.ServiceJobRepo}

		// Create a new instance of the controller
		controller := New(logger.MockLogger{}, &repoMock, &senderMock, &taskLaunchMock, attributeParser, dependentServices)

		requestBody := data
		// Create a new HTTP request with the fuzzed JSON data
		req := httptest.NewRequest("POST", "http://localhost", bytes.NewReader(requestBody))
		// Create a ResponseRecorder to record the response
		w := httptest.NewRecorder()

		// Set up mocks for dependencies if necessary
		repoMock.On("Create", mock.Anything).Return(expectedId, true, nil)

		repoMock.On("RetrieveById", mock.Anything).Return(mock.Anything, nil)
		taskLaunchMock.On("MatchTask", mock.Anything).Return(true, nil)

		repoMock.On("Update", mock.Anything, mock.Anything).Return(nil, nil)

		senderMock.On("HandleJob", mock.Anything).Return(nil)

		// Call the handler function
		controller.NotifyNewFileHandler(w, req)

		// Check if the response is as expected
		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("Handler returned unexpected status code: %v", resp.StatusCode)
		}
	})
}
