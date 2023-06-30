/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package types

const OwnerPipelineValidator = "pipeline-val"

// PipelineParams are the parameters that should get returned by the pipeline service.
type PipelineParams struct {
	Id                string `json:"id"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	SubscriptionTopic string `json:"subscriptionTopic"`
	Status            string `json:"status"`
}

// LaunchInfo is the structure that defines the minimal information that must be passed to create a job and send it to the pipeline.
type LaunchInfo struct {
	InputFileLocation string
	PipelineTopic     string
	OutputFileFolder  string
	ModelParams       map[string]string
}
