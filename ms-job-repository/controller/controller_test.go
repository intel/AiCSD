/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package controller

import (
	"aicsd/pkg/translation"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	"aicsd/ms-job-repository/persist"
	persistMocks "aicsd/ms-job-repository/persist/mocks"
	"aicsd/pkg"
	"aicsd/pkg/helpers"
	"aicsd/pkg/types"

	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/logger"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const fileHostname = "oem"

var (
	localizationFiles = []string{"../../pkg/translation/dictionary/en.json", "../../pkg/translation/dictionary/zh.json"}
)

func TestJobRepoController_Create(t *testing.T) {
	expected := helpers.CreateTestJob(pkg.OwnerDataOrg, fileHostname)
	expectedWithId := expected
	expectedWithId.Id = "1"

	tests := []struct {
		Name               string // test name
		Expected           *types.Job
		PersistMockStatus  string
		PersistJob         types.Job // consider using a pointer? (nil)
		PersistMockErr     error
		ExpectedStatusCode int
		ExpectedErrorMsg   string
	}{
		{"happy path: new", &expected, persist.StatusCreated, expectedWithId, nil, http.StatusCreated, ""},
		{"happy path: exists", &expected, persist.StatusExists, expectedWithId, nil, http.StatusConflict, ""},
		{"unmarshal failed", nil, "", types.Job{}, nil, http.StatusBadRequest, "unexpected end of JSON input"},
		{"persist create failed", &types.Job{}, "", types.Job{}, errors.New("persist create failed"), http.StatusBadRequest, "failed to create job"},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var requestBody []byte
			persistMock := persistMocks.Persistence{}
			testLocalizationBundle, err := translation.NewBundler(localizationFiles)
			require.NoError(t, err)
			jobRepoController := New(logger.MockLogger{}, &persistMock, testLocalizationBundle)

			if test.Expected != nil {
				requestBody, _ = json.Marshal(test.Expected)
			} else {
				requestBody = nil
			}

			persistMock.On("Create", mock.Anything).Return(test.PersistMockStatus, test.PersistJob, test.PersistMockErr)

			req := httptest.NewRequest("POST", "http://localhost", bytes.NewReader(requestBody))
			w := httptest.NewRecorder()

			jobRepoController.Create(w, req)
			resp := w.Result()
			defer resp.Body.Close()

			require.Equal(t, test.ExpectedStatusCode, resp.StatusCode, "invalid status code")
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			if (test.ExpectedStatusCode != http.StatusCreated) && (test.ExpectedStatusCode != http.StatusConflict) {
				require.Contains(t, string(body), test.ExpectedErrorMsg)
				return
			}
			require.NotEmpty(t, body)
			persistMock.AssertCalled(t, "Create", mock.Anything)

		})
	}
}

func TestJobRepoController_GetAll(t *testing.T) {
	job := []types.Job{helpers.CreateTestJob(pkg.OwnerDataOrg, fileHostname)}
	jobs := []types.Job{helpers.CreateTestJob(pkg.OwnerDataOrg, fileHostname),
		helpers.CreateTestJob(pkg.OwnerFileSenderOem, fileHostname)}
	jobs[1].Id = "2"
	jobs[1].InputFile.Name = "test-image2.tiff"
	tests := []struct {
		Name               string
		PersistMockJobs    []types.Job
		PersistMockErr     error
		ExpectedStatusCode int
		ExpectedErrorMsg   string
	}{
		{"happy path - no jobs", []types.Job{}, nil, http.StatusOK, ""},
		{"happy path - single job", job, nil, http.StatusOK, ""},
		{"happy path - multiple jobs", jobs, nil, http.StatusOK, ""},
		{"error retrieving", nil, pkg.ErrRetrieving, http.StatusInternalServerError, pkg.ErrRetrieving.Error()},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			persistMock := persistMocks.Persistence{}
			testLocalizationBundle, err := translation.NewBundler(localizationFiles)
			require.NoError(t, err)
			jobRepoController := New(logger.MockLogger{}, &persistMock, testLocalizationBundle)

			persistMock.On("GetAll", mock.Anything).Return(test.PersistMockJobs, test.PersistMockErr)

			req := httptest.NewRequest("GET", "http://localhost", nil)
			w := httptest.NewRecorder()

			jobRepoController.GetAll(w, req)
			resp := w.Result()
			defer resp.Body.Close()

			require.Equal(t, test.ExpectedStatusCode, resp.StatusCode, "invalid status code")
			body, err := io.ReadAll(resp.Body)
			if test.ExpectedStatusCode != http.StatusOK {
				require.NoError(t, err)
				require.Contains(t, string(body), test.ExpectedErrorMsg)
				return
			}

			var jobs []types.Job
			err = json.Unmarshal(body, &jobs)
			require.NoError(t, err)
			assert.Equal(t, test.PersistMockJobs, jobs)
			persistMock.AssertCalled(t, "GetAll", mock.Anything)
		})
	}
}

