/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package controller

import (
	"aicsd/ms-data-organizer/clients/task_launcher"
	"aicsd/pkg/wait"
	"aicsd/pkg/werrors"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/edgexfoundry/app-functions-sdk-go/v2/pkg/interfaces"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/dtos"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/sunshineplan/imgconv"

	taskPkg "aicsd/as-task-launcher/pkg"
	"aicsd/pkg"
	"aicsd/pkg/clients/job_repo"
	"aicsd/pkg/helpers"
	"aicsd/pkg/types"
)

type Controller struct {
	lc                 logger.LoggingClient
	fileHostname       string
	jobRepoClient      job_repo.Client
	taskLauncherClient task_launcher.Client
	publisher          interfaces.BackgroundPublisher
	service            interfaces.ApplicationService
	jobMap             map[string]types.Job
	archiveFolder      string
	rejectFolder       string
	DependentServices  wait.Services
}

func New(lc logger.LoggingClient, jobRepoClient job_repo.Client, taskRepoClient task_launcher.Client, publisher interfaces.BackgroundPublisher, service interfaces.ApplicationService, fileHostname string, archiveFolder string, rejectFolder string) (*Controller, error) {
	c := &Controller{
		lc:                 lc,
		jobRepoClient:      jobRepoClient,
		taskLauncherClient: taskRepoClient,
		fileHostname:       fileHostname,
		publisher:          publisher,
		service:            service,
		jobMap:             make(map[string]types.Job),
		archiveFolder:      archiveFolder,
		rejectFolder:       rejectFolder,
		DependentServices:  wait.Services{wait.ServiceConsul, wait.ServiceRedis, wait.ServiceJobRepo},
	}
	err := c.initJobMap()
	return c, err
}

// RegisterRoutes is a function to register the necessary endpoints for the controller.
// It returns an error if an error occurred and nil if no errors occurred
func (c *Controller) RegisterRoutes(service interfaces.ApplicationService) error {
	err := service.AddRoute(pkg.EndpointDataToHandle, c.HandleNewJob, http.MethodPost)
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, pkg.EndpointDataToHandle)
	}
	err = service.AddRoute(pkg.EndpointTransmitFileJobId, c.TransmitFile, http.MethodGet)
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, pkg.EndpointTransmitFileJobId)
	}
	err = service.AddRoute(pkg.EndpointRetry, c.retry, http.MethodPost)
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, pkg.EndpointRetry)
	}
	err = service.AddRoute(pkg.EndpointArchiveFile, c.ArchiveFile, http.MethodPost)
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, pkg.EndpointArchiveFile)
	}
	err = service.AddRoute(pkg.EndpointRejectFile, c.RejectFile, http.MethodPost)
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, pkg.EndpointRejectFile)
	}
	err = service.AddRoute(pkg.EndpointRejectFile, c.RejectFile, http.MethodDelete)
	if err != nil {
		return werrors.WrapMsgf(err, pkg.ErrFmtRegisterRoutes, pkg.EndpointRejectFile)
	}
	return nil
}

// RejectFile is called when the user rejects a job that is not rejected (POST)
// or when the user unrejects a job that is rejected (DELETE).
// The function copies an image from the archive folder to the rejected folder.
func (c *Controller) RejectFile(writer http.ResponseWriter, request *http.Request) {
	jobId, err := helpers.GetByKeyFromRequest(request, pkg.JobIdKey)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, err, http.StatusBadRequest)
		return
	}

	job, err := c.jobRepoClient.RetrieveById(jobId)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, fmt.Errorf("%s for JobId: %s", pkg.ErrJobInvalid, jobId), http.StatusInternalServerError)
		return
	}

	taskId := job.PipelineDetails.TaskId
	task, err := c.taskLauncherClient.RetrieveById(taskId)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, fmt.Errorf("%s for TaskId: %s, Error: %s", taskPkg.ErrTaskInvalid, taskId, err), http.StatusInternalServerError)
		return
	}

	modelName := task.PipelineId
	if strings.Contains(modelName, "/") {
		split := strings.Split(modelName, "/")
		modelName = split[len(split)-1]
	}

	src := job.InputFile.ArchiveName
	dstDir := path.Join(c.rejectFolder, modelName)
	dst := path.Join(dstDir, job.InputFile.Name)

	dirExists := true
	if _, err := os.Stat(dstDir); os.IsNotExist(err) {
		dirExists = false
	}

	if _, err := os.Stat(dst); errors.Is(err, os.ErrNotExist) && request.Method == http.MethodPost {
		if !dirExists {
			os.Mkdir(dstDir, os.FileMode(0777))
			os.Chmod(dstDir, os.FileMode(0777))
		}
		helpers.CopyFile(c.lc, src, dst)
		os.Chmod(dst, os.FileMode(0777))
	} else if request.Method == http.MethodDelete {
		err = os.Remove(dst)
		if err != nil {
			helpers.HandleErrorMessage(c.lc, writer, fmt.Errorf("%s for JobId: %s", pkg.ErrFileDeletingReject, jobId), http.StatusInternalServerError)
		}

		if ok, err := helpers.DirectoryIsEmpty(dstDir); ok && err == nil {
			err := os.RemoveAll(dstDir)
			if err != nil {
				helpers.HandleErrorMessage(c.lc, writer, fmt.Errorf("failed to remove empty directory %s", dstDir), http.StatusInternalServerError)
			}
		}
	}
}

