// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"fmt"
	"strings"
)

// Request describes one server the steps are for. A nil Want means the
// extensions a probe looks for. PreloadKnown false means the current
// shared_preload_libraries value was not readable, so the steps must not
// replace it.
type Request struct {
	Major          int
	Provider       string
	Patroni        bool
	PreloadKnown   bool
	CurrentPreload []string
	Want           []string
	Installed      map[string]bool
}

// Step is one line of the setup, shown before anything runs.
type Step struct {
	Name string
	// Kind is package, preload, setting, create, restart, or unavailable.
	Kind string
	Text string
}

// MergePreload appends libs that are not already in current, keeping the
// existing order. It does not drop anything already loaded.
func MergePreload(current, libs []string) []string {
	out := append([]string{}, current...)
	seen := map[string]bool{}
	for _, name := range out {
		seen[name] = true
	}
	for _, name := range libs {
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

// Steps returns the setup for req. An unknown name in Want is an error.
// Nothing is saved and nothing is executed.
func Steps(req Request) ([]Step, error) {
	names := req.Want
	if names == nil {
		for _, e := range LookedFor() {
			names = append(names, e.Name)
		}
	}
	var chosen []Extension
	for _, name := range names {
		e, ok := ByName(name)
		if !ok {
			return nil, fmt.Errorf("unknown extension %q", name)
		}
		chosen = append(chosen, e)
	}

	var steps []Step
	var preloadAdd []string
	var settings []Setting
	var creates []Extension
	restart := false

	for _, e := range chosen {
		if !e.Offered(req.Provider, req.Major) {
			why := "not offered"
			if req.Major != 0 && (req.Major < e.MinMajor || req.Major > e.MaxMajor) {
				why = fmt.Sprintf("not built for PostgreSQL %d", req.Major)
			}
			steps = append(steps, Step{
				Name: e.Name, Kind: "unavailable",
				Text: fmt.Sprintf("-- %s is %s on %s", e.Name, why, providerLabel(req.Provider)),
			})
			continue
		}
		preloaded := req.PreloadKnown && contains(req.CurrentPreload, e.Name)
		installed := req.Installed[e.Name]
		ready := (!e.Preload || preloaded) && (!e.Create || installed)
		if e.Preload && !preloaded {
			preloadAdd = append(preloadAdd, e.Name)
			if e.Restart {
				restart = true
			}
		}
		if !ready {
			settings = append(settings, e.Settings...)
		}
		if e.Create && !installed {
			creates = append(creates, e)
		}
		if community(req.Provider) && !e.Contrib && req.Major > 0 && !ready {
			steps = append(steps, Step{
				Name: e.Name, Kind: "package",
				Text: fmt.Sprintf("-- package %s (PGDG)", e.PGDGPackage(req.Major)),
			})
		}
	}

	if len(preloadAdd) > 0 {
		if !req.PreloadKnown {
			steps = append(steps, Step{
				Name: "shared_preload_libraries", Kind: "preload",
				Text: "-- shared_preload_libraries could not be read. Add " + strings.Join(preloadAdd, ", ") + " without removing the current list.",
			})
		} else if req.Patroni {
			steps = append(steps, patroniStep(req.CurrentPreload, preloadAdd, settings))
			settings = nil
		} else {
			merged := MergePreload(req.CurrentPreload, preloadAdd)
			steps = append(steps, Step{
				Name: "shared_preload_libraries", Kind: "preload",
				Text: "ALTER SYSTEM SET shared_preload_libraries = " + sqlString(strings.Join(merged, ",")) + ";",
			})
		}
	}
	if restart {
		steps = append(steps, Step{Name: "restart", Kind: "restart", Text: "-- restart PostgreSQL so the preload change takes effect"})
	}
	for _, s := range settings {
		text := "ALTER SYSTEM SET " + s.Name + " = " + sqlString(s.Value) + ";"
		if req.Patroni {
			text = "patronictl edit-config  # postgresql.parameters." + s.Name + ": " + s.Value
		}
		steps = append(steps, Step{Name: s.Name, Kind: "setting", Text: text})
	}
	for _, e := range creates {
		line := "CREATE EXTENSION IF NOT EXISTS " + e.Name + ";"
		if len(e.Depends) > 0 {
			line += "  -- requires " + strings.Join(e.Depends, ", ")
		}
		steps = append(steps, Step{Name: e.Name, Kind: "create", Text: line})
	}
	return steps, nil
}

func patroniStep(current, add []string, settings []Setting) Step {
	merged := MergePreload(current, add)
	var b strings.Builder
	b.WriteString("patronictl edit-config\n")
	b.WriteString("postgresql:\n  parameters:\n")
	fmt.Fprintf(&b, "    shared_preload_libraries: %s\n", yamlString(strings.Join(merged, ",")))
	for _, s := range settings {
		fmt.Fprintf(&b, "    %s: %s\n", s.Name, yamlString(s.Value))
	}
	return Step{Name: "shared_preload_libraries", Kind: "preload", Text: strings.TrimRight(b.String(), "\n")}
}

func contains(list []string, name string) bool {
	for _, item := range list {
		if item == name {
			return true
		}
	}
	return false
}

func sqlString(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func yamlString(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func community(provider string) bool {
	return provider == "" || provider == ProviderCommunity
}

func providerLabel(provider string) string {
	switch provider {
	case "", ProviderCommunity:
		return "self-managed PostgreSQL"
	case ProviderRDS:
		return "Amazon RDS"
	case ProviderAurora:
		return "Amazon Aurora"
	case ProviderCloudSQL:
		return "Cloud SQL"
	case ProviderAlloyDB:
		return "AlloyDB"
	case ProviderAzure:
		return "Azure Database for PostgreSQL"
	case ProviderNeon:
		return "Neon"
	default:
		return provider
	}
}
