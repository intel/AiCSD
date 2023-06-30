/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package pkg

import "fmt"

// Common errors
var (
	// main level errors
	ErrLoadingConfig  = fmt.Errorf("failed to load configuration")
	ErrRunningService = fmt.Errorf("failed to run the service")

	// controller level errors
	ErrFmtRegisterRoutes  = "could not create route for %s"
	ErrFmtInvalidInput    = "invalid input got %s, expected %s"
	ErrInvalidInput       = fmt.Errorf("invalid input")
	ErrEmptyInput         = fmt.Errorf("input required, but none was received")
	ErrCallingGet         = fmt.Errorf("failed calling get")
	ErrWritingHttpResp    = fmt.Errorf("failed to write http response")
	ErrFmtWritingHttpResp = "failed to write http response for %s"
	ErrFmtProcessingReq   = "failed to process request: %s"
	ErrFmtTcpLookup       = "dial tcp: lookup %s"

	// job validation errors
	ErrJobInvalid       = fmt.Errorf("failed to validate job")
	ErrJobIdEmpty       = fmt.Errorf("no job id specified")
	ErrJobIdNotFound    = "job id not found for %s"
	ErrMarshallingJob   = fmt.Errorf("failed to marshal job(s)")
	ErrUnmarshallingJob = fmt.Errorf("failed to unmarshal job")

	// job handling errors
	ErrPublishing          = fmt.Errorf("failed to publish message to message bus")
	ErrRetrieving          = fmt.Errorf("failed to retrieve job(s) from job repo")
	ErrJobsEmpty           = fmt.Errorf("job repo is empty")
	ErrUpdating            = fmt.Errorf("failed to update job from job repo")
	ErrHandleJob           = fmt.Errorf("failed to handle job")
	ErrJobCreation         = fmt.Errorf("failed to create job")
	ErrTransmitJob         = fmt.Errorf("failed to transmit job")
	ErrFmtRetrieving       = "failed to retrieve job(s) %s from job repo"
	ErrFmtJobDelete        = "failed to delete job for id %s"
	ErrFmtRedisWatchFailed = "failed to watch redis for job id %s"

	// file related errors
	ErrFileTransmitting      = fmt.Errorf("failed to transmit file")
	ErrFmtFileTransmitting   = "failed to transmit file %s"
	ErrFileWrite             = fmt.Errorf("failed to write file")
	ErrFmtFileWrite          = "failed to write file %s"
	ErrFileArchiving         = fmt.Errorf("failed to archive file")
	ErrFmtFileArchiving      = "failed to archive file named %s"
	ErrFileInvalid           = fmt.Errorf("failed to validate file")
	ErrFmtFileInvalid        = "failed to validate file named %s"
	ErrFileRejecting         = fmt.Errorf("failed to reject file")
	ErrFmtFileRejecting      = "failed to reject file named %s"
	ErrFileDeletingReject    = fmt.Errorf("failed to delete reject file")
	ErrFmtFileDeletingReject = "failed to delete reject file named %s"

	// generic errors for testing
	ErrJSONMarshalErr = fmt.Errorf("unexpected end of JSON input")

	// job error details
	ErrFmtJobDetails     = "(%s): %s"
	ErrJobNoMatchingTask = fmt.Errorf("no tasks could be matched to the input file name")
	ErrPipelineFailed    = fmt.Errorf("an error occurred in the processing pipeline")

	// miscellaneous errors
	ErrTranslating = fmt.Errorf("error translating field")
	ErrAuthHeader  = fmt.Errorf("could not add authentication header")
)

// UserFacingError is meant to help define the line of errors an end user may see on the UI
// as opposed to errors meant to help with debugging or provide more underlying information.
type UserFacingError struct {
	Owner string
	// Error is of string type.
	// Albeit slightly misleading at first glance,
	// but string type is necessary for translations to other languages...
	Error string
}

// CreateUserFacingError is a constructor for any *UserFacingError.
// This is meant to help create a defined error response for the UI,
// keeping in mind that internationalization of fields is a necessity moving forward.
// That is part of the reason the UserFacingError.Error is of type string.
func CreateUserFacingError(owner string, err error) *UserFacingError {
	if err != nil {
		return &UserFacingError{
			owner, err.Error(),
		}
	} else {
		return &UserFacingError{
			owner, "",
		}
	}
}

// translationErrorMap is a map to allow for easier look up on user facing error types to then translate
// We need a map with <key>:<value> of <internal err name to project>:<err value string> to properly translate,
// so that is why this translationErrorMap was created to translate job and file user facing errors as needed.
var translationErrorMap = map[string]string{
	// job validation errors
	"ErrJobInvalid": ErrJobInvalid.Error(),
	"ErrJobIdEmpty": ErrJobIdEmpty.Error(),

	// job handling errors
	"ErrPublishing":  ErrPublishing.Error(),
	"ErrRetrieving":  ErrRetrieving.Error(),
	"ErrJobsEmpty":   ErrJobsEmpty.Error(),
	"ErrUpdating":    ErrUpdating.Error(),
	"ErrHandleJob":   ErrHandleJob.Error(),
	"ErrJobCreation": ErrJobCreation.Error(),
	"ErrTransmitJob": ErrTransmitJob.Error(),
	//ErrFmtRetrieving       = "failed to retrieve job(s) %s from job repo" // TODO?
	//ErrFmtJobDelete        = "failed to delete job for id %s" // TODO?

	// file related errs
	"ErrFileTransmitting": ErrFileTransmitting.Error(),
	"ErrFileArchiving":    ErrFileArchiving.Error(),
	"ErrFileWrite":        ErrFileWrite.Error(),
	"ErrFileInvalid":      ErrFileInvalid.Error(),

	"ErrJobNoMatchingTask": ErrJobNoMatchingTask.Error(),
	"ErrPipelineFailed":    ErrPipelineFailed.Error(),

	// translation errors
	"ErrTranslating": ErrTranslating.Error(),
}

// GetErrorType is a lookup helper function to take in an err message string,
// and respond back with the error key name so that err string can then be translated to the UI.
func GetErrorType(errMsg string) string {
	for errType, errContents := range translationErrorMap {
		if errContents == errMsg {
			return errType
		}
	}
	// If there is an issue finding the correct err key, then return ErrTranslating
	return "ErrTranslating"
}
