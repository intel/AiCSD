/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package helpers

import (
	"aicsd/pkg"
	"aicsd/pkg/clients/job_repo"
	"aicsd/pkg/types"
	"aicsd/pkg/werrors"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/edgexfoundry/go-mod-core-contracts/v2/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/dtos"
	"github.com/google/uuid"

	"github.com/diegoholiveira/jsonlogic"
	"github.com/edgexfoundry/app-functions-sdk-go/v2/pkg/interfaces"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/logger"
	"github.com/gorilla/mux"
)

const (
	MqttResultsTopic = "pipeline/params"
	CustomTopic      = "pipeline-params"
)

// HandleErrorMessage is a helper function to log the error message and send it to the writer with the given http status.
func HandleErrorMessage(lc logger.LoggingClient, writer http.ResponseWriter, err error, httpStatus int) {
	lc.Error(err.Error())
	http.Error(writer, err.Error(), httpStatus)
}

// GetAppSetting takes a key and parse the Application Setting for that keyword and return it
func GetAppSetting(service interfaces.ApplicationService, key string, canBeEmpty bool) (string, error) {
	field, err := service.GetAppSetting(key)
	if err != nil {
		return "", err
	}
	if len(strings.TrimSpace(field)) == 0 && !canBeEmpty {
		return "", fmt.Errorf("config field %s cannot be empty", key)
	}
	return field, nil
}

// GetUrlFromAppSetting takes a keyword and parse the Application Setting for the Host and Port.
// It then puts the url together and tests it before returning.
func GetUrlFromAppSetting(service interfaces.ApplicationService, key string, protocol string, canBeEmpty bool) (string, error) {
	host, err := GetAppSetting(service, fmt.Sprintf("%sHost", key), canBeEmpty)
	if err != nil {
		return "", err
	}
	port, err := GetAppSetting(service, fmt.Sprintf("%sPort", key), canBeEmpty)
	if err != nil {
		return "", err
	}
	keyUrl := fmt.Sprintf("%s://%s:%s", protocol, host, port)
	_, err = url.Parse(keyUrl)
	if err != nil {
		return "", fmt.Errorf("invalid URL for %s: %s, %s", key, keyUrl, err.Error())
	}
	return keyUrl, nil
}

// UnmarshalTask reads the httprequest, and tries to unmarshal the request into a task.
// It returns the task, the http status and the error if one occurred
func UnmarshalTask(request *http.Request) (types.Task, int, error) {
	var err error
	var entry types.Task

	// read the request body
	requestBody := make([]byte, request.ContentLength)
	_, err = io.ReadFull(request.Body, requestBody)

	if err != nil {
		return types.Task{}, http.StatusInternalServerError, werrors.WrapMsgf(err, pkg.ErrFmtProcessingReq, request.URL.String())
	}

	err = json.Unmarshal(requestBody, &entry)
	if err != nil {
		return types.Task{}, http.StatusBadRequest, fmt.Errorf("failed to unmarshal request (%s) to task: %s",
			request.URL.String(), err.Error())
	}
	return entry, http.StatusOK, nil
}

// UnmarshalJob reads the httprequest, and tries to unmarshal the request into a job.
// It returns the job, the http status and the error if one occurred
func UnmarshalJob(request *http.Request) (types.Job, int, error) {
	var err error
	var job types.Job

	// read the request body
	requestBody := make([]byte, request.ContentLength)
	_, err = io.ReadFull(request.Body, requestBody)
	if err != nil {
		return types.Job{}, http.StatusInternalServerError, werrors.WrapMsgf(err, pkg.ErrFmtProcessingReq, request.URL.String())
	}

	err = json.Unmarshal(requestBody, &job)
	if err != nil {
		return types.Job{}, http.StatusBadRequest, fmt.Errorf("failed to unmarshal request (%s): %s",
			request.URL.String(), err.Error())
	}
	return job, http.StatusOK, nil
}

// GetByKeyFromRequest is used to extract the value given the key from the URL variables
func GetByKeyFromRequest(request *http.Request, key string) (string, error) {
	vars := mux.Vars(request)
	value, ok := vars[key]
	if !ok {
		return "", fmt.Errorf("missing %s in url", key)
	}
	if value == "" {
		return "", fmt.Errorf("empty %s in url", key)
	}
	return value, nil
}

