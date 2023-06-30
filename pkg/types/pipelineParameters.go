/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package types

// PipelineParams Object Attributes
type PipelineParameters struct {
	// InputFileLocation is the path to the unprocessed file
	InputFileLocation string
	// OutputFileFolder is the path to inference result file (CSV or Image files)
	OutputFileFolder string
	// ModelParams are parameters specific to a pipeline and applied as data is launched in pipeline execution
	ModelParams map[string]string
	// JobUpdateUrl is Url to update the associated job
	JobUpdateUrl string
	// PipelineStatusUrl is Url to update the status of pipeline execution
	PipelineStatusUrl string
}
