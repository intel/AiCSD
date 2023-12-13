/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"aicsd/as-pipeline-val/clients"
	"aicsd/as-pipeline-val/config"
	"aicsd/as-pipeline-val/types"

	appsdk "github.com/edgexfoundry/app-functions-sdk-go/v3/pkg"
	"github.com/edgexfoundry/app-functions-sdk-go/v3/pkg/interfaces/mocks"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/common"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestController_LaunchEventForPipeline(t *testing.T) {

	validInput := types.LaunchInfo{
		InputFileLocation: "./test.tiff",
		PipelineTopic:     "MultiFile",
		OutputFileFolder:  "/tmp/output/files",
		ModelParams:       map[string]string{"key1": "value1"},
	}

	tests := []struct {
		Name               string
		InputInfo          *types.LaunchInfo
		PublisherErr       error
		ExpectedStatusCode int
		ExpectedErrorMsg   string
	}{
		{"happy path", &validInput, nil, http.StatusCreated, ""},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var requestBody []byte
			backgroundPublisherMock := mocks.BackgroundPublisher{}
			appServiceMock := mocks.ApplicationService{}
			mockLogger := logger.MockLogger{}
			kpSimController := New(&mockLogger, &backgroundPublisherMock, &appServiceMock, &config.Configuration{}, clients.NewClient("", 10))

			requestBody, err := json.Marshal(test.InputInfo)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "http://localhost", bytes.NewReader(requestBody))
			w := httptest.NewRecorder()

			ctx := appsdk.NewAppFuncContextForTest(uuid.NewString(), mockLogger)
			appServiceMock.On("BuildContext", mock.Anything, common.ContentTypeJSON).Return(ctx)
			backgroundPublisherMock.On("Publish", mock.Anything, ctx).Return(test.PublisherErr)

			kpSimController.LaunchEventForPipeline(w, req)
			resp := w.Result()

			require.Equal(t, test.ExpectedStatusCode, resp.StatusCode, "invalid status code")
			appServiceMock.AssertExpectations(t)
			backgroundPublisherMock.AssertExpectations(t)
		})
	}
}