// ApplyJsonLogicToJob takes a job and logic string that evaluates to true or false
// It returns a bool if the logic result is a bool and an error otherwise
func ApplyJsonLogicToJob(job types.Job, logic string) (bool, error) {
	var logicResult bytes.Buffer
	jsonJob, err := json.Marshal(job)
	if err != nil {
		return false, werrors.WrapErr(err, pkg.ErrMarshallingJob)
	}
	data := strings.NewReader(string(jsonJob))
	selector := strings.NewReader(logic)
	err = jsonlogic.Apply(selector, data, &logicResult)
	if err != nil {
		return false, werrors.WrapErr(err, pkg.ErrJobInvalid)
	}
	isValid, err := strconv.ParseBool(string(bytes.TrimSpace(logicResult.Bytes())))
	if err != nil {
		return false, werrors.WrapMsgf(err, "could not parse bool from json logic result, got %s", logicResult.String())
	}
	return isValid, nil
}

// AppFunctionEventValidation performs the dtos.Event validation checks necessary for the application services,
// and returns an error if the event is invalid.
func AppFunctionEventValidation(event dtos.Event, sourceName, resourceName string) error {
	if event.SourceName != sourceName {
		return werrors.WrapMsgf(pkg.ErrInvalidInput, pkg.ErrFmtInvalidInput, event.SourceName, sourceName)
	}

	if len(event.Readings) != 1 {
		return werrors.WrapMsgf(pkg.ErrEmptyInput,
			"event must have 1 reading. Found '%d' readings", len(event.Readings))
	}

	if event.Readings[0].ResourceName != resourceName {
		return werrors.WrapMsgf(pkg.ErrInvalidInput, pkg.ErrFmtInvalidInput,
			event.Readings[0].ResourceName, resourceName)
	}

	if event.Readings[0].ValueType != common.ValueTypeObject {
		return werrors.WrapMsgf(pkg.ErrInvalidInput, pkg.ErrFmtInvalidInput,
			event.Readings[0].ValueType, common.ValueTypeObject)
	}

	return nil
}

// IsNetworkError passes in a non nil error to see if it is a network error or not
func IsNetworkError(err error) bool {
	switch err.(type) {
	case *net.DNSError, *net.OpError, syscall.Errno, net.Error:
		return true
	}
	return false
}

// UpdateJobFields is a job specific helper to update the job repository with job changes
func UpdateJobFields(jobRepoClient job_repo.Client, jobId, owner, status, pipelineStatus string, errDetails *pkg.UserFacingError, inputArchiveName string, inputViewableName string, outputFiles []types.OutputFile) (types.Job, error) {
	jobFields := make(map[string]interface{})
	if owner != "" {
		jobFields[types.JobOwner] = owner
	}
	if status != "" {
		jobFields[types.JobStatus] = status
	}
	if pipelineStatus != "" {
		jobFields[types.JobPipelineStatus] = pipelineStatus
	}
	if errDetails != nil && errDetails.Error != "" {
		jobFields[types.JobErrorDetailsOwner] = errDetails.Owner
		jobFields[types.JobErrorDetailsErrorMsg] = errDetails.Error
	}
	if inputArchiveName != "" {
		jobFields[types.JobInputArchiveName] = inputArchiveName
	}
	if inputViewableName != "" {
		jobFields[types.JobInputViewableName] = inputViewableName
	}
	if outputFiles != nil {
		jobFields[types.JobPipelineOutputFiles] = outputFiles
	}

	//TODO: add retry logic here?
	return jobRepoClient.Update(jobId, jobFields)
}

