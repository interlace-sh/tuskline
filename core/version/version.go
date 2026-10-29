// SPDX-License-Identifier: Apache-2.0

// Package version holds build metadata, set at link time with -ldflags.
package version

import "fmt"

// These are overridden at build time, e.g.
//
//	-ldflags "-X github.com/interlace-sh/tuskline/core/version.Version=v0.1.0"
var (
	Version = "0.0.0-dev"
	Commit  = "none"
	Date    = "unknown"
)

// String returns a one-line, human-readable version.
func String() string {
	return fmt.Sprintf("%s (commit %s, built %s)", Version, Commit, Date)
}