// RetryOnStartup will be called on startup to look at what job objects the file sender gateway owns and attempts to
// process them. The function checks that the job host name matches a pipeline and will send the data onto
// the message bus for the file receiver oem.
// If an err is encountered, then the owner is set to none and the job is updated.
func (c *Controller) RetryOnStartup() error {
	var err, errs error
	// TODO: add retry logic around the Retrieve call in case job repo is not up yet.
	err = c.refreshJobMap()
	if err != nil {
		return werrors.WrapErr(err, pkg.ErrRetrieving)
	}
	c.lc.Debugf("jobs map on startup with refresh: %v", c.jobMap)

	for _, job := range c.jobMap {
		c.lc.Debugf("processing job: %s on restart", job.FullInputFileLocation())

		// publish job object to message bus
		ctx := c.service.BuildContext(uuid.NewString(), common.ContentTypeJSON)
		publishErr := c.publishJob(job, ctx)
		if publishErr != nil {
			// Note: owner is set as file sender gw to enable the job to be retried upon startup
			jobErr := pkg.CreateUserFacingError(pkg.OwnerFileSenderGateway, pkg.ErrPublishing)
			_, updateErr := helpers.UpdateJobFields(c.jobRepoClient, job.Id, pkg.OwnerFileSenderGateway, pkg.StatusTransmissionFailed, "", jobErr, "", "", nil)
			if updateErr != nil {
				errs = multierror.Append(errs, werrors.WrapErr(updateErr, pkg.ErrUpdating))
				continue
			}
			errs = multierror.Append(errs, fmt.Errorf("%s for job %s: %s", pkg.ErrPublishing, job.FullInputFileLocation(), err.Error()))
			continue
		}

		// no issues with job so update job fields
		_, updateErr := helpers.UpdateJobFields(c.jobRepoClient, job.Id, pkg.OwnerFileSenderGateway, pkg.StatusIncomplete, "", pkg.CreateUserFacingError("", nil), "", "", nil)
		if updateErr != nil {
			errs = multierror.Append(errs, werrors.WrapErr(updateErr, pkg.ErrUpdating))
			continue
		}

		c.lc.Debugf("Processed Job with input file %s", job.FullInputFileLocation())
	}

	return errs
}

// The retry function is a wrapper for the RetryOnStartup call that is utilized by the retry endpoint
func (c *Controller) retry(writer http.ResponseWriter, request *http.Request) {
	err := c.initJobMap()
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, err, http.StatusInternalServerError)
		return
	}

	err = c.refreshJobMap()
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, err, http.StatusInternalServerError)
		return
	}

	err = c.RetryOnStartup()
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, err, http.StatusInternalServerError)
		return
	}
	writer.WriteHeader(http.StatusOK)
	c.lc.Debug("Retry endpoint successfully called")
}

