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
	"testing"

	"aicsd/pkg/wait"

	"github.com/edgexfoundry/go-mod-core-contracts/v3/clients/logger"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	fileReceiverMocks "aicsd/ms-file-sender-oem/clients/file_receiver/mocks"
	"aicsd/pkg"
	jobRepoMocks "aicsd/pkg/clients/job_repo/mocks"
	"aicsd/pkg/helpers"
	"aicsd/pkg/types"
)

const fileHostname = "oem"

var dependentServices = wait.Services{wait.ServiceConsul, wait.ServiceJobRepo}

func TestFileHandler_RetryOnStartup(t *testing.T) {
	expected := []types.Job{helpers.CreateTestJob(pkg.OwnerFileSenderOem, fileHostname)}

	tests := []struct {
		Name                  string
		Jobs                  *[]types.Job
		RepoMockRetrieveError error
		TransmitJobError      error
		TransmitFileError     error
		RepoUpdateError       error
		ExpectedError         error
	}{
		{"happy path", &expected, nil, nil, nil, nil, nil},
		{"retrieve failed", &[]types.Job{}, errors.New("database RetrieveAllByOwner failed"), nil, nil, nil, errors.New("could not retrieve file-sender-oem data: database RetrieveAllByOwner failed")},
		{"transmit job failed", &expected, nil, errors.New("transmit job failed"), nil, nil, fmt.Errorf("1 error occurred:\n\t* failed to send NotifyNewFile request for file %s: transmit job failed\n\n", expected[0].FullInputFileLocation())},
		{"transmit job failed - file receiver gateway down", &expected, nil, fmt.Errorf(pkg.ErrFmtTcpLookup, pkg.OwnerFileRecvGateway), nil, nil, fmt.Errorf("1 error occurred:\n\t* failed to send NotifyNewFile request for file %s: dial tcp: lookup %s\n\n", expected[0].FullInputFileLocation(), pkg.OwnerFileRecvGateway)},
		{"transmit file failed", &expected, nil, nil, errors.New("transmit file failed"), nil, fmt.Errorf("1 error occurred:\n\t* failed to Transmit File %s: transmit file failed\n\n", expected[0].FullInputFileLocation())},
		{"transmit file failed - file receiver gateway down", &expected, nil, nil, fmt.Errorf(pkg.ErrFmtTcpLookup, pkg.OwnerFileRecvGateway), nil, fmt.Errorf("1 error occurred:\n\t* failed to Transmit File %s: dial tcp: lookup %s\n\n", expected[0].FullInputFileLocation(), pkg.OwnerFileRecvGateway)},
		{"transmit file repo update failed", &expected, nil, nil, errors.New("transmit file failed"), errors.New("update failed"), fmt.Errorf("2 errors occurred:\n\t* failed to update job status to File Transmission Failed for file %s: update failed\n\t* failed to Transmit File %s: transmit file failed\n\n", expected[0].FullInputFileLocation(), expected[0].FullInputFileLocation())},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			repoMock := jobRepoMocks.Client{}
			receiverMock := fileReceiverMocks.Client{}
			fileHandler := New(logger.MockLogger{}, &repoMock, &receiverMock, fileHostname, dependentServices)

			repoMock.On("RetrieveAllByOwner", pkg.OwnerFileSenderOem).Return(*test.Jobs, test.RepoMockRetrieveError)
			receiverMock.On("TransmitJob", mock.Anything).Return(test.TransmitJobError)
			receiverMock.On("TransmitFile", mock.Anything, mock.Anything).Return(0, test.TransmitFileError)
			repoMock.On("Update", mock.Anything, mock.Anything).Return(types.Job{}, test.RepoUpdateError)

			err := fileHandler.RetryOnStartup()

			if test.ExpectedError == nil {
				require.Nil(t, err)
			} else {
				require.Equal(t, test.ExpectedError.Error(), err.Error())
			}
		})
	}
}

