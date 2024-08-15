/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package pkg

import (
	"aicsd/pkg/types"
	"path/filepath"
	"time"
)

// time constants
const (
	HttpTimeout   = 15 * time.Second
	MorePauseTime = 20 * time.Second
	PauseTime     = 10 * time.Second
	LessPauseTime = 5 * time.Second
	ContainerWait = 30 * time.Second
)

// file names
const (
	File1        = "test-image.tiff"
	File2        = "test-image-2.tiff"
	File3        = "test-image-3.tiff"
	File4        = "test-image-4.tiff"
	File5        = "test-image-5.tiff"
	FileHiDef    = "test-image-4k.tiff"
	FileParsable = "t1.tiff"

	File1out        = "test-image-sim.tiff"
	File2out        = "test-image-2-sim.tiff"
	File3out        = "test-image-3-sim.tiff"
	File4out        = "test-image-4-sim.tiff"
	File5out        = "test-image-5-sim.tiff"
	FileHiDefout    = "test-image-4k-sim.tiff"
	FileParsableOut = "t1-sim.tiff"

	File1archive    = "test-image_archive_"
	File1outArchive = "test-image-sim_archive_"
)

// Urls
const (
	ConsulHeaderKey            = "Authorization"
	ConsulTokenFmt             = "Bearer %s"
	ConsulGetServicesUrl       = "http://127.0.0.1:8500/v1/agent/services"
	ConsulChangeLogLevelUrl    = "http://127.0.0.1:8500/v1/kv/edgex/appservices/2.0/%s/Writable/LogLevel?dc=dc1&flags=0"
	ConsulChangeFWConfigVarUrl = "http://127.0.0.1:8500/v1/kv/edgex/appservices/2.0/app-file-watcher/UpdatableSettings/%s"
	ConsulGetLogLevelUrl       = "http://127.0.0.1:8500/v1/kv/edgex/appservices/2.0/%s/Writable/LogLevel?dc=dc1"
	ConsulTaskLauncherRetryUrl = "http://127.0.0.1:8500/v1/kv/edgex/appservices/2.0/app-task-launcher/ApplicationSettings/RetryWindow?dc=dc1&flags=0"
	ConsulAttributeParserUrl   = "http://127.0.0.1:8500/v1/kv/edgex/appservices/2.0/app-data-organizer/AttributeParser/%s?dc=dc1"

	ConsulUrl              = "http://127.0.0.1:8500"
	RedisUrl               = "http://127.0.0.1:6379"
	FileWatcherUrl         = "http://127.0.0.1:59780"
	DataOraganizerUrl      = "http://127.0.0.1:59781"
	FileSenderOEMUrl       = "http://127.0.0.1:59782"
	FileReceiverGatewayUrl = "http://127.0.0.1:59783"
	JobRepositoryUrl       = "http://127.0.0.1:59784"
	TaskLauncherUrl        = "http://127.0.0.1:59785"
	FileSenderGatewayUrl   = "http://127.0.0.1:59786"
	FileReceiverOEMUrl     = "http://127.0.0.1:59787"
	WebUIUrl               = "http://127.0.0.1:4200"
	PipelineSimUrl         = "http://127.0.0.1:59789"
	PipelineValUrl         = "http://127.0.0.1:59788"
	PingEndpoint           = "/api/v2/ping"
	GetJobUrl              = "http://127.0.0.1:59784/api/v1/job"

	MaxRetries = 3
)

