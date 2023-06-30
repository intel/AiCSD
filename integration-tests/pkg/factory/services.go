/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package factory

import "aicsd/integration-tests/pkg"

type TestService struct {
	Name           string
	Port           int
	StartupMessage string
}

// edgeXServices returns the TestService services for the EdgeX services using for integration tests.
func edgeXServices() []TestService {
	return []TestService{
		{pkg.ServiceConsul, pkg.PortConsul, pkg.StartupMsgConsul},
		{pkg.ServiceRedis, pkg.PortRedis, pkg.StartupMsgRedis},
	}
}

// PipelineSimTestServices returns the TestService services for the Pipeline Simulator integration tests.
func PipelineSimTestServices() []TestService {
	services := []TestService{
		{pkg.ServiceFileWatcher, pkg.PortFileWatcher, pkg.StartupMsgFileWatcher},
		{pkg.ServiceDataOrg, pkg.PortDataOrg, pkg.StartupMsgDataOrg},
		{pkg.ServiceSenderOem, pkg.PortSenderOem, pkg.StartupMsgSenderOem},
		{pkg.ServiceReceiverGW, pkg.PortReceiverGW, pkg.StartupMsgReceiverGW},
		{pkg.ServiceJobRepo, pkg.PortJobRepo, pkg.StartupMsgJobRepo},
		{pkg.ServiceTaskLauncher, pkg.PortTaskLauncher, pkg.StartupMsgTaskLauncher},
		{pkg.ServiceSenderGW, pkg.PortSenderGW, pkg.StartupMsgSenderGW},
		{pkg.ServiceReceiverOem, pkg.PortReceiverOem, pkg.StartupMsgReceiverOem},
		{pkg.ServicePipelineSim, pkg.PortPipelineSim, pkg.StartupMsgPipelineSim},
		{pkg.ServiceAppMQTTExport, pkg.PortAppMQTTExport, ""},
	}
	return append(services, edgeXServices()...)
}

// PipelineSimTestServices returns the TestService services for the Pipeline Validator integration tests.
func PipelineValTestServices() []TestService {
	services := []TestService{
		{pkg.ServicePipelineVal, pkg.PortPipelineVal, pkg.StartupMsgPipelineVal},
		{pkg.ServicePipelineSim, pkg.PortPipelineSim, pkg.StartupMsgPipelineSim},
	}
	return append(services, edgeXServices()...)
}