// HandleNewJob will add messages to the message bus
// for the receiver on the OEM side to then pull.
// New jobs to the file sender are saved to local app memory in a jobMap where the keys are job ids and the values
// are the corresponding job objects.
func (c *Controller) HandleNewJob(writer http.ResponseWriter, request *http.Request) {
	job, httpStatus, err := helpers.UnmarshalJob(request)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, err, httpStatus)
		return
	}

	// set file sender gateway as owner
	// init output file fields
	for fileId, file := range job.PipelineDetails.OutputFiles {
		if file.Status == "" {
			job.UpdateOutputFile(fileId, "", "", "", "", "", nil, pkg.FileStatusIncomplete, "")
			file.ErrorDetails = pkg.CreateUserFacingError("", nil)
		}
	}
	_, updateErr := helpers.UpdateJobFields(c.jobRepoClient, job.Id, pkg.OwnerFileSenderGateway, pkg.StatusIncomplete, "", nil, "", "", job.PipelineDetails.OutputFiles)
	if updateErr != nil {
		helpers.HandleErrorMessage(c.lc, writer,
			fmt.Errorf("%s for job %s: %s", pkg.ErrUpdating, job.FullInputFileLocation(), err.Error()),
			http.StatusInternalServerError)
		return
	}

	c.jobMap[job.Id] = job
	// ack back to task-launcher
	writer.WriteHeader(http.StatusOK)
	// publish job object to message bus
	ctx := c.service.BuildContext(uuid.NewString(), common.ContentTypeJSON)
	err = c.publishJob(job, ctx)
	if err != nil {
		c.lc.Info("%s for job: %s: %s", pkg.ErrPublishing, job.FullInputFileLocation(), err.Error())
		jobErr := pkg.CreateUserFacingError(job.Owner, pkg.ErrPublishing)
		_, updateErr = helpers.UpdateJobFields(c.jobRepoClient, job.Id, pkg.OwnerFileSenderGateway, pkg.StatusTransmissionFailed, "", jobErr, "", "", nil)
		if updateErr != nil {
			helpers.HandleErrorMessage(c.lc, writer,
				fmt.Errorf("%s for job %s: %s", pkg.ErrUpdating, job.FullInputFileLocation(), err.Error()),
				http.StatusInternalServerError)
			return
		}
		return
	}

	c.lc.Debugf("Published Job for TaskID %s to receiver", job.PipelineDetails.TaskId)
}

// publishJob publishes a single job to the EdgeX Message Bus
func (c *Controller) publishJob(job types.Job, ctx interfaces.AppFunctionContext) error {
	myEvent := dtos.NewEvent(pkg.OwnerFileSenderGateway, pkg.OwnerFileSenderGateway, pkg.OwnerFileSenderGateway)
	myEvent.AddObjectReading(pkg.ResourceNameJob, job)

	jsonData, err := json.Marshal(myEvent)
	if err != nil {
		return err
	}

	err = c.publisher.Publish(jsonData, ctx)
	if err != nil {
		return err
	}

	return nil
}

// TransmitFile will be how the OEM will pull the specified file to the oem receiver
func (c *Controller) TransmitFile(writer http.ResponseWriter, request *http.Request) {
	jobId, err := helpers.GetByKeyFromRequest(request, pkg.JobIdKey)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, err, http.StatusBadRequest)
		return
	}
	fileIdString, err := helpers.GetByKeyFromRequest(request, pkg.FileIdKey)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, err, http.StatusBadRequest)
		return
	}
	fileId, err := strconv.Atoi(fileIdString)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, err, http.StatusBadRequest)
		return
	}

	// validate job exists
	job, ok := c.jobMap[jobId]
	if !ok {
		helpers.HandleErrorMessage(c.lc, writer, fmt.Errorf("%s for Job %s", pkg.ErrJobInvalid, jobId), http.StatusInternalServerError)
		return
	}

	c.lc.Debugf("Processing output file %d for input file named %s to receiver", fileId, job.FullInputFileLocation())
	// TODO: refactor responses into helper
	fileContents, err := os.ReadFile(filepath.Join(job.PipelineDetails.OutputFiles[fileId].DirName, job.PipelineDetails.OutputFiles[fileId].Name))
	if errors.Is(err, os.ErrNotExist) {
		helpers.HandleErrorMessage(c.lc, writer,
			fmt.Errorf("failed to ReadFile for invalid Job %s where file number does not exist %d: %s", jobId, fileId, err.Error()),
			http.StatusInternalServerError)
		return
	} else if err != nil {
		helpers.HandleErrorMessage(c.lc, writer,
			fmt.Errorf("failed to ReadFile for request for Job %s file number %d: %s", jobId, fileId, err.Error()),
			http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "text/plain")
	_, err = writer.Write(fileContents)
	if err != nil {
		c.lc.Errorf("failed to write http response for output file %d for input file named: %s with error: %s", fileId, job.FullInputFileLocation(), err.Error())
		// TODO: implement retry logic for sending a file or save somewhere that this job file failed to write?
		// Consideration: what delivery guarantee do we care about here? at-least once vs at-most once vs exactly-once?
	}

	c.lc.Debugf("Passed output file %s for input file %s to receiver", job.PipelineDetails.OutputFiles[fileId], job.FullInputFileLocation())
}

