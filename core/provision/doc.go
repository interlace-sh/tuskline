// SPDX-License-Identifier: Apache-2.0

// Package provision plans a deployment and applies it. A plan is a diff of
// files, units and manifests. Apply writes only what the plan names, streams
// a log, and resumes from the last completed step. Compose targets are
// non-production. Secrets are not written here. Milestone M5.1.
package provision
