// SPDX-License-Identifier: Apache-2.0

// Package core is the root of the Tuskline core engine: the Apache-2.0 Go
// module shared by the CLI, the agent, the server and the desktop app.
//
// Sub-packages:
//
//	version    build metadata
//	inventory  organisations, projects, environments, policies, clusters
//	conn       connection profiles: TLS, SSH hops, cloud IAM, secret refs
//	collector  scheduled, budgeted collectors over statistics views
//	sqlpack    version-aware SQL packs for PostgreSQL 14-18
//	analyser   health checks and diagnoses with evidence
//	advisor    postgresql.conf linter and tuning advisor
//	lint       lock-aware migration linter
//	provision  plan/apply provisioning planner (Compose, CNPG, SSH)
//	store      embedded SQLite metadata store
//
// The core must never import server/, web/ or desktop/ (AGPL-3.0).
package core