// ArchiveFile will take a request for a job ID and archive the files on the gateway to the directory specified in the
// configuration.toml file. It only archives job files that have no present errors.
func (c *Controller) ArchiveFile(writer http.ResponseWriter, request *http.Request) {
	jobId, err := helpers.GetByKeyFromRequest(request, pkg.JobIdKey)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, err, http.StatusBadRequest)
		return
	}

	job, ok := c.jobMap[jobId]
	if !ok {
		helpers.HandleErrorMessage(c.lc, writer, fmt.Errorf("%s for JobId %s:", pkg.ErrJobInvalid, jobId), http.StatusInternalServerError)
		return
	}

	// archive input file
	timestamp := time.Now().UTC().UnixNano()
	inputModifier := fmt.Sprintf("_archive_%s_%d_input.", job.Id, timestamp)
	inputArchiveName := filepath.Join(c.archiveFolder, strings.Replace(job.InputFile.Name, ".", inputModifier, 1))
	err = os.Rename(filepath.Join(job.InputFile.DirName, job.InputFile.Name), inputArchiveName)
	if err != nil {
		helpers.HandleErrorMessage(c.lc, writer, werrors.WrapMsgf(err, "failed to archive input file for JobId %s", jobId), http.StatusInternalServerError)
		// if err when trying to archive input file, then update job repo
		jobErr := pkg.CreateUserFacingError(pkg.OwnerFileSenderGateway, pkg.ErrFileArchiving)
		// update to owner none as we don't want to reprocess files with issues
		_, updateErr := helpers.UpdateJobFields(c.jobRepoClient, job.Id, pkg.OwnerNone, pkg.StatusFileError, "", jobErr, "", "", nil)
		if updateErr != nil {
			helpers.HandleErrorMessage(c.lc, writer,
				fmt.Errorf("%s for job %s: %s", pkg.ErrUpdating, job.FullInputFileLocation(), err.Error()),
				http.StatusInternalServerError)
			return
		}
		return
	}

	// validate output files before proceeding with archival process
	err = job.ValidateFiles()
	if err != nil {
		jobErr := pkg.CreateUserFacingError(pkg.OwnerFileSenderGateway, pkg.ErrFileInvalid)
		_, updateErr := helpers.UpdateJobFields(c.jobRepoClient, job.Id, pkg.OwnerNone, pkg.StatusFileError, "", jobErr, inputArchiveName, "", job.PipelineDetails.OutputFiles)
		if updateErr != nil {
			helpers.HandleErrorMessage(c.lc, writer,
				fmt.Errorf("%s for job %s: %s", pkg.ErrUpdating, job.FullInputFileLocation(), err.Error()),
				http.StatusInternalServerError)
			return
		}
		c.lc.Warnf("files %s not found for job id %s: %v but still continuing to archive valid job output files", job.PipelineDetails.OutputFiles, job.Id, err)
	}

	// archive output files
	for fileId, _ := range job.PipelineDetails.OutputFiles {
		// if an output file already has an errored state, then do not archive that output files
		if job.PipelineDetails.OutputFiles[fileId].Status == pkg.FileStatusWriteFailed || job.PipelineDetails.OutputFiles[fileId].Status == pkg.FileStatusTransmissionFailed || job.PipelineDetails.OutputFiles[fileId].Status == pkg.FileStatusInvalid {
			continue // don't archive files that weren't written
		}

		// move the output files to the archive
		outputModifier := fmt.Sprintf("_archive_%s_%d_output.", job.Id, timestamp)
		outputArchiveName := filepath.Join(c.archiveFolder, strings.Replace(job.PipelineDetails.OutputFiles[fileId].Name, ".", outputModifier, 1))
		err = os.Rename(filepath.Join(job.PipelineDetails.OutputFiles[fileId].DirName, job.PipelineDetails.OutputFiles[fileId].Name), outputArchiveName)
		if err != nil {
			jobErr := pkg.CreateUserFacingError(pkg.OwnerFileSenderGateway, pkg.ErrFileArchiving)
			// update output file and job fields if error upon trying to archive output file
			// update file owner for job
			job.UpdateOutputFile(fileId, "", "", "", "", "", pkg.ErrFileArchiving, pkg.FileStatusArchiveFailed, pkg.OwnerFileSenderGateway)
			c.lc.Debugf("failure archiving file named %s for job %s", job.PipelineDetails.OutputFiles[fileId].Name, jobId)
			_, updateErr := helpers.UpdateJobFields(c.jobRepoClient, job.Id, pkg.OwnerNone, pkg.StatusFileError, "", jobErr, inputArchiveName, "", job.PipelineDetails.OutputFiles)
			if updateErr != nil {
				helpers.HandleErrorMessage(c.lc, writer,
					fmt.Errorf("%s for job %s: %s", pkg.ErrUpdating, job.FullInputFileLocation(), err.Error()),
					http.StatusInternalServerError)
				return
			}
			continue
		} else {
			// if this point is reached for an output file, then it had no error and can be marked complete
			job.UpdateOutputFile(fileId, "", "", "", outputArchiveName, outputArchiveName, err, pkg.FileStatusComplete, pkg.OwnerNone)
			c.lc.Debugf("for JobId %s archived output file %d", jobId, fileId)
		}
	}

	if job.Status != pkg.StatusFileError {
		job.Status = pkg.StatusComplete
	}
	updatedJob, updateErr := helpers.UpdateJobFields(c.jobRepoClient, job.Id, pkg.OwnerNone, job.Status, "", job.ErrorDetails, inputArchiveName, inputArchiveName, job.PipelineDetails.OutputFiles)
	if updateErr != nil {
		helpers.HandleErrorMessage(c.lc, writer,
			fmt.Errorf("%s for job %s: %s", pkg.ErrUpdating, job.FullInputFileLocation(), err.Error()),
			http.StatusInternalServerError)
		return
	}
	c.lc.Debugf("updated job %v after job update after archiving files in sender gw", updatedJob)

	writer.WriteHeader(http.StatusOK)

	if updatedJob.InputFile.Extension != ".png" && updatedJob.InputFile.Extension != ".jpg" && updatedJob.InputFile.Extension != ".jpeg" && updatedJob.InputFile.Extension != ".gif" {
		c.Convert(writer, updatedJob, inputArchiveName)
	}

	// TODO(@Sam): Should we delete jobs if all output files are not archived?
	// TODO(@Sam): Retry logic for archiving output files if failure?
	// delete sent job
	delete(c.jobMap, jobId)
}

