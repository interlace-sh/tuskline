// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"fmt"
	"strings"
)

// PlanLine is one line of an install preview. Apply is a later milestone;
// producing a plan does not run it.
type PlanLine struct {
	Name string
	// Kind is note, repo, package, sql, command, or unavailable.
	Kind string
	Text string
}

// InstallPlan is the preview for req: packages, then the configuration and
// SQL. Managed providers get their own command, or a line that the extension
// is not offered. Nothing is compiled from source.
func InstallPlan(req Request) ([]PlanLine, error) {
	steps, err := Steps(req)
	if err != nil {
		return nil, err
	}
	lines := []PlanLine{{
		Kind: "note",
		Text: "# Preview only. Nothing here has been run.",
	}}
	if community(req.Provider) {
		lines = append(lines, selfManagedPackages(req)...)
		for _, step := range steps {
			if step.Kind == "package" {
				continue
			}
			lines = append(lines, PlanLine{Name: step.Name, Kind: "sql", Text: step.Text})
		}
		return lines, nil
	}
	lines = append(lines, managedCommands(req)...)
	for _, step := range steps {
		switch step.Kind {
		case "create", "restart":
			lines = append(lines, PlanLine{Name: step.Name, Kind: "sql", Text: step.Text})
		case "unavailable":
			lines = append(lines, PlanLine{Name: step.Name, Kind: "unavailable", Text: "-- " + step.Name + " is not offered here"})
		}
	}
	return lines, nil
}

func selfManagedPackages(req Request) []PlanLine {
	var lines []PlanLine
	var pkgs []Extension
	for _, e := range wanted(req) {
		if !needsInstall(req, e) || e.Contrib {
			continue
		}
		pkgs = append(pkgs, e)
		if e.Repo == "pigsty" {
			lines = append(lines, PlanLine{
				Name: e.Name, Kind: "repo",
				Text: "# PGDG does not ship " + e.Name + ". Add the Pigsty DEB or RPM repository (https://pigsty.io/docs/install/repo), which publishes that package name.",
			})
		}
	}
	for _, e := range pkgs {
		name := e.PGDGPackage(req.Major)
		note := "PGDG build"
		if e.Repo == "pigsty" {
			note = "Pigsty build"
		}
		lines = append(lines, PlanLine{
			Name: e.Name, Kind: "package",
			Text: "sudo apt-get install -y " + name + "\n# RHEL/Fedora: install the " + note + " of " + name + ". Do not compile it.",
		})
	}
	return lines
}

func managedCommands(req Request) []PlanLine {
	var preload []string
	var settings []Setting
	var creates []string
	for _, e := range wanted(req) {
		if !e.Offered(req.Provider, req.Major) || extensionReady(req, e) {
			continue
		}
		if e.Preload && !(req.PreloadKnown && contains(req.CurrentPreload, e.Name)) {
			preload = append(preload, e.Name)
		}
		if e.Create && !req.Installed[e.Name] {
			creates = append(creates, e.Name)
		}
		settings = append(settings, e.Settings...)
	}
	if len(preload) == 0 && len(settings) == 0 {
		return nil
	}
	merged := ""
	if req.PreloadKnown {
		merged = strings.Join(MergePreload(req.CurrentPreload, preload), ",")
	}
	text := providerCommand(req.Provider, merged, req.PreloadKnown, settings, creates)
	if text == "" {
		return nil
	}
	return []PlanLine{{Name: req.Provider, Kind: "command", Text: text}}
}

