// SPDX-License-Identifier: Apache-2.0

//go:build windows

package cmd

import "os"

// forwardedSignals are passed from tl to the server it runs. Windows
// delivers ctrl+c to the whole console, so this mostly keeps tl alive
// until the server has exited.
var forwardedSignals = []os.Signal{os.Interrupt}