func (c *Controller) Convert(writer http.ResponseWriter, job types.Job, inputArchiveDir string) {

	jobId := job.Id
	inputViewableDir := inputArchiveDir
	//open input file in order to convert and save
	filebytes, err := os.ReadFile(inputArchiveDir)

	if len(filebytes) != 0 {
		if err != nil {
			helpers.HandleErrorMessage(c.lc, writer, werrors.WrapMsgf(err, "failed to open non-zero byte file for conversion for JobId %s", jobId), http.StatusInternalServerError)
			return
		}
		time.Sleep(1 * time.Second)
		srcImg, err := imgconv.Open(inputArchiveDir)
		if err != nil {
			helpers.HandleErrorMessage(c.lc, writer, werrors.WrapMsgf(err, "failed to open archival file for conversion for JobId %s", jobId), http.StatusInternalServerError)
			return
		}
		// update the archive file path
		inputViewableDir = strings.Replace(inputArchiveDir, job.InputFile.Extension, ".jpeg", 1)

		// save resulting file as jpeg
		err = imgconv.Save(inputViewableDir, srcImg, &imgconv.FormatOption{Format: imgconv.JPEG})
		if err != nil {
			jobErr := pkg.CreateUserFacingError(pkg.OwnerFileSenderGateway, pkg.ErrFileInvalid)
			_, updateErr := helpers.UpdateJobFields(c.jobRepoClient, job.Id, pkg.OwnerNone, pkg.StatusFileError, "", jobErr, inputArchiveDir, "", job.PipelineDetails.OutputFiles)
			if updateErr != nil {
				helpers.HandleErrorMessage(c.lc, writer, werrors.WrapMsgf(err, "failed to write archival jpeg for input file for JobId %s", jobId), http.StatusInternalServerError)
				return
			}
		}
		c.lc.Debugf("visualization input file created for job %s", jobId)
	}

	for fileId, _ := range job.PipelineDetails.OutputFiles {
		outputArchiveName := job.PipelineDetails.OutputFiles[fileId].ArchiveName
		filebytes, err := os.ReadFile(outputArchiveName)
		if err != nil {
			helpers.HandleErrorMessage(c.lc, writer, werrors.WrapMsgf(err, "failed to ReadFile"), http.StatusInternalServerError)
			return
		}
		if len(filebytes) != 0 {
			if err != nil {
				helpers.HandleErrorMessage(c.lc, writer, werrors.WrapMsgf(err, "failed to open non-zero byte file for output file %d", fileId), http.StatusInternalServerError)
				return
			}

			//open output file for conversion
			srcImg, err := imgconv.Open(outputArchiveName)
			if err != nil {
				helpers.HandleErrorMessage(c.lc, writer, werrors.WrapMsgf(err, "failed to open archival output file for conversion for JobId %s", jobId), http.StatusInternalServerError)
				return
			}
			// update the archive file path
			outputViewableName := strings.Replace(outputArchiveName, job.PipelineDetails.OutputFiles[fileId].Extension, ".jpeg", 1)

			// save resulting file as jpeg
			err = imgconv.Save(outputViewableName, srcImg, &imgconv.FormatOption{Format: imgconv.JPEG})
			if err != nil {
				jobErr := pkg.CreateUserFacingError(pkg.OwnerFileSenderGateway, pkg.ErrFileArchiving)
				// update output file and job fields if error upon trying to create visual output file
				// update file owner for job
				job.UpdateOutputFile(fileId, "", "", "", "", "", pkg.ErrFileArchiving, pkg.FileStatusArchiveFailed, pkg.OwnerFileSenderGateway)
				c.lc.Errorf("failed to create visualized output file named %s for job %s", job.PipelineDetails.OutputFiles[fileId].Name, jobId)
				_, updateErr := helpers.UpdateJobFields(c.jobRepoClient, job.Id, pkg.OwnerNone, pkg.StatusFileError, "", jobErr, inputArchiveDir, "", job.PipelineDetails.OutputFiles)
				if updateErr != nil {
					helpers.HandleErrorMessage(c.lc, writer,
						fmt.Errorf("%s for job %s: %s", pkg.ErrUpdating, job.FullInputFileLocation(), err.Error()),
						http.StatusInternalServerError)
					return
				}
				continue
			}
			c.lc.Debugf("visualization file created for output file %d on job %s", fileId, jobId)
			job.UpdateOutputFile(fileId, "", "", "", outputArchiveName, outputViewableName, err, pkg.FileStatusComplete, pkg.OwnerNone)
		}
	}
	updatedJob, updateErr := helpers.UpdateJobFields(c.jobRepoClient, jobId, pkg.OwnerNone, job.Status, "", job.ErrorDetails, inputArchiveDir, inputViewableDir, job.PipelineDetails.OutputFiles)
	if updateErr != nil {
		helpers.HandleErrorMessage(c.lc, writer,
			fmt.Errorf("%s for job %s: %s", pkg.ErrUpdating, job.FullInputFileLocation(), err.Error()),
			http.StatusInternalServerError)
		return
	}
	c.lc.Debugf("updated job %v after job update after visualization files were created and set", updatedJob)

}

