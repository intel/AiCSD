/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package pkg

// Keys for endpoints
const (
	// REST keys
	JobIdKey    = "jobid"
	TaskIdKey   = "taskid"
	FileIdKey   = "fileid"
	FilenameKey = "filename"
	OwnerKey    = "owner"
	// MQTT key
	PublishTopicKey = "publish-topic"
	CustomTopicKey  = "custom-topic"
)

// Endpoints defined here
const (
	EndpointDataToHandle = "/api/v1/dataToHandle"
	EndpointJob          = "/api/v1/job"
	EndpointJobId        = "/api/v1/job/{" + JobIdKey + "}"
	EndpointJobOwner     = "/api/v1/job/owner/{" + OwnerKey + "}"
	EndpointJobPipeline  = "/api/v1/job/pipeline/{" + JobIdKey + "}/{" + TaskIdKey + "}"

	// TODO: implement update for job results as PUT
	// EndpointJobResults = "/api/v1/job/results/{" + JobIdKey + "}"
	EndpointMatchTask         = "/api/v1/matchTask"
	EndpointNotifyNewFile     = "/api/v1/notifyNewFile"
	EndpointPipelineStatus    = "/api/v1/pipelineStatus/{" + JobIdKey + "}/{" + TaskIdKey + "}"
	EndpointTask              = "/api/v1/task"
	EndpointTaskId            = "/api/v1/task/{" + TaskIdKey + "}"
	EndpointTransmitJob       = "/api/v1/transmitJob"
	EndpointTransmitFile      = "/api/v1/transmitFile"
	EndpointTransmitFileJobId = "/api/v1/transmitFile/{" + JobIdKey + "}/{" + FileIdKey + "}"
	EndpointArchiveFile       = "/api/v1/archiveFile/{" + JobIdKey + "}"
	EndpointRetry             = "/api/v1/retry"
	EndpointRejectFile        = "/api/v1/reject/{" + JobIdKey + "}"

	// Endpoints for pipeline validator
	EndpointLaunchPipeline = "/api/v1/launchPipeline"
	EndpointGetPipelines   = "/api/v1/pipelines"
)

// Job Status
const (
	StatusComplete           = "Complete"
	StatusIncomplete         = "Incomplete" // Job is still processing
	StatusNoPipeline         = "NoPipelineFound"
	StatusPipelineError      = "PipelineError"
	StatusTransmissionFailed = "TransmissionFailed" // This is for job publishing errors, and is unrelated to files
	StatusFileError          = "FileErrored"        // An output file error occurred
)

// File Status
const (
	FileStatusComplete           = "FileComplete"
	FileStatusIncomplete         = "FileIncomplete"
	FileStatusTransmissionFailed = "FileTransmissionFailed"
	FileStatusArchiveFailed      = "FileArchivalFailed"
	FileStatusWriteFailed        = "FileWriteFailed"
	FileStatusInvalid            = "FileInvalid"
)

// Job Owners
const (
	OwnerNone              = "none"
	OwnerFileWatcher       = "file-watcher"
	OwnerDataOrg           = "data-organizer"
	OwnerFileSenderOem     = "file-sender-oem"
	OwnerFileRecvGateway   = "file-receiver-gateway"
	OwnerTaskLauncher      = "task-launcher"
	OwnerFileSenderGateway = "file-sender-gateway"
	OwnerFileRecvOem       = "file-receiver-oem"
	JobRepository          = "job-repository"
)

// Task Status
const (
	TaskStatusComplete     = "PipelineComplete"
	TaskStatusProcessing   = "PipelineProcessing"
	TaskStatusFailed       = "PipelineFailed"
	TaskStatusFileNotFound = "FileNotFound"
)

// File specifications for writing a file
const (
	FilePermissions   = 0666
	FolderPermissions = 0777
)

// EdgeX related
const (
	ResourceNameJob = "job"
	ServiceKeyFmt   = "app-%s"
	DatabasePath    = "redisdb"
)

// Loop up to 5 output files for the pipeline sim and integration tests
const (
	LoopForMultiOutputFiles = 5
)

// Request headers related
const (
	AcceptLanguage  = "Accept-Language"
	LanguageChinese = "zh"
)
