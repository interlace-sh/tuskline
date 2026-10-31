// SPDX-License-Identifier: Apache-2.0

package conn

import "github.com/interlace-sh/tuskline/core/catalog"

// Recommendation is an extension Tuskline uses (docs/onboarding-and-deploy.md).
type Recommendation struct {
	Name string `json:"name"`
	// Unlocks says what Tuskline can do once it is installed.
	Unlocks string `json:"unlocks"`
	// Required extensions are needed for core features.
	Required bool `json:"required"`
	// Preload is true when the library must be in shared_preload_libraries.
	Preload bool `json:"preload"`
	// Create is false for libraries that are not CREATE EXTENSION, such as auto_explain.
	Create bool `json:"create"`
}

// Recommended lists the extensions Tuskline looks for, most important first.
// It is the catalogue's looked-for entries.
var Recommended = recommendations()

func recommendations() []Recommendation {
	looked := catalog.LookedFor()
	out := make([]Recommendation, len(looked))
	for i, e := range looked {
		out[i] = Recommendation{Name: e.Name, Unlocks: e.Unlocks, Required: e.Required, Preload: e.Preload, Create: e.Create}
	}
	return out
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
		available := e.Available
		if !r.Create {
			if !needsPreload {
				continue
			}
			out = append(out, Missing{Recommendation: r, Available: true, NeedsPreload: true})
			continue
		}
		if e.Installed() && !needsPreload {
			continue
		}
		out = append(out, Missing{Recommendation: r, Available: available, NeedsPreload: needsPreload, Installed: e.Installed()})
	}
	return out
}
