// SPDX-License-Identifier: Apache-2.0

//go:build !windows

package cmd

import (
	"os"
	"syscall"
)

// forwardedSignals are passed from tl to the server it runs.
var forwardedSignals = []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGHUP}