// PublishEventForPipeline publishes an event containing the job information to the EdgeX Message Bus.
// It uses the information in the task passed in to determine the publish topic and fill in the callback urls.
func PublishEventForPipeline(publisher interfaces.BackgroundPublisher, service interfaces.ApplicationService, lc logger.LoggingClient, job types.Job, matchedTask *types.Task, jobUpdateBaseUrl string, pipelineStatusBaseUrl string, deviceProfileName string, deviceName string, resourceName string) error {
	var resultFolder string
	// create struct to add PipelineParams: JobId, TaskId, InputFileLocation, OutputFileFolder, ModelParameters
	if matchedTask.ResultFileFolder == "" {
		resultFolder = job.InputFile.DirName
	} else {
		resultFolder = matchedTask.ResultFileFolder
	}
	pipelineParams := struct {
		InputFileLocation string
		OutputFileFolder  string
		ModelParams       map[string]string
		JobUpdateUrl      string
		PipelineStatusUrl string
	}{
		InputFileLocation: filepath.Join(job.InputFile.DirName, job.InputFile.Name),
		OutputFileFolder:  resultFolder,
		ModelParams:       matchedTask.ModelParameters,
		JobUpdateUrl:      fmt.Sprintf("%s%s", jobUpdateBaseUrl, pkg.EndpointJobPipeline),
		PipelineStatusUrl: fmt.Sprintf("%s%s", pipelineStatusBaseUrl, pkg.EndpointPipelineStatus),
	}

	jobIdPlaceholder := fmt.Sprintf("{%s}", pkg.JobIdKey)
	taskIdPlaceholder := fmt.Sprintf("{%s}", pkg.TaskIdKey)
	pipelineParams.JobUpdateUrl = strings.ReplaceAll(pipelineParams.JobUpdateUrl, jobIdPlaceholder, job.Id)
	pipelineParams.JobUpdateUrl = strings.ReplaceAll(pipelineParams.JobUpdateUrl, taskIdPlaceholder, job.PipelineDetails.TaskId)
	pipelineParams.PipelineStatusUrl = strings.ReplaceAll(pipelineParams.PipelineStatusUrl, jobIdPlaceholder, job.Id)
	pipelineParams.PipelineStatusUrl = strings.ReplaceAll(pipelineParams.PipelineStatusUrl, taskIdPlaceholder, job.PipelineDetails.TaskId)

	lc.Debugf("Pipeline Topic to publish on is: %s", matchedTask.PipelineId)
	if strings.Contains(matchedTask.PipelineId, "ovms/") {

		//publish event for pipelines with models served via ovms/bentoml service
		err := publishPipielineParametersForJob(service, publisher, lc, pipelineParams, job, matchedTask.PipelineId)
		if err != nil {
			return fmt.Errorf("could not publish event for job id %s, task id %s: %s", job.Id, job.PipelineDetails.TaskId, err.Error())
		}

		lc.Debugf("Publish Event to %s for JobId=%s, TaskId=%s and File=%s", matchedTask.PipelineId, job.Id, matchedTask.Id, job.FullInputFileLocation())
		return nil
	}

	// create event reading for resources
	myEvent := dtos.NewEvent(deviceProfileName, deviceName, resourceName)
	timestamp := time.Now().UTC().UnixNano()
	myEvent.Origin = timestamp
	myEvent.AddObjectReading(resourceName, pipelineParams)
	jsonData, err := json.Marshal(myEvent)
	if err != nil {
		return fmt.Errorf("could not marshal event to publish for job id %s, task id %s: %s", job.Id, job.PipelineDetails.TaskId, err.Error())
	}

	// publish event
	ctx := service.BuildContext(uuid.NewString(), common.ContentTypeJSON)
	ctx.AddValue(pkg.PublishTopicKey, matchedTask.PipelineId)
	err = publisher.Publish(jsonData, ctx)
	if err != nil {
		return fmt.Errorf("could not publish event for job id %s, task id %s: %s", job.Id, job.PipelineDetails.TaskId, err.Error())
	}

	lc.Debugf("Publish Event to %s for JobId=%s, TaskId=%s and File=%s", matchedTask.PipelineId, job.Id, matchedTask.Id, job.FullInputFileLocation())
	return nil

}

// publishPipielineParametersForJob publishes the pipeline parameters to the EdgeX Message Bus to the pipeline topic
func publishPipielineParametersForJob(service interfaces.ApplicationService, publisher interfaces.BackgroundPublisher, lc logger.LoggingClient, pipelineParameters types.PipelineParameters, job types.Job, pipelineId string) error {

	ctx := service.BuildContext(uuid.NewString(), common.ContentTypeJSON)
	ctx.AddValue(pkg.PublishTopicKey, MqttResultsTopic)
	ctx.AddValue(pkg.CustomTopicKey, CustomTopic)

	msg, err := json.Marshal(pipelineParameters)
	if err != nil {
		return fmt.Errorf("could not marshal pipeline parameters to publish for job id %s, task id %s: %s", job.Id, job.PipelineDetails.TaskId, err.Error())
	}

	err = publisher.Publish(msg, ctx)
	if err != nil {
		return fmt.Errorf("could not publish pipeline parameters for job id %s, job input file %s: %s", job.Id, job.FullInputFileLocation(), err.Error())
	}
	lc.Debugf("Publish pipeline parameters for job id=%s, input file=%s", job.Id, job.FullInputFileLocation())
	return nil
}

// CopyFile copies a file from the provided source to the provided destination
// It also uses a given logger to log errors
func CopyFile(lc logger.LoggingClient, src string, dst string) {
	// OOM Error if the file is too large
	data, err := os.ReadFile(src)
	if err != nil {
		lc.Error(err.Error())
	}

	err = os.WriteFile(dst, data, 0644)
	if err != nil {
		lc.Error(err.Error())
	}
}

// DirectoryIsEmpty checks if a given directory is empty and returns a boolean.
// It also return an error if an error occurs.
func DirectoryIsEmpty(dir string) (bool, error) {
	f, err := os.Open(dir)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}
