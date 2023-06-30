/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package wait

import (
	"fmt"
	"time"
)

// Services is a string array storing
// the services that are to be waited for
type Services []Service
type Service string

// List of host:port services that are dependencies for other services
const (
	ServiceTaskLauncher Service = "task-launcher:59785"
	ServiceJobRepo      Service = "job-repository:59784"
	ServiceConsul       Service = "edgex-core-consul:8500"
	ServiceRedis        Service = "edgex-redis:6379"
)

// ServiceMaxTimeout is a timer for the service dependency waiting logic
const (
	ServiceMaxTimeout = 1 * time.Minute
)

var (
	ErrServiceMaxTimeout = fmt.Errorf("max service startup timeout duration of %d exceeded waiting for service dependencies", ServiceMaxTimeout)
)
