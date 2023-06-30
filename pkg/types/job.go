/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package types

import (
	"aicsd/pkg"
	"aicsd/pkg/translation"
	"fmt"
	"time"

	"github.com/nicksnyder/go-i18n/v2/i18n"
)

const (
	JobOwner                = "Owner"
	JobStatus               = "Status"
	JobInputFileHost        = "InputFile.Hostname"
	JobInputFileDir         = "InputFile.DirName"
	JobInputArchiveName     = "InputFile.ArchiveName"
	JobInputViewableName    = "InputFile.Viewable"
	JobPipelineTaskId       = "PipelineDetails.TaskId"
	JobPipelineStatus       = "PipelineDetails.Status"
	JobPipelineQCFlags      = "PipelineDetails.QCFlags"
	JobPipelineOutputHost   = "PipelineDetails.OutputFileHost"
	JobPipelineOutputFiles  = "PipelineDetails.OutputFiles"
	JobPipelineResults      = "PipelineDetails.Results"
	JobErrorDetailsOwner    = "ErrorDetails.Owner"
	JobErrorDetailsErrorMsg = "ErrorDetails.Error"
)

// TODO: add json marshalling attributes
type Job struct {
	// Id is the unique identifier
	Id string
	// Owner is the component that owns object
	Owner string
	// InputFile contains information on the unprocessed file
	InputFile FileInfo
	// PipelineDetails contains the information pertaining to the task run for this job
	PipelineDetails PipelineInfo
	// LastUpdated is the update time in ns from UTC
	LastUpdated int64
	// Status is the current status of job
	Status string
	// ErrorDetails is the piece containing user facing error information
	ErrorDetails *pkg.UserFacingError
	// Verification contains the state of manual review in the form of an enum
	// 0 = Pending; 1 = Accepted; 2 = Rejected
	Verification int
}

type PipelineInfo struct {
	// TaskId is the unique identifier for the task that matched resulting in the pipeline being launched
	TaskId string
	// Status is the status of the pipeline running the task
	Status string
	// QCFlags are the flags set by the pipeline
	QCFlags string
	// OutputFileHost is the hostname associated with the output of the task
	OutputFileHost string
	// OutputFiles are the output files for a given task
	OutputFiles []OutputFile
	// Results is the string of the output from the model with or without an image file
	Results string
}

func (j *Job) SetLastUpdated() {
	j.LastUpdated = time.Now().UTC().UnixNano()
}

// ValidateHost checks that the job hostname is valid,
// and updates ErrorDetails if otherwise.
func (j *Job) ValidateHost(thisHostname string) error {
	if j.PipelineDetails.OutputFileHost != thisHostname {
		j.ErrorDetails = pkg.CreateUserFacingError(j.Owner, pkg.ErrJobInvalid)
		return fmt.Errorf("hostname does not match: got %s, expected %s", j.PipelineDetails.OutputFileHost, thisHostname)
	}
	return nil
}

// Translate translates all job fields that are need to be translated according
// to the accept language from the HTTP header Accept-Language.
func (j *Job) Translate(bundle *i18n.Bundle, accept string) error {
	loc := i18n.NewLocalizer(bundle, accept)

	if j.Owner != "" {
		jobOwner, err := translation.TranslateField(loc, j.Owner)
		if err != nil {
			return fmt.Errorf("error translating job.Owner field with value: %s", j.Owner)
		}
		j.Owner = jobOwner
	}

	if j.Status != "" {
		jobStatus, err := translation.TranslateField(loc, j.Status)
		if err != nil {
			return fmt.Errorf("error translating job.Status field with value: %s", j.Status)
		}
		j.Status = jobStatus
	}

	if j.ErrorDetails != nil {
		errDetails, err := translation.TranslateErrorDetails(loc, j.ErrorDetails)
		if err != nil {
			return fmt.Errorf("error translating job.ErrorDetails field with value: %s", j.ErrorDetails)
		}
		j.ErrorDetails = errDetails
	}

	if j.PipelineDetails.Status != "" {
		pipelineStatus, err := translation.TranslateField(loc, j.PipelineDetails.Status)
		if err != nil {
			return fmt.Errorf("error translating job.PipelineDetails.Status field with value: %s", j.PipelineDetails.Status)
		}
		j.PipelineDetails.Status = pipelineStatus
	}

	for k := range j.PipelineDetails.OutputFiles {
		if j.PipelineDetails.OutputFiles[k].Status != "" {
			outputFileStatus, err := translation.TranslateField(loc, j.PipelineDetails.OutputFiles[k].Status)
			if err != nil {
				return fmt.Errorf("error translating job.PipelineDetails.OutputFiles[%d].Status field with value: %s", k, j.PipelineDetails.OutputFiles[k].Status)
			}
			j.PipelineDetails.OutputFiles[k].Status = outputFileStatus
		}

		if j.PipelineDetails.OutputFiles[k].Owner != "" {
			outputFileOwner, err := translation.TranslateField(loc, j.PipelineDetails.OutputFiles[k].Owner)
			if err != nil {
				return fmt.Errorf("error translating job.PipelineDetails.OutputFiles[%d].Status field with value: %s", k, j.PipelineDetails.OutputFiles[k].Status)
			}
			j.PipelineDetails.OutputFiles[k].Owner = outputFileOwner
		}

		if j.PipelineDetails.OutputFiles[k].ErrorDetails != nil {
			errDetails, err := translation.TranslateErrorDetails(loc, j.PipelineDetails.OutputFiles[k].ErrorDetails)
			if err != nil {
				return fmt.Errorf("error translating job.PipelineDetails.OutputFiles[%d].ErrorDetails field with value: %s", k, j.ErrorDetails)
			}
			j.PipelineDetails.OutputFiles[k].ErrorDetails = errDetails
		}
	}

	return nil
}
