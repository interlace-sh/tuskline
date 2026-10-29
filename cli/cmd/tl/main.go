// SPDX-License-Identifier: Apache-2.0

// Command tl is the Tuskline CLI (Apache-2.0). It links the core and the
// agent, never the AGPL server: "tl serve" execs the separate tuskline
// binary instead (see docs/adr/0010-binary-licence-boundary.md).
package main

import (
	"os"

	"github.com/interlace-sh/tuskline/cli/internal/cmd"
)

func main() {
	os.Exit(cmd.Execute())
}