func TestFileHandler_SendNewFile(t *testing.T) {
	expected := helpers.CreateTestJob(pkg.OwnerFileSenderOem, fileHostname)
	noFileExpected := expected
	noFileExpected.InputFile.Name = "bogus"
	tests := []struct {
		Name               string
		Expected           *types.Job
		RepoUpdateError1   error
		TransmitJobError   error
		TransmitFileError  error
		RepoUpdateError2   error
		ExpectedStatusCode int
		ExpectedErrorMsg   string
	}{
		{"happy path", &expected, nil, nil, nil, nil, http.StatusOK, ""},
		{"no file specified", &noFileExpected, nil, nil, nil, nil, http.StatusBadRequest, "error accessing file"},
		{"bad request body", &types.Job{}, nil, nil, nil, nil, http.StatusBadRequest, "unexpected end of JSON input"},
		{"job repo update failed", &expected, errors.New("update failed"), nil, nil, nil, http.StatusInternalServerError, "update failed"},
		{"transmit job failed", &expected, nil, errors.New("transmit job failed"), nil, nil, http.StatusOK, ""},
		{"transmit job failed - file receiver gateway down", &expected, nil, fmt.Errorf(pkg.ErrFmtTcpLookup, pkg.OwnerFileRecvGateway), nil, nil, http.StatusOK, fmt.Sprintf("dial tcp: lookup %s", pkg.OwnerFileRecvGateway)},
		{"transmit file failed", &expected, nil, nil, errors.New("transmit file failed"), nil, http.StatusOK, ""},
		{"transmit file failed - file receiver gateway down", &expected, nil, nil, fmt.Errorf(pkg.ErrFmtTcpLookup, pkg.OwnerFileRecvGateway), nil, http.StatusOK, fmt.Sprintf("dial tcp: lookup %s", pkg.OwnerFileRecvGateway)},
		{"transmit file repo update failed", &expected, nil, nil, nil, errors.New("update failed"), http.StatusOK, ""},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var requestBody []byte
			repoMock := jobRepoMocks.Client{}
			receiverMock := fileReceiverMocks.Client{}
			fileHandler := New(logger.MockLogger{}, &repoMock, &receiverMock, fileHostname, dependentServices)
			if test.Expected.Id != "" {
				requestBody, _ = json.Marshal(test.Expected)
			} else {
				requestBody = nil
			}
			req := httptest.NewRequest("POST", "http://localhost", bytes.NewReader(requestBody))
			w := httptest.NewRecorder()
			jobFields := make(map[string]interface{})
			jobFields[types.JobOwner] = pkg.OwnerFileSenderOem
			repoMock.On("Update", mock.Anything, jobFields).Return(types.Job{}, test.RepoUpdateError1)
			receiverMock.On("TransmitJob", mock.Anything).Return(test.TransmitJobError)
			receiverMock.On("TransmitFile", mock.Anything, mock.Anything).Return(0, test.TransmitFileError)
			failedTransmissionFields := make(map[string]interface{})
			failedTransmissionFields[types.JobStatus] = pkg.StatusTransmissionFailed
			if test.TransmitFileError != nil {
				failedTransmissionFields[types.JobErrorDetailsErrorMsg] = pkg.ErrFileTransmitting.Error()
			} else {
				failedTransmissionFields[types.JobErrorDetailsErrorMsg] = pkg.ErrTransmitJob.Error()
			}
			failedTransmissionFields[types.JobErrorDetailsOwner] = pkg.OwnerFileSenderOem
			repoMock.On("Update", mock.Anything, failedTransmissionFields).Return(types.Job{}, test.RepoUpdateError2)

			fileHandler.SendNewFile(w, req)
			resp := w.Result()
			defer resp.Body.Close()

			require.Equal(t, test.ExpectedStatusCode, resp.StatusCode, "invalid status code")
			if test.ExpectedStatusCode != http.StatusOK {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				require.Contains(t, string(body), test.ExpectedErrorMsg)
				return
			}

			receiverMock.AssertCalled(t, "TransmitJob", mock.Anything)
			if test.TransmitJobError == nil {
				receiverMock.AssertCalled(t, "TransmitFile", mock.Anything, mock.Anything)
			}
		})
	}
}
