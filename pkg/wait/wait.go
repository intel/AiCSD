/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package wait

import (
	"errors"
	"net"
	"sync"
	"time"

	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/logger"
)

// Note: inspiration and work in this file is derived from https://github.com/roerohan/wait-for-it.
// The aforementioned repo implements the tool as a dependency to be brought in at the Docker layer,
// but the code is adapted and brought into this repo such that it may become an importable dependency
// for our go code. The repo in its current state prevents such usage of it.

// ForDependencies allows the service to wait for its dependencies to be up and ready
// for a configurable amount of time. If the service dependency request timeout is reached
// and the dependent services are not available yet,
// then the timeout wait interval will continue until the dependencies are up for a maximum wait time of 1 minute.
func ForDependencies(lc logger.LoggingClient, dependentServices Services, serviceRequestTimeout time.Duration) error {
	serviceTimeout := int(serviceRequestTimeout.Seconds())
	var err error
	success := make(chan bool, 1)

	if len(dependentServices) > 0 {
		lc.Infof("Service startup timeout invoked to wait %d seconds for dependent services %s", serviceTimeout, dependentServices)
		ok := wait(lc, dependentServices, serviceTimeout)
		if ok {
			success <- true
		} else {
			lc.Infof("Waiting for service dependencies to become available...")
		}
	} else {
		return nil
	}

	// return err if service wait time exceeds ServiceMaxTimeout time
	select {
	case <-success:
		return nil
	case <-time.After(ServiceMaxTimeout):
		err = ErrServiceMaxTimeout
	}
	return err
}

// Wait waits for all services
func wait(lc logger.LoggingClient, services Services, tSeconds int) bool {
	t := time.Duration(tSeconds) * time.Second
	now := time.Now()

	var wg sync.WaitGroup
	wg.Add(len(services))
	success := make(chan bool, 1)

	go func() {
		for _, service := range services {
			go waitOne(lc, service, &wg, now)
		}
		wg.Wait()
		success <- true
	}()

	select {
	case <-success:
		return true
	case <-time.After(t):
		return false
	}
}

func waitOne(lc logger.LoggingClient, service Service, wg *sync.WaitGroup, start time.Time) {
	defer wg.Done()
	for {
		_, err := net.Dial("tcp", string(service))
		if err == nil {
			lc.Infof("%s is available after %s", service, time.Since(start))
			break
		}
		opErr, ok := err.(*net.OpError)
		if ok && errors.Is(err, opErr) {
			lc.Errorf("failed to dial service %s with error: %s", service, opErr.Error())
			break
		}
		time.Sleep(time.Second)
	}
}