func TestJobRepoController_GetById(t *testing.T) {
	expected := helpers.CreateTestJob(pkg.OwnerDataOrg, fileHostname)

	tests := []struct {
		Name               string
		Id                 string
		PersistMockJobs    types.Job
		PersistMockErr     error
		ExpectedStatusCode int
		ExpectedErrorMsg   string
	}{
		{"happy path", "1", expected, nil, http.StatusOK, ""},
		{"missing id", "nil", types.Job{}, nil, http.StatusBadRequest, "missing jobid in url"},
		{"empty id", "", types.Job{}, nil, http.StatusBadRequest, "empty jobid in url"},
		{"id not found", "bogus", types.Job{}, pkg.ErrRetrieving, http.StatusInternalServerError, pkg.ErrRetrieving.Error()},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			persistMock := persistMocks.Persistence{}
			testLocalizationBundle, err := translation.NewBundler(localizationFiles)
			require.NoError(t, err)
			jobRepoController := New(logger.MockLogger{}, &persistMock, testLocalizationBundle)

			persistMock.On("GetById", mock.Anything).Return(test.PersistMockJobs, test.PersistMockErr)

			req := httptest.NewRequest("GET", "http://localhost", nil)
			if test.Id != "nil" {
				req = mux.SetURLVars(req, map[string]string{pkg.JobIdKey: test.Id})
			}
			w := httptest.NewRecorder()

			jobRepoController.GetById(w, req)
			resp := w.Result()
			defer resp.Body.Close()

			require.Equal(t, test.ExpectedStatusCode, resp.StatusCode, "invalid status code")
			body, err := io.ReadAll(resp.Body)
			if test.ExpectedStatusCode != http.StatusOK {
				require.NoError(t, err)
				require.Contains(t, string(body), test.ExpectedErrorMsg)
				return
			}

			var entry types.Job
			err = json.Unmarshal(body, &entry)
			require.NoError(t, err)
			assert.Equal(t, test.PersistMockJobs, entry)
			persistMock.AssertCalled(t, "GetById", mock.Anything)
		})
	}
}

func TestJobRepoController_GetByOwner(t *testing.T) {
	job := []types.Job{helpers.CreateTestJob(pkg.OwnerDataOrg, fileHostname)}
	jobs := []types.Job{helpers.CreateTestJob(pkg.OwnerTaskLauncher, fileHostname),
		helpers.CreateTestJob(pkg.OwnerTaskLauncher, fileHostname)}
	jobs[1].Id = "2"
	jobs[1].InputFile.Name = "test-image2.tiff"
	tests := []struct {
		Name               string
		Owner              string
		PersistMockJobs    []types.Job
		PersistMockErr     error
		ExpectedStatusCode int
		ExpectedErrorMsg   string
	}{
		{"happy path - no job", pkg.OwnerFileWatcher, []types.Job{}, nil, http.StatusOK, ""},
		{"happy path - single job", job[0].Owner, job, nil, http.StatusOK, ""},
		{"happy path - multiple jobs", jobs[0].Owner, jobs, nil, http.StatusOK, ""},
		{"missing owner", "nil", nil, nil, http.StatusBadRequest, "missing owner in url"},
		{"empty owner", "", nil, nil, http.StatusBadRequest, "empty owner in url"},
		{"owner not found", "bogus", nil, pkg.ErrRetrieving, http.StatusInternalServerError, pkg.ErrRetrieving.Error()},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			persistMock := persistMocks.Persistence{}
			testLocalizationBundle, err := translation.NewBundler(localizationFiles)
			require.NoError(t, err)
			jobRepoController := New(logger.MockLogger{}, &persistMock, testLocalizationBundle)

			persistMock.On("GetByOwner", mock.Anything).Return(test.PersistMockJobs, test.PersistMockErr)

			req := httptest.NewRequest("GET", "http://localhost", nil)
			if test.Owner != "nil" {
				req = mux.SetURLVars(req, map[string]string{pkg.OwnerKey: test.Owner})
			}
			w := httptest.NewRecorder()

			jobRepoController.GetByOwner(w, req)
			resp := w.Result()
			defer resp.Body.Close()

			require.Equal(t, test.ExpectedStatusCode, resp.StatusCode, "invalid status code")
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			if test.ExpectedStatusCode != http.StatusOK {
				require.Contains(t, string(body), test.ExpectedErrorMsg)
				return
			}

			var jobs []types.Job
			err = json.Unmarshal(body, &jobs)
			require.NoError(t, err)
			assert.Equal(t, test.PersistMockJobs, jobs)
			persistMock.AssertCalled(t, "GetByOwner", mock.Anything)
		})
	}
}

