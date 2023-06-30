/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package functions

import (
	"aicsd/as-pipeline-sim/config"
	"aicsd/pkg/types"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"aicsd/pkg/helpers"

	"github.com/edgexfoundry/app-functions-sdk-go/v2/pkg/interfaces"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/dtos"

	"aicsd/pkg"
	"aicsd/pkg/werrors"
)

const ResourceName = "PipelineParameters"

// PipelineSim provides the data for the App Function Pipelines in this package
type PipelineSim struct {
	params          *PipelineParams
	lc              logger.LoggingClient
	outputFiles     []types.OutputFile
	pipelineResults string
	qcFlags         string
	config          *config.Configuration
}

type pipelineBodyResp struct {
	Name        string  `json:"name"`
	Probability float32 `json:"probability"`
}

func NewPipelineSim(pipelineResults string, qcFlags string, config *config.Configuration) PipelineSim {
	return PipelineSim{
		pipelineResults: pipelineResults,
		qcFlags:         qcFlags,
		config:          config,
	}
}

// PipelineParams is the data expected in the EdgeX Reading objectValue
type PipelineParams struct {
	InputFileLocation string
	OutputFileFolder  string
	ModelParams       map[string]string
	JobUpdateUrl      string // already includes the parameterized jobid and taskid values
	PipelineStatusUrl string // already includes the parameterized jobid and taskid values
}

// ProcessEvent is the entry point App Pipeline Function that receives and processes the EdgeX Event/Reading that
// contains the Pipeline Parameters
func (p *PipelineSim) ProcessEvent(ctx interfaces.AppFunctionContext, data interface{}) (bool, interface{}) {
	p.lc = ctx.LoggingClient()
	p.lc.Debugf("Running ProcessEvent...")

	if data == nil {
		err := fmt.Errorf("no data received")
		p.lc.Errorf("ProcessEvent failed: %s", err.Error())
		return false, err // Terminate since can not send back status w/o knowing the URL that is passed in the data
	}

	event, ok := data.(dtos.Event)
	if !ok {
		err := fmt.Errorf("type received is not an Event")
		p.lc.Errorf("ProcessEvent failed: %s", err.Error())
		return false, err // Terminate since can not send back status w/o knowing the URL that is passed in the data
	}

	err := helpers.AppFunctionEventValidation(event, ResourceName, ResourceName)
	if err != nil {
		return true, werrors.WrapMsg(err, "failed AppFunctionEventValidation")
	}

	jsonData, err := json.Marshal(event.Readings[0].ObjectValue)
	if err != nil {
		err = fmt.Errorf("unable to marshal Object Value back to JSON: %s", err.Error())
		p.lc.Errorf("ProcessEvent failed: %s", err.Error())
		return false, err // Terminate since can not send back status w/o knowing the URL that is passed in the data
	}

	params := PipelineParams{}
	if err = json.Unmarshal(jsonData, &params); err != nil {
		err = fmt.Errorf("unable to unmarshal Object Value back from JSON to struct: %s", err.Error())
		p.lc.Errorf("ProcessEvent failed: %s", err.Error())
		return false, err // Terminate since can not send back status w/o knowing the URL that is passed in the data
	}

	p.params = &params

	ctx.LoggingClient().Debugf("Pipeline %s: Received the following PipelineParams: %+v", ctx.PipelineId(), params)

	return true, data // All is good, this indicates success for the next function
}

// CreateSimulatedOutputFile is an App Pipeline Function that creates the simulated output file by copying the
// input file to new name in the specified output folder
func (p *PipelineSim) CreateSimulatedOutputFile(ctx interfaces.AppFunctionContext, data interface{}) (bool, interface{}) {
	p.lc.Debugf("Running CreateSimulatedOutputFile...")

	// Previous function returns any error as its result which is passed as `data` to this function.
	if pipelineErr, ok := data.(error); ok {
		p.lc.Info("CreateSimulatedOutputFile: Forwarding error to next function")
		return true, pipelineErr // Keep passing the error forward inorder for to make it to the ReportStatus pipeline function
	}

	_, filename := path.Split(p.params.InputFileLocation)
	extension := filepath.Ext(filename)
	outputFilename := strings.Replace(filename, extension, "", 1) + "-sim" + extension
	p.outputFiles = []types.OutputFile{types.CreateOutputFile(p.params.OutputFileFolder, outputFilename, extension, "", "", "", "", nil)}

	contents, err := os.ReadFile(p.params.InputFileLocation)
	if err != nil {
		p.lc.Errorf("CreateSimulatedOutputFile ReadFile failed: %s", err.Error())
		return true, err
	}

	err = os.WriteFile(filepath.Join(p.params.OutputFileFolder, p.outputFiles[0].Name), contents, pkg.FilePermissions)
	if err != nil {
		p.lc.Errorf("CreateSimulatedOutputFile WriteFile failed: %s", err.Error())
		return true, err
	}

	p.lc.Debugf("Pipeline %s: Simulated output file %s created", ctx.PipelineId(), p.outputFiles[0])

	return true, data // All is good, this indicates success for the next function
}

