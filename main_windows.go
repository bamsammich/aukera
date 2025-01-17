// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build windows

package main

import (
	"fmt"

	"./auklib"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc"
	"github.com/google/logger"
)

// Type winSvc implements svc.Handler.
type winSvc struct{}

func startService(isDebug bool) error {
	logger.Infof("Starting %s service.", auklib.ServiceName)
	run := svc.Run
	if isDebug {
		run = debug.Run
	}
	if err := run(auklib.ServiceName, winSvc{}); err != nil {
		return fmt.Errorf("%s service failed: %v", auklib.ServiceName, err)
	}
	logger.Infof("%s service stopped.", auklib.ServiceName)
	return nil
}

// Execute starts the internal goroutine and waits for service
// signals from Windows. Execute is called by svc.Run which runs
// in a loop itself and interprets data in the changes channel
// for windows. When we receive a command to Stop or Shutdown,
// we break out of the loop and send a StopPending status to
// Windows, which will stop the service process and all child processes.
func (m winSvc) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	var (
		ssec  bool
		errno uint32
	)
	errch := make(chan error)

	changes <- svc.Status{State: svc.StartPending}
	go func() {
		errch <- runMainLoop()
	}()
	logger.Infof("Service started.")

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
loop:
	for {
		select {
		// Watch for the aukera goroutine to fail for some reason.
		case err := <-errch:
			logger.Errorf("%s goroutine has failed: %v", auklib.ServiceName, err)
			break loop
		// Watch for service signals.
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				break loop
			case svc.Pause:
				changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
			case svc.Continue:
				changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
			default:
				logger.Errorf("unexpected control request #%d", c)
			}
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	return ssec, errno
}

func run() error {
	isIntSess, err := svc.IsAnInteractiveSession()
	if err != nil {
		return fmt.Errorf("Failed to determine if running in an interactive session: %v", err)
	}
	// Running as Service
	if !isIntSess {
		return startService(*runInDebug)
	}
	return fmt.Errorf("interactive sessions are unsupported")
}