func TestJobRepoController_Update(t *testing.T) {
	job := helpers.CreateTestJob(pkg.JobRepository, fileHostname)
	expected := make(map[string]interface{})
	expected[types.JobOwner] = pkg.JobRepository
	expected[types.JobStatus] = pkg.StatusNoPipeline

	tests := []struct {
		Name               string
		Expected           map[string]interface{}
		Id                 string
		PersistMockErr     error
		ExpectedStatusCode int
		ExpectedErrorMsg   string
	}{
		{"happy path", expected, "1", nil, http.StatusOK, ""},
		{"missing id", nil, "nil", nil, http.StatusBadRequest, "missing jobid in url"},
		{"empty id", nil, "", nil, http.StatusBadRequest, "empty jobid in url"},
		{"unmarshal failed", nil, "1", nil, http.StatusBadRequest, pkg.ErrUnmarshallingJob.Error()},
		{"update error", expected, "1", errors.New("update failed"), http.StatusNotFound, "failed to update job for id"},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var requestBody []byte
			persistMock := persistMocks.Persistence{}
			testLocalizationBundle, err := translation.NewBundler(localizationFiles)
			require.NoError(t, err)
			jobRepoController := New(logger.MockLogger{}, &persistMock, testLocalizationBundle)
			if test.Expected != nil {
				requestBody, _ = json.Marshal(test.Expected)
			} else {
				requestBody = nil
			}
			persistMock.On("Update", test.Id, test.Expected).Return(job, test.PersistMockErr)
			req := httptest.NewRequest("PUT", "http://localhost", bytes.NewReader(requestBody))
			w := httptest.NewRecorder()
			if test.Id != "nil" {
				req = mux.SetURLVars(req, map[string]string{pkg.JobIdKey: test.Id})
			}
			jobRepoController.Update(w, req)
			resp := w.Result()
			defer resp.Body.Close()

			require.Equal(t, test.ExpectedStatusCode, resp.StatusCode, "invalid status code")
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			if test.ExpectedStatusCode != http.StatusOK {
				require.Contains(t, string(body), test.ExpectedErrorMsg)
				return
			}
			var actualJob types.Job
			err = json.Unmarshal(body, &actualJob)
			assert.NoError(t, err)
			persistMock.AssertExpectations(t)
		})
	}
}