// CreateSimulatedMultiOutputFiles is an App Pipeline Function that creates multiple simulated output files by copying the
// input file to new name in the specified output folder
func (p *PipelineSim) CreateSimulatedMultiOutputFiles(ctx interfaces.AppFunctionContext, data interface{}) (bool, interface{}) {
	p.lc.Debugf("Running CreateSimulatedMultiOutputFiles...")

	// Previous function returns any error as its result which is passed as `data` to this function.
	if pipelineErr, ok := data.(error); ok {
		p.lc.Info("CreateSimulatedMultiOutputFiles: Forwarding error to next function")
		return true, pipelineErr // Keep passing the error forward inorder for to make it to the ReportStatus pipeline function
	}

	_, filename := path.Split(p.params.InputFileLocation)
	extension := filepath.Ext(filename)

	contents, err := os.ReadFile(p.params.InputFileLocation)
	if err != nil {
		p.lc.Errorf("CreateSimulatedMultiOutputFiles failed: %s", err.Error())
		return true, err
	}

	for i := 0; i < pkg.LoopForMultiOutputFiles; i++ {
		outputFilename := strings.Replace(filename, extension, "", 1) + "-sim" + fmt.Sprintf("%d", i) + extension
		p.lc.Debugf("fixing to write output file: %s", outputFilename)
		simOutputFile := types.CreateOutputFile(p.params.OutputFileFolder, outputFilename, extension, "", "", "", "", nil)
		p.outputFiles = append(p.outputFiles, simOutputFile)
		err = os.WriteFile(filepath.Join(p.params.OutputFileFolder, simOutputFile.Name), contents, pkg.FilePermissions)
		if err != nil {
			p.lc.Errorf("CreateSimulatedMultiOutputFiles %s failed writing the file: %s", outputFilename, err.Error())
			return true, err
		}
		p.lc.Debugf("Pipeline %s: Simulated output file %s created with full path of %s", ctx.PipelineId(), outputFilename, simOutputFile.Name)
	}
	p.lc.Debugf("here are my multiple output files %v", p.outputFiles)

	return true, data // All is good, this indicates success for the next function
}

// UpdateJobRepo is an App Pipeline Function that does the PUT request to update the Job Repo with the Pipeline status details
func (p *PipelineSim) UpdateJobRepo(ctx interfaces.AppFunctionContext, data interface{}) (bool, interface{}) {
	p.lc.Debugf("Running UpdateJobRepo...")

	pipelineStatus := struct {
		Status      string
		QCFlags     string
		OutputFiles []types.OutputFile
		Results     string
	}{
		Status:      pkg.TaskStatusComplete,
		QCFlags:     p.qcFlags,
		OutputFiles: p.outputFiles,
		Results:     p.pipelineResults,
	}

	// Previous function returns any error as its result which is passed as `data` to this function.
	if _, ok := data.(error); ok {
		pipelineStatus.Status = pkg.TaskStatusFailed
		pipelineStatus.Results = ""
		pipelineStatus.QCFlags = ""
		pipelineStatus.OutputFiles = []types.OutputFile{}
	}

	body, err := json.Marshal(pipelineStatus)
	if err != nil {
		err = werrors.WrapMsg(err, "failed to marshal Pipeline Status for updating Job Repo.")
		p.lc.Errorf("UpdateJobRepo failed: %s", err.Error())
		return true, err
	}

	request, err := http.NewRequest(http.MethodPut, p.params.JobUpdateUrl, bytes.NewBuffer(body))
	if err != nil {
		err = werrors.WrapMsg(err, "unable to create http request to update Job Repo.")
		p.lc.Errorf("UpdateJobRepo failed: %s", err.Error())
		return true, err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		err = werrors.WrapMsg(err, "failed to send update Job Repo request.")
		p.lc.Errorf("UpdateJobRepo failed: %s", err.Error())
		return true, err
	}

	if response.StatusCode != http.StatusOK {
		err := fmt.Errorf("update Job Repo request failed with status code %d", response.StatusCode)
		p.lc.Errorf("UpdateJobRepo failed: %s", err.Error())
		return true, err
	}

	p.lc.Debugf("Pipeline %s: Updated Pipeline details with '%+v' using URL %s", ctx.PipelineId(), pipelineStatus, p.params.JobUpdateUrl)

	return true, data // Pass data which was passed to this function (which could be error) on to next function
}