// initJobMap is a helper function to initialize the job map with the jobs it will need to send to the File Receiver OEM
func (c *Controller) initJobMap() error {
	jobs, err := c.jobRepoClient.RetrieveAllByOwner(pkg.OwnerFileRecvOem)
	if err != nil {
		return fmt.Errorf("%s %s: %s", pkg.OwnerFileSenderGateway, pkg.ErrRetrieving, err.Error())
	}
	for _, job := range jobs {
		c.jobMap[job.Id] = job
		for fileId, file := range job.PipelineDetails.OutputFiles {
			if file.Status == "" {
				job.UpdateOutputFile(fileId, "", "", "", "", "", nil, pkg.FileStatusIncomplete, pkg.OwnerFileSenderGateway)
			}
		}
	}
	c.lc.Debugf("added %d jobs to the jobMap", len(c.jobMap))
	return nil
}

// refreshJobMap is a helper function to refresh the job map with the jobs it will need to send to the File Receiver OEM
func (c *Controller) refreshJobMap() error {
	jobs, err := c.jobRepoClient.RetrieveAllByOwner(pkg.OwnerFileSenderGateway)
	if err != nil {
		return fmt.Errorf("%s %s: %s", pkg.OwnerFileSenderGateway, pkg.ErrRetrieving, err.Error())
	}
	for _, job := range jobs {
		c.jobMap[job.Id] = job
		for fileId, file := range job.PipelineDetails.OutputFiles {
			if file.Status == "" {
				job.UpdateOutputFile(fileId, "", "", "", "", "", nil, pkg.FileStatusIncomplete, pkg.OwnerFileSenderGateway)
			}
		}
	}
	c.lc.Debugf("refreshed jobs in the jobMap with quantity %d", len(c.jobMap))
	return nil
}
