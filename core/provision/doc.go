// SPDX-License-Identifier: Apache-2.0

// Package provision plans a deployment and applies it. A plan is a diff of
// files, units and manifests. Apply writes only what the plan names, streams
// a log, and resumes from the last completed step. Compose targets are
// non-production. Compose generates that stack. Secrets are written only
// through a secret store, as mode 0600 files for a local Compose directory.
// Milestones M5.1, M5.2a, M5.2b and M5.3.
package provision