// ReportStatus is an App Pipeline Function that does the POST request to Task Launcher for PipelineStatus
func (p *PipelineSim) ReportStatus(ctx interfaces.AppFunctionContext, data interface{}) (bool, interface{}) {
	p.lc.Debugf("Running ReportStatus...")

	// Previous function returns any error as its result which is passed as `data` to this function.
	status := pkg.TaskStatusComplete
	if _, ok := data.(error); ok {
		status = pkg.TaskStatusFailed
		p.lc.Errorf("Pipeline failed. Sending %s status to %s", status, p.params.PipelineStatusUrl)
	}

	request, err := http.NewRequest(http.MethodPost, p.params.PipelineStatusUrl, bytes.NewBuffer([]byte(status)))
	if err != nil {
		err = werrors.WrapMsg(err, "failed to create http request to send pipeline status.")
		p.lc.Errorf("ReportStatus failed: %s", err.Error())
		return false, err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		err = werrors.WrapMsg(err, "failed to send pipeline status request.")
		p.lc.Errorf("ReportStatus failed: %s", err.Error())
		return false, err
	}

	if response.StatusCode != http.StatusOK {
		err := fmt.Errorf("pipeline status request failed with status code %d", response.StatusCode)
		p.lc.Errorf("ReportStatus failed: %s", err.Error())
		return false, err
	}

	// Note: This is needed to clear the output files that the pipeline sim should process on future runs
	p.outputFiles = []types.OutputFile{}

	p.lc.Debugf("Pipeline %s: Pipeline Status complete using URL %s", ctx.PipelineId(), p.params.PipelineStatusUrl)

	return false, nil // All done
}

func (p *PipelineSim) TriggerGetiPipeline(ctx interfaces.AppFunctionContext, data interface{}) (bool, interface{}) {
	p.lc.Debugf("Running TriggerGetiPipeline...")

	pipelineTopic, _ := ctx.GetValue("receivedtopic")

	_, filename := path.Split(p.params.InputFileLocation)
	extension := filepath.Ext(filename)

	outputFilename := strings.Replace(filename, extension, "", 1) + "-Geti" + extension
	p.outputFiles = []types.OutputFile{types.CreateOutputFile(p.params.OutputFileFolder, outputFilename, extension, "", "", "", "", nil)}
	p.outputFiles[0].Extension = extension

	outputFilenamePath := p.params.OutputFileFolder + "/" + outputFilename

	pipelineParams := struct {
		InputFileLocation string
		OutputFileFolder  string
		ModelName         string
	}{
		InputFileLocation: p.params.InputFileLocation,
		OutputFileFolder:  outputFilenamePath,
		ModelName:         strings.TrimLeft(pipelineTopic, "geti/"),
	}

	body, err := json.Marshal(pipelineParams)
	if err != nil {
		err = werrors.WrapMsg(err, "failed to marshal pipelineParams to send to Geti pipeline.")
		p.lc.Errorf("TriggerGetiPipeline failed: %s", err.Error())
		return true, err
	}

	request, err := http.NewRequest(http.MethodPost, p.config.GetiUrl, bytes.NewBuffer([]byte(body)))
	if err != nil {
		err = werrors.WrapMsg(err, "failed to create http request to trigger Geti pipeline.")
		p.lc.Errorf("TriggerGetiPipeline failed: %s", err.Error())
		return true, err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		err = werrors.WrapMsg(err, "failed to trigger Geti pipeline request.")
		p.lc.Errorf("TriggerGetiPipeline failed: %s", err.Error())
		return true, err
	}

	if response.StatusCode != http.StatusOK {
		err := fmt.Errorf("trigger Geti pipeline request failed with status code %d", response.StatusCode)
		p.lc.Errorf("TriggerGetiPipeline failed: %s", err.Error())
		return true, err
	}

	resbody, err := io.ReadAll(response.Body)
	if err != nil {
		err := werrors.WrapMsg(err, "failed to read response from Geti pipeline")
		p.lc.Errorf("TriggerGetiPipeline failed: %s", err.Error())
		return true, err
	}

	var pipelineResp pipelineBodyResp
	err = json.Unmarshal(resbody, &pipelineResp)
	if err != nil {
		err = werrors.WrapMsgf(err, "unable to unmarshal data from Geti response")
		p.lc.Errorf("TriggerGetiPipeline failed: %s", err.Error())
		return true, err
	}

	output, err := json.Marshal(pipelineResp)
	if err != nil {
		err = werrors.WrapMsgf(err, "unable to marshal data from Geti response")
		p.lc.Errorf("TriggerGetiPipeline failed: %s", err.Error())
		return true, err
	}

	p.pipelineResults = string(output)

	return true, data
}
