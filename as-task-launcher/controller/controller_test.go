/*********************************************************************
 * Copyright (c) Intel Corporation 2023
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
	"path/filepath"
	"testing"
	"time"

	"aicsd/as-task-launcher/config"
	persistMocks "aicsd/as-task-launcher/persist/mocks"
	taskPkg "aicsd/as-task-launcher/pkg"
	"aicsd/pkg"
	jobHandlerMocks "aicsd/pkg/clients/job_handler/mocks"
	jobRepoMocks "aicsd/pkg/clients/job_repo/mocks"
	"aicsd/pkg/helpers"
	"aicsd/pkg/types"

	appsdk "github.com/edgexfoundry/app-functions-sdk-go/v3/pkg"
	"github.com/edgexfoundry/app-functions-sdk-go/v3/pkg/interfaces/mocks"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/common"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	RetryThreshold = 1e12 // 100 seconds (in ns)
	fileHostname   = "gateway"
	file           = "outfile1.tiff"
	file2          = "outfile2.tiff"
	bogusFile      = "bogus.tiff"
)

func TestController_Create(t *testing.T) {
	valid := types.Task{
		Id:               "",
		Description:      "Count Cells",
		JobSelector:      `{ "==" : [ { "var" : "Id" }, "1" ] }`,
		PipelineId:       "1",
		ResultFileFolder: filepath.Join(".", "test"),
		ModelParameters:  map[string]string{"Brightness": "0"},
	}

	invalidWithId := valid
	invalidWithId.Id = "123"

	expectedId := "1"

	tests := []struct {
		Name               string
		Task               *types.Task
		ExpectedId         string
		PersistMockErr     error
		ExpectedStatusCode int
		ExpectedErrorMsg   string
	}{
		{"happy path: new", &valid, expectedId, nil, http.StatusCreated, ""},
		{"bad request body", &invalidWithId, expectedId, nil, http.StatusBadRequest, taskPkg.ErrDuplicateTaskId.Error()},
		{"unmarshal failed", nil, "", nil, http.StatusBadRequest, pkg.ErrJSONMarshalErr.Error()},
		{"persist create failed", &types.Task{}, "", taskPkg.ErrTaskCreation, http.StatusInternalServerError, taskPkg.ErrTaskCreation.Error()},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var requestBody []byte
			persistMock := persistMocks.Persistence{}
			taskRepoController := New(logger.MockLogger{}, &persistMock, nil, nil, nil, nil, &config.Configuration{})

			if test.Task != nil {
				requestBody, _ = json.Marshal(test.Task)
			} else {
				requestBody = nil
			}

			persistMock.On("Create", mock.Anything).Return(test.ExpectedId, test.PersistMockErr)

			req := httptest.NewRequest("POST", "http://localhost", bytes.NewReader(requestBody))
			w := httptest.NewRecorder()

			taskRepoController.Create(w, req)
			resp := w.Result()
			body, err := io.ReadAll(resp.Body)
			defer resp.Body.Close()

			require.Equal(t, test.ExpectedStatusCode, resp.StatusCode, "invalid status code")
			require.NoError(t, err)
			if test.ExpectedStatusCode != http.StatusCreated {
				require.Contains(t, string(body), test.ExpectedErrorMsg)
				return
			}

			require.Equal(t, expectedId, string(body))

			persistMock.AssertCalled(t, "Create", mock.Anything)

		})
	}

}

func TestController_Get(t *testing.T) {
	validTasks := []types.Task{
		{
			Id:               "1",
			Description:      "Count Cells",
			JobSelector:      `{ "==" : [ { "var" : "Id" }, "1" ] }`,
			PipelineId:       "1",
			ResultFileFolder: filepath.Join(".", "test"),
			ModelParameters:  map[string]string{"Brightness": "0"},
		},
		{
			Id:               "2",
			Description:      "Increase Brightness",
			JobSelector:      `{ "==" : [ { "var" : "Id" }, "1" ] }`,
			PipelineId:       "1",
			ResultFileFolder: filepath.Join(".", "test"),
			ModelParameters:  map[string]string{"Brightness": "1"},
		},
	}

	tests := []struct {
		Name               string // test name
		ExpectedTasks      []types.Task
		PersistMockErr     error
		ExpectedStatusCode int
		ExpectedErrorMsg   string
	}{
		{"happy path: get all Tasks", validTasks, nil, http.StatusOK, ""},
		{"empty Tasks", []types.Task{}, nil, http.StatusNoContent, ""},
		{"bad path: failed Get", validTasks, errors.New("failed Get"), http.StatusInternalServerError, pkg.ErrCallingGet.Error()},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			persistMock := persistMocks.Persistence{}
			taskRepoController := New(logger.MockLogger{}, &persistMock, nil, nil, nil, nil, &config.Configuration{})

			persistMock.On("GetAll", mock.Anything).Return(test.ExpectedTasks, test.PersistMockErr)

			req := httptest.NewRequest("GET", "http://localhost", nil)

			w := httptest.NewRecorder()

			taskRepoController.Get(w, req)
			resp := w.Result()
			defer resp.Body.Close()

			require.Equal(t, test.ExpectedStatusCode, resp.StatusCode, "invalid status code")
			body, err := io.ReadAll(resp.Body)
			if test.ExpectedStatusCode != http.StatusOK {
				require.NoError(t, err)
				require.Contains(t, string(body), test.ExpectedErrorMsg)
				return
			}

			var actualTasks []types.Task
			err = json.Unmarshal(body, &actualTasks)
			require.NoError(t, err)
			require.Equal(t, test.ExpectedTasks, actualTasks)
			persistMock.AssertExpectations(t)
		})
	}

}

func TestTaskRepoController_Update(t *testing.T) {
	valid := types.Task{
		Id:               "1",
		Description:      "description",
		JobSelector:      `{ "==" : [ { "var" : "Id" }, "1" ] }`,
		PipelineId:       "1",
		ResultFileFolder: ".",
		ModelParameters:  map[string]string{},
		LastUpdated:      time.Now().UTC().UnixNano(),
	}

	tests := []struct {
		Name               string
		Task               *types.Task
		Id                 string
		PersistMockErr     error
		ExpectedStatusCode int
		ExpectedErrorMsg   string
	}{
		{"happy path", &valid, valid.Id, nil, http.StatusOK, ""},
		{"update error", &valid, valid.Id, errors.New("update failed"), http.StatusNotFound, "failed to update task for Id"},
		{"bad json object", nil, valid.Id, nil, http.StatusBadRequest, "failed to unmarshal request (http://localhost) to task: invalid character 'b' looking for beginning of object key string"},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var requestBody []byte
			persistMock := persistMocks.Persistence{}
			taskRepoController := New(logger.MockLogger{}, &persistMock, nil, nil, nil, nil, &config.Configuration{})
			if test.Task != nil {
				requestBody, _ = json.Marshal(test.Task)
			} else {
				requestBody = []byte("{ badJSON }")
			}
			persistMock.On("Update", mock.Anything, mock.Anything).Return(test.PersistMockErr)
			req := httptest.NewRequest("PUT", "http://localhost", bytes.NewReader(requestBody))
			w := httptest.NewRecorder()
			taskRepoController.Update(w, req)
			resp := w.Result()
			defer resp.Body.Close()

			require.Equal(t, test.ExpectedStatusCode, resp.StatusCode, "invalid status code")
			if test.ExpectedStatusCode != http.StatusOK {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				require.Contains(t, string(body), test.ExpectedErrorMsg)
				return
			}
			persistMock.AssertCalled(t, "Update", mock.Anything, mock.Anything)

		})
	}
}

func TestTaskRepoController_Delete(t *testing.T) {
	tests := []struct {
		Name               string
		Id                 string
		PersistMockErr     error
		ExpectedStatusCode int
		ExpectedErrorMsg   string
	}{
		{"happy path", "1", nil, http.StatusOK, ""},
		{"missing id", "nil", nil, http.StatusBadRequest, "missing taskid in url"},
		{"empty id", "", nil, http.StatusBadRequest, "empty taskid in url"},
		{"delete failed", "1", errors.New("delete failed"), http.StatusBadRequest, "failed to delete task for Id"},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			persistMock := persistMocks.Persistence{}
			taskRepoController := New(logger.MockLogger{}, &persistMock, nil, nil, nil, nil, &config.Configuration{})

			persistMock.On("Delete", mock.Anything).Return(test.PersistMockErr)

			req := httptest.NewRequest("DELETE", "http://localhost", nil)

			if test.Id != "nil" {
				req = mux.SetURLVars(req, map[string]string{pkg.TaskIdKey: test.Id})
			}
			w := httptest.NewRecorder()

			taskRepoController.Delete(w, req)
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

func TestController_RetryOnStartup(t *testing.T) {

	completeJob := []types.Job{helpers.CreateTestJob(pkg.OwnerTaskLauncher, fileHostname)}

	completeJobMultiOutputFiles := []types.Job{helpers.CreateTestJob(pkg.OwnerTaskLauncher, fileHostname)}
	completeJobMultiOutputFiles[0].PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile(file, pkg.FileStatusIncomplete, pkg.OwnerTaskLauncher, pkg.CreateUserFacingError("", nil)),
		helpers.CreateTestFile(file2, pkg.FileStatusIncomplete, pkg.OwnerTaskLauncher, pkg.CreateUserFacingError("", nil))}

	noFileJobs := []types.Job{helpers.CreateTestJob(pkg.OwnerTaskLauncher, fileHostname)}
	noFileJobs[0].PipelineDetails.OutputFiles = []types.OutputFile{}

	noFileFailedJobs := []types.Job{helpers.CreateTestJob(pkg.OwnerTaskLauncher, fileHostname)}
	noFileFailedJobs[0].PipelineDetails.OutputFiles = []types.OutputFile{}
	noFileFailedJobs[0].PipelineDetails.Status = pkg.TaskStatusFailed

	failedJobs := []types.Job{helpers.CreateTestJob(pkg.OwnerTaskLauncher, fileHostname)}
	failedJobs[0].PipelineDetails.Status = pkg.TaskStatusFailed

	processingJobs := []types.Job{helpers.CreateTestJob(pkg.OwnerTaskLauncher, fileHostname)}
	processingJobs[0].PipelineDetails.Status = pkg.TaskStatusProcessing

	noFileFoundJobs := []types.Job{helpers.CreateTestJob(pkg.OwnerTaskLauncher, fileHostname)}
	noFileFoundJobs[0].PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile(bogusFile, pkg.FileStatusIncomplete, pkg.OwnerTaskLauncher, pkg.CreateUserFacingError("", nil))}

	noFileFoundOutputFileJobs := []types.Job{helpers.CreateTestJob(pkg.OwnerTaskLauncher, fileHostname)}
	noFileFoundOutputFileJobs[0].PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile(bogusFile, pkg.FileStatusIncomplete, pkg.OwnerTaskLauncher, pkg.CreateUserFacingError("", nil))}

	noFileFoundMultiOutputFileJobs := []types.Job{helpers.CreateTestJob(pkg.OwnerTaskLauncher, fileHostname)}
	noFileFoundMultiOutputFileJobs[0].PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile(file, pkg.FileStatusIncomplete, pkg.OwnerTaskLauncher, pkg.CreateUserFacingError("", nil)),
		helpers.CreateTestFile(bogusFile, pkg.FileStatusIncomplete, pkg.OwnerTaskLauncher, pkg.CreateUserFacingError("", nil))}

	tasks := []types.Task{
		{Id: "task1",
			PipelineId:  "pipeline1",
			JobSelector: `{ "==" : [ { "var" : "Id" }, "1" ] }`,
		},
	}
	noTasks := []types.Task{
		{Id: "task2",
			PipelineId:  "badtask",
			JobSelector: `{ "==" : [ { "var" : "Id" }, "2" ] }`,
		},
	}
	noBoolTasks := []types.Task{
		{Id: "task2",
			PipelineId:  "badtask",
			JobSelector: `{ "var" : [ "Owner" ] }`,
		},
	}

	tests := []struct {
		Name              string
		RetryWindow       int64
		Jobs              []types.Job
		RepoRetrieveErr   error
		RepoUpdateErr     error
		SenderErr         error
		PersistTasks      []types.Task
		PersistGetErr     error
		RepoUpdateErr2    error
		PublisherErr      error
		ExpectedError     error
		ExpectedMatchTask bool
	}{
		{"happy path pipeline - single output file complete", 0, completeJob, nil, nil, nil, nil, nil, nil, nil, nil, false},
		{"happy path pipeline - multi output file complete", 0, completeJobMultiOutputFiles, nil, nil, nil, nil, nil, nil, nil, nil, false},
		{"happy path pipeline complete, no file", 0, noFileJobs, nil, nil, nil, nil, nil, nil, nil, nil, false},
		{"happy path pipeline failed, no file", 0, noFileFailedJobs, nil, nil, nil, nil, nil, nil, nil, nil, false},
		{"happy path pipeline failed", 0, failedJobs, nil, nil, nil, nil, nil, nil, nil, nil, false},
		{"happy path pipeline processing", 1, processingJobs, nil, nil, nil, tasks, nil, nil, nil, nil, true},
		{"happy path pipeline processing no match", 0, processingJobs, nil, nil, nil, noTasks, nil, nil, nil, nil, false},
		{"happy path pipeline processing no boolean task", 0, processingJobs, nil, nil, nil, noBoolTasks, nil, nil, nil, nil, false},
		{"happy path pipeline processing outside retry band", RetryThreshold, processingJobs, nil, nil, nil, nil, nil, nil, nil, nil, false},
		{"retrieve all failed", 0, nil, pkg.ErrRetrieving, nil, nil, nil, nil, nil, nil, pkg.ErrRetrieving, false},
		{"complete pipeline no file update failed", 0, noFileJobs, nil, pkg.ErrUpdating, nil, nil, nil, nil, nil, pkg.ErrUpdating, false},
		{"failed pipeline no file update failed", 0, noFileFailedJobs, nil, pkg.ErrUpdating, nil, nil, nil, nil, nil, pkg.ErrUpdating, false},
		{"complete pipeline file not found", 0, noFileFoundJobs, nil, nil, nil, nil, nil, nil, nil, nil, false},
		{"complete pipeline file not found update failed", 0, noFileFoundJobs, nil, nil, nil, nil, nil, pkg.ErrUpdating, nil, pkg.ErrUpdating, false},
		{"complete pipeline output file not found", 0, noFileFoundOutputFileJobs, nil, nil, nil, nil, nil, nil, nil, nil, false},
		{"complete pipeline output file not found update failed", 0, noFileFoundOutputFileJobs, nil, nil, nil, nil, nil, pkg.ErrUpdating, nil, pkg.ErrUpdating, false},
		{"complete pipeline multi output file not found", 0, noFileFoundMultiOutputFileJobs, nil, nil, nil, nil, nil, nil, nil, nil, false},
		{"complete pipeline multi output file not found update failed", 0, noFileFoundMultiOutputFileJobs, nil, nil, nil, nil, nil, pkg.ErrUpdating, nil, pkg.ErrUpdating, false},
		{"complete pipeline send failed", 0, completeJob, nil, nil, pkg.ErrHandleJob, nil, nil, nil, nil, pkg.ErrHandleJob, false},
		{"complete pipeline multi output file send failed", 0, completeJobMultiOutputFiles, nil, nil, pkg.ErrHandleJob, nil, nil, nil, nil, pkg.ErrHandleJob, false},
		{"job processing no match update failed", 0, processingJobs, nil, nil, nil, noTasks, nil, pkg.ErrUpdating, nil, pkg.ErrUpdating, false},
		{"job processing match update failed", 0, processingJobs, nil, nil, nil, tasks, nil, pkg.ErrUpdating, nil, pkg.ErrUpdating, true},
		{"job processing match publish failed", 0, processingJobs, nil, nil, nil, tasks, nil, nil, pkg.ErrPublishing, pkg.ErrPublishing, true},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			outputNotFound := true
			taskCompleteOrFailed := false
			taskProcessing := false
			noOutputFile := false
			if test.Jobs != nil {
				taskCompleteOrFailed = test.Jobs[0].PipelineDetails.Status == pkg.TaskStatusComplete || test.Jobs[0].PipelineDetails.Status == pkg.TaskStatusFailed
				taskProcessing = test.Jobs[0].PipelineDetails.Status == pkg.TaskStatusProcessing && test.RetryWindow < RetryThreshold
				noOutputFile = len(test.Jobs[0].PipelineDetails.OutputFiles) == 0
			}

			// build task handler
			mockLogger := logger.MockLogger{}
			persistMock := persistMocks.Persistence{}
			repoMock := jobRepoMocks.Client{}
			senderMock := jobHandlerMocks.Client{}
			backgroundPublisherMock := mocks.BackgroundPublisher{}
			appServiceMock := mocks.ApplicationService{}
			taskConfig := config.Configuration{
				DeviceProfileName: "my-profile",
				DeviceName:        "device1",
				ResourceName:      "PipelineParameters",
				FileHostname:      fileHostname,
			}
			taskHandler := New(mockLogger, &persistMock, &repoMock, &senderMock, &backgroundPublisherMock, &appServiceMock, &taskConfig)
			ctx := appsdk.NewAppFuncContextForTest(uuid.NewString(), mockLogger)

			// set mocks
			repoMock.On("RetrieveAllByOwner", pkg.OwnerTaskLauncher).Return(test.Jobs, test.RepoRetrieveErr)
			if taskCompleteOrFailed {
				if noOutputFile {
					jobFields := make(map[string]interface{})
					jobFields[types.JobOwner] = pkg.OwnerNone
					if test.Jobs[0].PipelineDetails.Status == pkg.TaskStatusComplete {
						jobFields[types.JobStatus] = pkg.StatusComplete
					} else {
						jobFields[types.JobStatus] = pkg.StatusPipelineError
						jobFields[types.JobErrorDetailsOwner] = pkg.OwnerTaskLauncher
						jobFields[types.JobErrorDetailsErrorMsg] = pkg.ErrPipelineFailed.Error()
					}
					repoMock.On("Update", mock.Anything, jobFields).Return(types.Job{}, test.RepoUpdateErr)
				} else if err := test.Jobs[0].ValidateFiles(); err != nil {
					jobFields := make(map[string]interface{})
					jobFields[types.JobOwner] = pkg.OwnerNone
					jobFields[types.JobStatus] = pkg.StatusPipelineError
					jobFields[types.JobPipelineStatus] = pkg.TaskStatusFileNotFound
					jobFields[types.JobPipelineOutputFiles] = test.Jobs[0].PipelineDetails.OutputFiles
					jobFields[types.JobErrorDetailsOwner] = pkg.OwnerTaskLauncher
					jobFields[types.JobErrorDetailsErrorMsg] = pkg.ErrFileInvalid.Error()
					repoMock.On("Update", mock.Anything, jobFields).Return(types.Job{}, test.RepoUpdateErr2)
					outputNotFound = false
				} else {
					senderMock.On("HandleJob", mock.Anything).Return(test.SenderErr)
				}
			} else if taskProcessing {
				persistMock.On("GetAll").Return(test.PersistTasks, test.PersistGetErr)
				if test.ExpectedMatchTask {
					jobFields := make(map[string]interface{})
					jobFields[types.JobOwner] = pkg.OwnerTaskLauncher
					jobFields[types.JobPipelineTaskId] = test.PersistTasks[0].Id
					jobFields[types.JobPipelineOutputHost] = fileHostname
					repoMock.On("Update", mock.Anything, jobFields).Return(types.Job{}, test.RepoUpdateErr2)
					backgroundPublisherMock.On("Publish", mock.Anything, ctx).Return(test.PublisherErr)
					appServiceMock.On("BuildContext", mock.Anything, common.ContentTypeJSON).Return(ctx)
				} else {
					jobFields := make(map[string]interface{})
					jobFields[types.JobOwner] = pkg.OwnerNone
					jobFields[types.JobStatus] = pkg.StatusNoPipeline
					jobFields[types.JobErrorDetailsOwner] = pkg.OwnerTaskLauncher
					jobFields[types.JobErrorDetailsErrorMsg] = pkg.ErrJobNoMatchingTask.Error()
					repoMock.On("Update", mock.Anything, jobFields).Return(types.Job{}, test.RepoUpdateErr2)
				}

			}
			// call retry on startup
			errs := taskHandler.RetryOnStartup(test.RetryWindow)
			// check the error against and expected error
			if test.ExpectedError != nil {
				require.NotNil(t, errs)
				require.Contains(t, errs.Error(), test.ExpectedError.Error())
				return
			}
			if taskCompleteOrFailed {
				repoMock.AssertExpectations(t)
				if outputNotFound {
					senderMock.AssertExpectations(t)
				}
			} else if taskProcessing {
				persistMock.AssertExpectations(t)
				if test.ExpectedMatchTask && test.RepoUpdateErr2 == nil {
					backgroundPublisherMock.AssertExpectations(t)
					if test.PublisherErr == nil {
						appServiceMock.AssertExpectations(t)
					}
				}
			}
		})
	}
}

func TestController_retry(t *testing.T) {
	completeJobs := []types.Job{helpers.CreateTestJob(pkg.OwnerTaskLauncher, fileHostname)}
	completeJobs[0].Status = pkg.TaskStatusComplete
	tests := []struct {
		Name               string
		RequestBody        []byte
		Jobs               []types.Job
		ExpectedErr        error
		ExpectedStatusCode int
	}{
		// NOTE: These test cases only test for invalid input,
		// because the RetryOnStartup function has already been unit tested
		{"good data", []byte("{ \"TimeoutDuration\":\"12s\" }"), completeJobs, nil, http.StatusOK},
		{"bad requestBody unmarshal", nil, completeJobs, fmt.Errorf("unexpected end of JSON input"),
			http.StatusBadRequest},
		{"bad timeDuration", []byte("{ \"bogus\": \"bogus\" }"),
			completeJobs, fmt.Errorf(pkg.ErrFmtProcessingReq, ""),
			http.StatusInternalServerError},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			// build task handler
			mockLogger := logger.MockLogger{}
			persistMock := persistMocks.Persistence{}
			repoMock := jobRepoMocks.Client{}
			senderMock := jobHandlerMocks.Client{}
			backgroundPublisherMock := mocks.BackgroundPublisher{}
			appServiceMock := mocks.ApplicationService{}
			taskConfig := config.Configuration{
				DeviceProfileName: "my-profile",
				DeviceName:        "device1",
				ResourceName:      "PipelineParameters",
				FileHostname:      fileHostname,
			}
			taskHandler := New(&mockLogger, &persistMock, &repoMock, &senderMock, &backgroundPublisherMock, &appServiceMock, &taskConfig)
			repoMock.On("RetrieveAllByOwner", pkg.OwnerTaskLauncher).Return(test.Jobs, nil)
			if test.Jobs[0].Status == pkg.TaskStatusComplete {
				senderMock.On("HandleJob", test.Jobs[0]).Return(nil)
			}
			req := httptest.NewRequest("POST", "http://localhost", bytes.NewReader(test.RequestBody))
			w := httptest.NewRecorder()

			// call retry
			taskHandler.retry(w, req)
			if test.ExpectedErr == nil {
				repoMock.AssertExpectations(t)
			}

			resp := w.Result()
			defer resp.Body.Close()

			// check the status code and body
			require.Equal(t, test.ExpectedStatusCode, resp.StatusCode, "invalid status code")
			if test.ExpectedStatusCode != http.StatusOK {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				require.Contains(t, string(body), test.ExpectedErr.Error())
				return
			}
		})
	}
}

func TestController_MatchTask(t *testing.T) {
	expected := helpers.CreateTestJob(pkg.OwnerTaskLauncher, fileHostname)

	tasks := []types.Task{
		{Id: "task1",
			PipelineId:  "pipeline1",
			JobSelector: `{ "==" : [ { "var" : "Id" }, "1" ] }`,
		},
	}
	noTasks := []types.Task{
		{Id: "task2",
			PipelineId:  "badtask",
			JobSelector: `{ "==" : [ { "var" : "Id" }, "2" ] }`,
		},
	}
	noBoolTasks := []types.Task{
		{Id: "task2",
			PipelineId:  "badtask",
			JobSelector: `{ "var" : [ "Owner" ] }`,
		},
	}

	tests := []struct {
		Name               string
		Expected           *types.Job
		PersistFilterTasks []types.Task
		PersistFilterErr   error
		ExpectedStatusCode int
		ExpectedErrorMsg   string
		ExpectedRespBody   string
	}{
		{"happy path with match", &expected, tasks, nil, http.StatusOK, "", "true"},
		{"happy path no match", &expected, noTasks, nil, http.StatusOK, "", "false"},
		{"bad request body", nil, []types.Task{}, nil, http.StatusBadRequest, "failed to unmarshal request", "bogus"},
		{"persist filter failed", &expected, []types.Task{}, errors.New("could not filter tasks"), http.StatusInternalServerError, "could not retrieve tasks", "bogus"},
		{"non bool json logic", &expected, noBoolTasks, nil, http.StatusInternalServerError, "could not parse bool from json logic result", "bogus"},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var requestBody []byte
			var err error

			// build task handler
			persistMock := persistMocks.Persistence{}
			repoMock := jobRepoMocks.Client{}
			taskHandler := New(logger.MockLogger{}, &persistMock, &repoMock, nil, nil, nil, &config.Configuration{})

			// build up the request
			if test.Expected != nil {
				requestBody, err = json.Marshal(test.Expected)
				require.NoError(t, err)
			} else {
				requestBody = nil
			}
			req := httptest.NewRequest("POST", "http://localhost", bytes.NewReader(requestBody))
			w := httptest.NewRecorder()

			// set mocks
			persistMock.On("GetAll").Return(test.PersistFilterTasks, test.PersistFilterErr)

			// call MatchTask
			taskHandler.MatchTask(w, req)
			resp := w.Result()
			defer resp.Body.Close()

			// check the status code and body
			require.Equal(t, test.ExpectedStatusCode, resp.StatusCode, "invalid status code")
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			if test.ExpectedStatusCode != http.StatusOK {
				require.Contains(t, string(body), test.ExpectedErrorMsg)
				return
			}
			require.Contains(t, string(body), test.ExpectedRespBody)
			persistMock.AssertCalled(t, "GetAll")
		})
	}
}

func TestController_HandleNewJob(t *testing.T) {
	expected := helpers.CreateTestJob(pkg.OwnerTaskLauncher, fileHostname)
	tasks := []types.Task{
		{Id: "task1",
			PipelineId:  "pipeline1",
			JobSelector: `{ "==" : [ { "var" : "Id" }, "1" ] }`,
			ModelParameters: map[string]string{
				"Brighten": "80",
			},
		},
	}
	noTasks := []types.Task{
		{Id: "task2",
			PipelineId:  "nomatch",
			JobSelector: `{ "==" : [ { "var" : "Id" }, "2" ] }`,
		},
	}

	tests := []struct {
		Name               string
		Expected           *types.Job
		PersistFilterTasks []types.Task
		PersistFilterErr   error
		RepoMockUpdateErr  error
		PublisherErr       error
		ExpectedStatusCode int
		ExpectedErrorMsg   string
	}{
		{"happy path with match", &expected, tasks, nil, nil, nil, http.StatusOK, ""},
		{"bad request body", nil, []types.Task{}, nil, nil, nil, http.StatusBadRequest, "failed to unmarshal request"},
		{"persist filter failed", &expected, []types.Task{}, errors.New("could not filter tasks"), nil, nil, http.StatusInternalServerError, "could not retrieve tasks"},
		{"happy path no matched task", &expected, noTasks, nil, nil, nil, http.StatusOK, ""},
		{"no matched task - fail repo update", &expected, noTasks, nil, errors.New("update failed"), nil, http.StatusInternalServerError, "could not update job repo to status no pipeline for job id"},
		{"fail repo update", &expected, tasks, nil, errors.New("update failed"), nil, http.StatusInternalServerError, "could not update job repo for job id"},
		{"publish failed", &expected, tasks, nil, nil, errors.New("publish failed"), http.StatusOK, ""},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var requestBody []byte
			var err error

			// build task handler
			mockLogger := logger.MockLogger{}
			persistMock := persistMocks.Persistence{}
			repoMock := jobRepoMocks.Client{}
			backgroundPublisherMock := mocks.BackgroundPublisher{}
			appServiceMock := mocks.ApplicationService{}
			taskConfig := config.Configuration{
				DeviceProfileName: "my-profile",
				DeviceName:        "device1",
				ResourceName:      "PipelineParameters",
			}
			taskHandler := New(&mockLogger, &persistMock, &repoMock, nil, &backgroundPublisherMock,
				&appServiceMock, &taskConfig)

			// build up the request
			if test.Expected != nil {
				requestBody, err = json.Marshal(test.Expected)
				require.NoError(t, err)
			} else {
				requestBody = nil
			}
			req := httptest.NewRequest("POST", "http://localhost", bytes.NewReader(requestBody))
			w := httptest.NewRecorder()

			// set mocks
			persistMock.On("GetAll").Return(test.PersistFilterTasks, test.PersistFilterErr)
			repoMock.On("Update", mock.Anything, mock.Anything).Return(types.Job{}, test.RepoMockUpdateErr)
			ctx := appsdk.NewAppFuncContextForTest(uuid.NewString(), mockLogger)
			backgroundPublisherMock.On("Publish", mock.Anything, ctx).Return(test.PublisherErr)
			appServiceMock.On("BuildContext", mock.Anything, common.ContentTypeJSON).Return(ctx)

			// call HandleNewJob
			taskHandler.HandleNewJob(w, req)
			resp := w.Result()
			defer resp.Body.Close()

			// check the status code and body
			require.Equal(t, test.ExpectedStatusCode, resp.StatusCode, "invalid status code")

			if test.Expected != nil {
				persistMock.AssertCalled(t, "GetAll")
			}
			if test.Expected != nil && test.PersistFilterErr == nil {
				repoMock.AssertCalled(t, "Update", mock.Anything, mock.Anything)
			}
			if test.ExpectedStatusCode != http.StatusOK {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				require.Contains(t, string(body), test.ExpectedErrorMsg)
				return
			}
			if test.PublisherErr != nil {
				backgroundPublisherMock.AssertCalled(t, "Publish", mock.Anything, ctx)
			}
		})
	}
}

func TestController_PipelineStatus(t *testing.T) {
	completeJob := helpers.CreateTestJob(pkg.OwnerTaskLauncher, fileHostname)

	completeJobMultiOutputFiles := completeJob
	completeJobMultiOutputFiles.PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile(file, pkg.FileStatusIncomplete, pkg.OwnerTaskLauncher, pkg.CreateUserFacingError("", nil)),
		helpers.CreateTestFile(file2, pkg.FileStatusIncomplete, pkg.OwnerTaskLauncher, pkg.CreateUserFacingError("", nil))}

	noFileJob := helpers.CreateTestJob(pkg.OwnerTaskLauncher, fileHostname)
	noFileJob.PipelineDetails.OutputFiles = []types.OutputFile{}

	noFileJobFailed := completeJob
	noFileJobFailed.PipelineDetails.OutputFiles = nil

	noFileJobFailed.PipelineDetails.Status = pkg.TaskStatusFailed
	noFileJobFailed.Status = pkg.StatusPipelineError

	bogusOutputJob := completeJob
	bogusOutputJob.PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile(bogusFile, pkg.FileStatusIncomplete, pkg.OwnerTaskLauncher, pkg.CreateUserFacingError("", nil))}

	invalidOutputFileJob := completeJob
	invalidOutputFileJob.PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile(bogusFile, pkg.FileStatusIncomplete, pkg.OwnerTaskLauncher, pkg.CreateUserFacingError("", nil))}

	invalidSecondOutputFileJob := completeJob
	invalidSecondOutputFileJob.PipelineDetails.OutputFiles = []types.OutputFile{helpers.CreateTestFile(file, pkg.FileStatusIncomplete, pkg.OwnerTaskLauncher, pkg.CreateUserFacingError("", nil)),
		helpers.CreateTestFile(bogusFile, pkg.FileStatusIncomplete, pkg.OwnerTaskLauncher, pkg.CreateUserFacingError("", nil))}

	tests := []struct {
		Name                string
		JobId               string
		TaskId              string
		TaskStatus          string
		Job                 types.Job
		JobRetrieveErr      error
		RepoUpdateErr       error
		RepoUpdateErrNoFile error
		SenderHandleJobErr  error
		PublisherErr        error
		ExpectedStatusCode  int
		ExpectedErrorMsg    string
	}{
		{"happy path with output file list", completeJob.Id, completeJob.PipelineDetails.TaskId, completeJob.PipelineDetails.Status, completeJob, nil, nil, nil, nil, nil, http.StatusOK, ""},
		{"happy path with multiple output files", completeJobMultiOutputFiles.Id, completeJobMultiOutputFiles.PipelineDetails.TaskId, completeJobMultiOutputFiles.PipelineDetails.Status, completeJobMultiOutputFiles, nil, nil, nil, nil, nil, http.StatusOK, ""},
		{"happy path no file complete", noFileJob.Id, noFileJob.PipelineDetails.TaskId, noFileJob.PipelineDetails.Status, noFileJob, nil, nil, nil, nil, nil, http.StatusOK, ""},
		{"happy path no file failed", noFileJobFailed.Id, noFileJobFailed.PipelineDetails.TaskId, noFileJobFailed.PipelineDetails.Status, noFileJobFailed, nil, nil, nil, nil, nil, http.StatusOK, ""},
		{"missing jobid", "nil", completeJob.PipelineDetails.TaskId, completeJob.PipelineDetails.Status, completeJob, nil, nil, nil, nil, nil, http.StatusBadRequest, "missing jobid in url"},
		{"empty jobid", "", completeJob.PipelineDetails.TaskId, completeJob.PipelineDetails.Status, completeJob, nil, nil, nil, nil, nil, http.StatusBadRequest, "empty jobid in url"},
		{"missing taskid", completeJob.Id, "nil", completeJob.PipelineDetails.Status, completeJob, nil, nil, nil, nil, nil, http.StatusBadRequest, "missing taskid in url"},
		{"empty taskid", completeJob.Id, "", completeJob.PipelineDetails.Status, completeJob, nil, nil, nil, nil, nil, http.StatusBadRequest, "empty taskid in url"},
		{"bogus status", completeJob.Id, completeJob.PipelineDetails.TaskId, "bogus", completeJob, nil, nil, nil, nil, nil, http.StatusBadRequest, "request body does not match expected task status"},
		{"task status processing", completeJob.Id, completeJob.PipelineDetails.TaskId, pkg.TaskStatusProcessing, completeJob, nil, nil, nil, nil, nil, http.StatusBadRequest, "task status not set to failed or complete for job id "},
		{"retrieve by id failed", completeJob.Id, completeJob.PipelineDetails.TaskId, completeJob.PipelineDetails.Status, types.Job{}, pkg.ErrRetrieving, nil, nil, nil, nil, http.StatusBadRequest, pkg.ErrRetrieving.Error()},
		{"task id mismatch", completeJob.Id, "bogus", completeJob.PipelineDetails.Status, completeJob, nil, nil, nil, nil, nil, http.StatusBadRequest, "task id mismatch"},
		{"task status mismatch", completeJob.Id, completeJob.PipelineDetails.TaskId, pkg.TaskStatusFailed, completeJob, nil, nil, nil, nil, nil, http.StatusBadRequest, "task status mismatch"},
		{"no file job complete repo update fail", noFileJob.Id, noFileJob.PipelineDetails.TaskId, noFileJob.PipelineDetails.Status, noFileJob, nil, pkg.ErrRetrieving, nil, nil, nil, http.StatusInternalServerError, "could not update job repo to status to Complete"},
		{"no file job failed repo update fail", noFileJobFailed.Id, noFileJobFailed.PipelineDetails.TaskId, noFileJobFailed.PipelineDetails.Status, noFileJobFailed, nil, pkg.ErrRetrieving, nil, nil, nil, http.StatusInternalServerError, "could not update job repo to status to PipelineError"},
		{"file not found", bogusOutputJob.Id, bogusOutputJob.PipelineDetails.TaskId, bogusOutputJob.PipelineDetails.Status, bogusOutputJob, nil, nil, nil, nil, nil, http.StatusOK, ""},
		{"file not found - update failed", bogusOutputJob.Id, bogusOutputJob.PipelineDetails.TaskId, bogusOutputJob.PipelineDetails.Status, bogusOutputJob, nil, nil, pkg.ErrUpdating, nil, nil, http.StatusOK, ""},
		// TODO: fix these error cases when failure strategy is determined
		{"output file list file not found", invalidOutputFileJob.Id, invalidOutputFileJob.PipelineDetails.TaskId, invalidOutputFileJob.PipelineDetails.Status, invalidOutputFileJob, nil, nil, nil, nil, nil, http.StatusOK, ""},
		{"output file list invalid second file", invalidSecondOutputFileJob.Id, invalidSecondOutputFileJob.PipelineDetails.TaskId, invalidSecondOutputFileJob.PipelineDetails.Status, invalidSecondOutputFileJob, nil, nil, nil, nil, nil, http.StatusOK, ""},
		{"output file list file not found - update failed", invalidOutputFileJob.Id, invalidOutputFileJob.PipelineDetails.TaskId, invalidOutputFileJob.PipelineDetails.Status, invalidOutputFileJob, nil, nil, pkg.ErrUpdating, nil, nil, http.StatusOK, ""},
		{"handle job call failed", completeJob.Id, completeJob.PipelineDetails.TaskId, completeJob.PipelineDetails.Status, completeJob, nil, nil, nil, pkg.ErrHandleJob, nil, http.StatusOK, ""},
		{"publish results failed", completeJob.Id, completeJob.PipelineDetails.TaskId, completeJob.PipelineDetails.Status, completeJob, nil, nil, nil, nil, pkg.ErrPublishing, http.StatusOK, ""},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var requestBody []byte
			var err error
			outputNotFound := true

			// build task handler
			mockLogger := logger.MockLogger{}
			repoMock := jobRepoMocks.Client{}
			senderMock := jobHandlerMocks.Client{}
			backgroundPublisherMock := mocks.BackgroundPublisher{}
			appServiceMock := mocks.ApplicationService{}
			taskHandler := New(&mockLogger, nil, &repoMock, &senderMock, &backgroundPublisherMock, &appServiceMock, &config.Configuration{FileHostname: fileHostname})

			// build up the request
			if test.TaskStatus != "" {
				requestBody = []byte(test.TaskStatus)
				require.NoError(t, err)
			} else {
				requestBody = nil
			}
			req := httptest.NewRequest("POST", "http://localhost", bytes.NewReader(requestBody))
			if test.JobId != "nil" && test.TaskId != "nil" {
				req = mux.SetURLVars(req, map[string]string{pkg.JobIdKey: test.JobId, pkg.TaskIdKey: test.TaskId})
			} else if test.JobId != "nil" && test.TaskId == "nil" {
				req = mux.SetURLVars(req, map[string]string{pkg.JobIdKey: test.JobId})
			} else if test.JobId == "nil" && test.TaskId != "nil" {
				req = mux.SetURLVars(req, map[string]string{pkg.TaskIdKey: test.TaskId})
			}
			w := httptest.NewRecorder()

			// set mocks
			repoMock.On("RetrieveById", mock.Anything).Return(test.Job, test.JobRetrieveErr)
			ctx := appsdk.NewAppFuncContextForTest(uuid.NewString(), mockLogger)
			appServiceMock.On("BuildContext", mock.Anything, common.ContentTypeText).Return(ctx)
			backgroundPublisherMock.On("Publish", mock.Anything, ctx).Return(test.PublisherErr)
			if len(test.Job.PipelineDetails.OutputFiles) == 0 {
				jobFields := make(map[string]interface{})
				jobFields[types.JobOwner] = pkg.OwnerNone
				if test.Job.PipelineDetails.Status == pkg.TaskStatusComplete {
					jobFields[types.JobStatus] = pkg.StatusComplete
				} else {
					jobFields[types.JobStatus] = pkg.StatusPipelineError
					jobFields[types.JobErrorDetailsOwner] = pkg.OwnerTaskLauncher
					jobFields[types.JobErrorDetailsErrorMsg] = pkg.ErrPipelineFailed.Error()
				}
				repoMock.On("Update", test.JobId, jobFields).Return(test.Job, test.RepoUpdateErr)
			} else {
				if err := test.Job.ValidateFiles(); err != nil {
					jobFields := make(map[string]interface{})
					jobFields[types.JobOwner] = pkg.OwnerNone
					jobFields[types.JobStatus] = pkg.StatusPipelineError
					jobFields[types.JobPipelineStatus] = pkg.TaskStatusFileNotFound
					jobFields[types.JobErrorDetailsOwner] = pkg.OwnerTaskLauncher
					jobFields[types.JobErrorDetailsErrorMsg] = pkg.ErrFileInvalid.Error()
					jobFields[types.JobPipelineOutputFiles] = test.Job.PipelineDetails.OutputFiles
					repoMock.On("Update", test.JobId, jobFields).Return(test.Job, test.RepoUpdateErrNoFile)
					outputNotFound = false
				} else {
					senderMock.On("HandleJob", mock.Anything).Return(test.SenderHandleJobErr)
				}
			}
			// call PipelineStatus
			taskHandler.PipelineStatus(w, req)
			resp := w.Result()
			defer resp.Body.Close()

			// check the status code and body
			require.Equal(t, test.ExpectedStatusCode, resp.StatusCode, "invalid status code")
			if test.ExpectedStatusCode != http.StatusOK {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				require.Contains(t, string(body), test.ExpectedErrorMsg)
				return
			}
			if !outputNotFound {
				repoMock.AssertExpectations(t)
			}
			if outputNotFound {
				senderMock.AssertExpectations(t)
			}
			appServiceMock.AssertExpectations(t)
			backgroundPublisherMock.AssertExpectations(t)
		})
	}
}