func providerCommand(provider, merged string, preloadKnown bool, settings []Setting, creates []string) string {
	var b strings.Builder
	switch provider {
	case ProviderRDS:
		writeRDS(&b, "modify-db-parameter-group", "--db-parameter-group-name <parameter-group>", merged, preloadKnown, settings)
	case ProviderAurora:
		writeRDS(&b, "modify-db-cluster-parameter-group", "--db-cluster-parameter-group-name <cluster-parameter-group>", merged, preloadKnown, settings)
	case ProviderCloudSQL:
		if !preloadKnown {
			b.WriteString("# shared_preload_libraries could not be read. Include the current flags and add the new libraries.\n")
		}
		b.WriteString("# --database-flags replaces the instance flag list. Repeat flags that are already set.\n")
		b.WriteString("gcloud sql instances patch <instance> --database-flags=")
		b.WriteString(flagList(merged, preloadKnown, settings))
	case ProviderAlloyDB:
		b.WriteString("# --database-flags replaces existing flags. Include flags already set.\n")
		b.WriteString("gcloud alloydb instances update <instance> --cluster=<cluster> --region=<region> --database-flags=")
		b.WriteString(flagList(merged, preloadKnown, settings))
	case ProviderAzure:
		if preloadKnown && merged != "" {
			fmt.Fprintf(&b, "az postgres flexible-server parameter set --resource-group <resource-group> --server-name <server> --name shared_preload_libraries --value %s\n", sqlString(merged))
		} else if !preloadKnown {
			b.WriteString("# shared_preload_libraries could not be read. Set it in the server parameters without removing the current list.\n")
		}
		if len(creates) > 0 {
			fmt.Fprintf(&b, "az postgres flexible-server parameter set --resource-group <resource-group> --server-name <server> --name azure.extensions --value %s\n", sqlString(strings.Join(creates, ",")))
			b.WriteString("# Keep extension names already listed in azure.extensions.\n")
		}
		for _, s := range settings {
			fmt.Fprintf(&b, "az postgres flexible-server parameter set --resource-group <resource-group> --server-name <server> --name %s --value %s\n", s.Name, sqlString(s.Value))
		}
	case ProviderNeon:
		if preloadKnown && merged != "" {
			fmt.Fprintf(&b, "# In the Neon console set shared_preload_libraries to %s\n", merged)
		} else if !preloadKnown {
			b.WriteString("# shared_preload_libraries could not be read. Add the new libraries in the Neon console without removing the current list.\n")
		}
		b.WriteString("# Neon has no package install. Run the SQL after the extension is allowed on the project.")
	default:
		return ""
	}
	return strings.TrimRight(b.String(), "\n")
}

func writeRDS(b *strings.Builder, action, target, merged string, preloadKnown bool, settings []Setting) {
	if preloadKnown && merged != "" {
		fmt.Fprintf(b, "aws rds %s %s --parameters ParameterName=shared_preload_libraries,ParameterValue=%s,ApplyMethod=pending-reboot\n", action, target, sqlString(merged))
	} else if !preloadKnown {
		b.WriteString("# shared_preload_libraries could not be read. Add the new libraries in the parameter group without removing the current list.\n")
	}
	for _, s := range settings {
		fmt.Fprintf(b, "aws rds %s %s --parameters ParameterName=%s,ParameterValue=%s,ApplyMethod=immediate\n", action, target, s.Name, sqlString(s.Value))
	}
}

func flagList(merged string, preloadKnown bool, settings []Setting) string {
	var parts []string
	if preloadKnown && merged != "" {
		parts = append(parts, "shared_preload_libraries="+merged)
	}
	for _, s := range settings {
		parts = append(parts, s.Name+"="+s.Value)
	}
	return strings.Join(parts, ",")
}

func wanted(req Request) []Extension {
	names := req.Want
	if names == nil {
		for _, e := range LookedFor() {
			names = append(names, e.Name)
		}
	}
	var out []Extension
	for _, name := range names {
		if e, ok := ByName(name); ok {
			out = append(out, e)
		}
	}
	return out
}

func needsInstall(req Request, e Extension) bool {
	return e.Offered(req.Provider, req.Major) && !extensionReady(req, e)
}

func extensionReady(req Request, e Extension) bool {
	preloaded := req.PreloadKnown && contains(req.CurrentPreload, e.Name)
	installed := req.Installed[e.Name]
	return (!e.Preload || preloaded) && (!e.Create || installed)
}
