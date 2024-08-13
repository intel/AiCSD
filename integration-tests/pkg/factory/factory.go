/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package factory

import (
	"aicsd/integration-tests/pkg"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/docker/go-connections/nat"
	tc "github.com/testcontainers/testcontainers-go/modules/compose"
	"github.com/testcontainers/testcontainers-go/wait"
)

type IntegrationTestFactory interface {
	GetDockerServiceNames() []string
	GetApplicationServiceNames() []string
	GetEdgeXServices() []string
	Down() error
	StartAllServices() error
	GetServiceUrl(service string) string
	StartServiceWithWait(service string) error
	StartServices(services ...string) error
	StopServices(services ...string) error
	StopAllServices() error
}

// TestServiceFactory is the main structure to interface with test dependencies
type TestServiceFactory struct {
	identifier   string
	Services     []TestService
	compose      *tc.LocalDockerCompose
	composeFiles []string
}

// NewTestFactory is the concrete type for integration tests leveraging
// the pipeline simulator and internal application/device services.
func NewTestFactory(composeFiles []string, identifier string, services []TestService) IntegrationTestFactory {
	return &TestServiceFactory{
		identifier:   identifier,
		Services:     services,
		compose:      tc.NewLocalDockerCompose(composeFiles, identifier),
		composeFiles: composeFiles,
	}
}

// GetDockerServiceNames is necessary for the Consul tests to check for service names based on our Docker Compose service naming convention.
func (f *TestServiceFactory) GetDockerServiceNames() []string {
	var services []string
	for _, service := range f.Services {
		services = append(services, service.Name)
	}
	return services
}

// GetApplicationServiceNames is necessary for checking for services using internal service naming conventions within Consul
func (f *TestServiceFactory) GetApplicationServiceNames() []string {
	return []string{
		"app-file-watcher",
		"app-data-organizer",
		"app-file-sender-oem",
		"app-file-receiver-gateway",
		"app-job-repository",
		"app-task-launcher",
		"app-file-sender-gateway",
		"app-file-receiver-oem",
		"app-pipeline-sim",
		"app-mqtt-export",
	}
}

// GetEdgeXServices returns EdgeX related services
func (f *TestServiceFactory) GetEdgeXServices() []string {
	var services []string
	for _, service := range edgeXServices() {
		services = append(services, service.Name)
	}
	return services
}

// Down invokes the down command on the compose stack.
func (f *TestServiceFactory) Down() error {
	execErr := f.compose.Down()
	return execErr.Error
}

// StartAllServices is used for the startup of integration test factories.
func (f *TestServiceFactory) StartAllServices() error {
	for _, service := range f.Services {
		switch service.Name {
		case pkg.ServiceConsul:
			f.compose.WithExposedService(service.Name, service.Port, wait.NewHTTPStrategy("/v1/catalog/services").WithPort(nat.Port(fmt.Sprintf("%d/tcp", service.Port))).
				WithResponseMatcher(func(body io.Reader) bool {
					return pkg.ConsulResponse(body)
				}).WithMethod(http.MethodGet).WithStartupTimeout(pkg.ContainerWait))
		default:
			f.compose.WithExposedService(service.Name, service.Port, wait.ForLog(service.StartupMessage).WithStartupTimeout(pkg.ContainerWait).WithOccurrence(1))
		}
	}
	execErr := f.compose.WithCommand([]string{"up", "-d"}).Invoke()
	return execErr.Error
}

// GetServiceUrl gets the URL for services when interfacing with them during tests.
func (f *TestServiceFactory) GetServiceUrl(service string) string {
	switch service {
	case pkg.ServiceFileWatcher:
		return fmt.Sprintf("http://-/containers/%s-%s/logs?stdout=1&stderr=1&follow=1", f.identifier, pkg.ServiceFileWatcher)
	}
	return ""
}

// StartServiceWithWait starts a service with a new testcontainers.Wait Strategy.
func (f *TestServiceFactory) StartServiceWithWait(service string) error {
	var execErr tc.ExecError

	command := append([]string{"-p", f.identifier}, "start")
	newCompose := tc.NewLocalDockerCompose(f.composeFiles, f.identifier)

	// wait for first service to be ready
	for k, findServiceDetails := range f.Services {
		if service == findServiceDetails.Name {
			switch f.Services[k].Name {
			case pkg.ServiceConsul:
				f.compose.WithExposedService(f.Services[k].Name, f.Services[k].Port, wait.NewHTTPStrategy("/v1/catalog/services").WithPort(nat.Port(fmt.Sprintf("%d/tcp", f.Services[k].Port))).
					WithResponseMatcher(func(body io.Reader) bool {
						return pkg.ConsulResponse(body)
					}).WithMethod(http.MethodGet).WithStartupTimeout(pkg.ContainerWait))
			default:
				execErr = newCompose.
					WithCommand(command).
					WithExposedService(f.Services[k].Name, f.Services[k].Port, wait.ForLog(f.Services[k].StartupMessage)).
					Invoke()
			}
			break
		}
	}
	return execErr.Error
}

// StartServices starts the specified service(s).
func (f *TestServiceFactory) StartServices(services ...string) error {
	command := append([]string{"-p", f.identifier}, "start")

	for _, service := range services {
		service = strings.TrimSuffix(service, "-1")
		command = append(command, service)
	}

	execErr := f.compose.WithCommand(command).Invoke()
	return execErr.Error
}

// StopServices stops the specified service(s).
func (f *TestServiceFactory) StopServices(services ...string) error {
	command := append([]string{"-p", f.identifier}, "stop")

	for _, service := range services {
		service = strings.TrimSuffix(service, "-1")
		command = append(command, service)
	}

	execErr := f.compose.WithCommand(command).Invoke()
	return execErr.Error
}

// Stop stocks the entire docker stack.
func (f *TestServiceFactory) StopAllServices() error {
	command := append([]string{"-p", f.identifier}, "stop")
	execErr := f.compose.WithCommand(command).Invoke()
	return execErr.Error
}