func TestJobRepoController_UpdatePipeline(t *testing.T) {
	// Note: expectedJob's owner is pkg.OwnerTaskLauncher as the pipeline is not an owner of the job.
	expectedJob := helpers.CreateTestJob(pkg.OwnerTaskLauncher, fileHostname)
	validDetails := expectedJob.PipelineDetails
	expectedJobMultiOutput := expectedJob
	expectedJobMultiOutput.PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile(filepath.Join(".", "test", "outfile1.tiff"), pkg.FileStatusIncomplete, pkg.OwnerDataOrg, pkg.CreateUserFacingError("", nil)),
		helpers.CreateTestFile(filepath.Join(".", "test", "outfile2.tiff"), pkg.FileStatusIncomplete, pkg.OwnerDataOrg, pkg.CreateUserFacingError("", nil))}
	validMultiOutputDetails := expectedJobMultiOutput.PipelineDetails
	invalidDetailsBadStatus := validDetails
	invalidDetailsBadStatus.Status = "bogus"
	existingJob := expectedJob
	existingJob.PipelineDetails.OutputFiles = []types.OutputFile{}
	existingJob.PipelineDetails.Results = ""

	tests := []struct {
		Name                 string
		ExpectedJob          *types.Job
		ExpectedDetails      *types.PipelineInfo
		JobId                string
		TaskId               string
		PersistGetMockErr    error
		PersistUpdateMockErr error
		ExpectedStatusCode   int
		ExpectedErrorMsg     string
	}{
		{"happy path - single output file", &expectedJob, &validDetails, expectedJob.Id, expectedJob.PipelineDetails.TaskId, nil, nil, http.StatusOK, ""},
		{"happy path - multiple output file", &expectedJobMultiOutput, &validMultiOutputDetails, expectedJobMultiOutput.Id, expectedJobMultiOutput.PipelineDetails.TaskId, nil, nil, http.StatusOK, ""},
		{"missing JobId", &expectedJob, &validDetails, "", expectedJob.PipelineDetails.TaskId, nil, nil, http.StatusBadRequest, "empty jobid in url"},
		{"missing TaskId", &expectedJob, &validDetails, expectedJob.Id, "", nil, nil, http.StatusBadRequest, "empty taskid in url"},
		{"invalid JobId", &expectedJob, &validDetails, "101", expectedJob.PipelineDetails.TaskId, errors.New("not found"), nil, http.StatusBadRequest, "url jobid parameter does not match a job"},
		{"invalid TaskId", &expectedJob, &validDetails, expectedJob.Id, "99", nil, nil, http.StatusBadRequest, "url taskid parameter does not match taskid for specified job"},
		{"invalid Details", &expectedJob, &invalidDetailsBadStatus, expectedJob.Id, expectedJob.PipelineDetails.TaskId, nil, nil, http.StatusBadRequest, "invalid Status value"},
		{"update error", &expectedJob, &validDetails, expectedJob.Id, expectedJob.PipelineDetails.TaskId, nil, errors.New("update failed"), http.StatusInternalServerError, "update failed"},
		{"unmarshal error", &expectedJob, nil, expectedJob.Id, expectedJob.PipelineDetails.TaskId, nil, errors.New("update failed"), http.StatusInternalServerError, "failed to unmarshal request"},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var requestBody []byte
			var err error
			persistMock := persistMocks.Persistence{}
			testLocalizationBundle, err := translation.NewBundler(localizationFiles)
			require.NoError(t, err)
			jobRepoController := New(logger.MockLogger{}, &persistMock, testLocalizationBundle)
			if test.ExpectedDetails != nil {
				requestBody, err = json.Marshal(test.ExpectedDetails)
				require.NoError(t, err)
			} else {
				requestBody = nil
			}
			persistMock.On("GetById", mock.Anything).Return(existingJob, test.PersistGetMockErr)
			persistMock.On("Update", mock.Anything, mock.Anything).Return(*test.ExpectedJob, test.PersistUpdateMockErr)
			req := httptest.NewRequest("PUT", "http://localhost", bytes.NewReader(requestBody))
			w := httptest.NewRecorder()
			req = mux.SetURLVars(req, map[string]string{pkg.JobIdKey: test.JobId, pkg.TaskIdKey: test.TaskId})

			jobRepoController.UpdatePipeline(w, req)
			resp := w.Result()
			defer resp.Body.Close()

			require.Equal(t, test.ExpectedStatusCode, resp.StatusCode, "invalid status code")
			if test.ExpectedStatusCode != http.StatusOK {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Contains(t, string(body), test.ExpectedErrorMsg)
				return
			}
			persistMock.AssertExpectations(t)

		})
	}
}

func TestJobRepoController_Delete(t *testing.T) {
	tests := []struct {
		Name               string // test name
		Id                 string
		PersistMockErr     error
		ExpectedStatusCode int
		ExpectedErrorMsg   string
	}{
		{"happy path", "1", nil, http.StatusOK, ""},
		{"missing id", "nil", nil, http.StatusBadRequest, "missing jobid in url"},
		{"empty id", "", nil, http.StatusBadRequest, "empty jobid in url"},
		{"delete failed", "1", errors.New("delete failed"), http.StatusInternalServerError, "failed to delete job for id"},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			persistMock := persistMocks.Persistence{}
			testLocalizationBundle, err := translation.NewBundler(localizationFiles)
			require.NoError(t, err)
			jobRepoController := New(logger.MockLogger{}, &persistMock, testLocalizationBundle)

			persistMock.On("Delete", mock.Anything).Return(test.PersistMockErr)

			req := httptest.NewRequest("DELETE", "http://localhost", nil)

			if test.Id != "nil" {
				req = mux.SetURLVars(req, map[string]string{pkg.JobIdKey: test.Id})
			}
			w := httptest.NewRecorder()

			jobRepoController.Delete(w, req)
			resp := w.Result()
			defer resp.Body.Close()

			require.Equal(t, test.ExpectedStatusCode, resp.StatusCode, "invalid status code")
			if test.ExpectedStatusCode != http.StatusOK {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				require.Contains(t, string(body), test.ExpectedErrorMsg)
				return
			}

			persistMock.AssertCalled(t, "Delete", mock.Anything)

		})
	}
}