const (
	// Service hostnames used for integration tests
	ServiceFileWatcher          = "file-watcher-1"
	ServiceDataOrg              = "data-organizer-1"
	ServiceSenderOem            = "file-sender-oem-1"
	ServiceReceiverGW           = "file-receiver-gateway-1"
	ServiceJobRepo              = "job-repository-1"
	ServiceTaskLauncher         = "task-launcher-1"
	ServiceSenderGW             = "file-sender-gateway-1"
	ServiceReceiverOem          = "file-receiver-oem-1"
	ServicePipelineSim          = "pipeline-sim-1"
	ServicePipelineVal          = "pipeline-val-1"
	ServiceConsul               = "edgex-core-consul-1"
	ServiceRedis                = "edgex-redis-1"
	ServiceAppMQTTExport        = "app-mqtt-export-1"
	ServiceKong                 = "kong-1"
	ServiceKongDb               = "kong-db-1"
	ServiceSecurityBootstrapper = "security-bootstrapper-1"
	ServiceProxySetup           = "security-proxy-setup-1"
	ServiceSecretStoreSetup     = "seturity-secretstore-setup-1"
	ServiceVault                = "vault-1"

	// Startup messages for services
	StartupMsgFileWatcher  = "Started the file watcher microservice"
	StartupMsgDataOrg      = "Starting HTTP Web Server on address data-organizer:59781"
	StartupMsgSenderOem    = "Starting HTTP Web Server on address file-sender-oem:59782"
	StartupMsgReceiverGW   = "Starting HTTP Web Server on address file-receiver-gateway:59783"
	StartupMsgJobRepo      = "Starting HTTP Web Server on address job-repository:59784"
	StartupMsgTaskLauncher = "Starting HTTP Web Server on address task-launcher:59785"
	StartupMsgSenderGW     = "Starting HTTP Web Server on address file-sender-gateway:59786"
	StartupMsgReceiverOem  = "Starting HTTP Web Server on address file-receiver-oem:59787"
	StartupMsgPipelineSim  = "Starting HTTP Web Server on address pipeline-sim:59789"
	StartupMsgPipelineVal  = "Starting HTTP Web Server on address pipeline-val:59788"
	StartupMsgConsul       = "Consul agent running!"
	StartupMsgRedis        = "Ready to accept connections"

	// Ports for services using iota to start with 59780 and + 1 for subsequent services
	PortFileWatcher = iota + 59780
	PortDataOrg
	PortSenderOem
	PortReceiverGW
	PortJobRepo
	PortTaskLauncher
	PortSenderGW
	PortReceiverOem
	PortPipelineSim   = 10107
	PortPipelineVal   = 59788
	PortConsul        = 8500
	PortAppMQTTExport = 59703
	PortRedis         = 6379
)

// log message for file-watcher
var FmtFileWatcherLog = "Sent new file notification for file %s"

// file-watcher config parameters
const (
	WatchSubfolders   = "WatchSubfolders"
	FileExclusionList = "FileExclusionList"
)

var EdgeXServices = []string{"edgex-core-consul", "edgex-redis", "edgex-mqtt-broker", "edgex-app-mqtt-export"}

var TaskOnlyResults = types.Task{
	Description:      "Generate Results",
	JobSelector:      "{ \"in\" : [ \"test-image\", {\"var\" : \"InputFile.Name\" } ] }",
	PipelineId:       "only-results",
	ResultFileFolder: filepath.Join("/tmp", "files", "output"),
	ModelParameters:  map[string]string{"Brightness": "0"},
}
var TaskOnlyFile = types.Task{
	Description:      "Generate File",
	JobSelector:      "{ \"in\" : [ \"test-image\", {\"var\" : \"InputFile.Name\" } ] }",
	PipelineId:       "only-file",
	ResultFileFolder: filepath.Join("/tmp", "files", "output"),
	ModelParameters:  map[string]string{"Brightness": "0"},
}
var TaskMultiFile = types.Task{
	Description:      "Generate Files",
	JobSelector:      "{ \"in\" : [ \"test-image\", {\"var\" : \"InputFile.Name\" } ] }",
	PipelineId:       "multi-file",
	ResultFileFolder: filepath.Join("/tmp", "files", "output"),
	ModelParameters:  map[string]string{"Brightness": "0"},
}

var LocalDir = filepath.Join("../", "sample-files")
var GatewayInputDir = filepath.Join("data", "gateway-files", "input")
var GatewayOutputDir = filepath.Join("data", "gateway-files", "output")
var GatewayArchiveDir = filepath.Join("data", "gateway-files", "archive")
var GatewayRejectDir = filepath.Join("data", "gateway-files", "reject")
var OemInputDir = filepath.Join("data", "oem-files", "input")
var OemOutputDir = filepath.Join("data", "oem-files", "output")
