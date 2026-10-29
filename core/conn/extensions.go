// SPDX-License-Identifier: Apache-2.0

package conn

// Recommendation is an extension Tuskline uses (docs/onboarding-and-deploy.md).
type Recommendation struct {
	Name string `json:"name"`
	// Unlocks says what Tuskline can do once it is installed.
	Unlocks string `json:"unlocks"`
	// Required extensions are needed for core features.
	Required bool `json:"required"`
	// Preload is true when the library must be in shared_preload_libraries.
	Preload bool `json:"preload"`
}

// Recommended lists the extensions Tuskline looks for, most important first.
var Recommended = []Recommendation{
	{Name: "pg_stat_statements", Unlocks: "query insights, load attribution and workload snapshots", Required: true, Preload: true},
	{Name: "pg_wait_sampling", Unlocks: "wait-event history between samples", Preload: true},
	{Name: "pg_stat_kcache", Unlocks: "CPU and real disk I/O per query", Preload: true},
	{Name: "hypopg", Unlocks: "index advice validated with hypothetical indexes"},
}

// Missing is a recommended extension that is not ready to use.
type Missing struct {
	Recommendation
	// Available is true when the server has the extension's files, so only
	// CREATE EXTENSION (and a preload, if needed) is missing.
	Available bool `json:"available"`
	// NeedsPreload is true when the library is not in
	// shared_preload_libraries (which means a restart), or that could not
	// be checked.
	NeedsPreload bool `json:"needs_preload"`
	// Installed is true when CREATE EXTENSION has run but the preload is
	// still missing.
	Installed bool `json:"installed"`
}

// MissingExtensions returns the recommended extensions that are not
// installed in the current database or not preloaded where they must be.
func (s ServerInfo) MissingExtensions() []Missing {
	var out []Missing
	for _, r := range Recommended {
		e, _ := s.Extension(r.Name)
		needsPreload := r.Preload && (!s.PreloadKnown || !s.Preloaded(r.Name))
		if e.Installed() && !needsPreload {
			continue
		}
		out = append(out, Missing{Recommendation: r, Available: e.Available, NeedsPreload: needsPreload, Installed: e.Installed()})
	}
	return out
}
