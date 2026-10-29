// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"github.com/interlace-sh/tuskline/core/inventory"
)

// lsResult is the --output json shape.
type lsResult struct {
	Inventory    string                  `json:"inventory"`
	Environments []inventory.Environment `json:"environments"`
	Clusters     []inventory.Cluster     `json:"clusters"`
}

func newLsCmd(a *app) *cobra.Command {
	var output string
	c := &cobra.Command{
		Use:   "ls",
		Short: "List environments and clusters in the inventory",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			if err := checkOutput(output); err != nil {
				return err
			}
			path, err := a.inventoryFile()
			if err != nil {
				return err
			}
			inv, err := inventory.Load(path)
			if err != nil {
				return err
			}
			if output == outputJSON {
				res := lsResult{Inventory: path, Environments: inv.Environments, Clusters: inv.List()}
				if res.Environments == nil {
					res.Environments = []inventory.Environment{}
				}
				if res.Clusters == nil {
					res.Clusters = []inventory.Cluster{}
				}
				return writeJSON(c.OutOrStdout(), res)
			}
			return a.printLs(c.OutOrStdout(), path, inv)
		},
	}
	addOutputFlag(c, &output)
	return c
}

func (a *app) printLs(w io.Writer, path string, inv *inventory.Inventory) error {
	if len(inv.Environments) == 0 {
		_, err := fmt.Fprintf(w, "The inventory (%s) is empty. Add a server with:\n  tl connect --dsn <url> <env>/<cluster>\n", path)
		return err
	}
	now := a.now()
	header := []string{"ENVIRONMENT", "CLUSTER", "TARGET", "VERSION", "PROVIDER", "ROLE", "LAST PROBE"}
	rows := [][]string{}
	kinds := []inventory.EnvKind{}
	clusters := inv.List()
	for _, env := range inv.Environments {
		label := env.Name
		if string(env.Kind) != env.Name {
			label += " (" + string(env.Kind) + ")"
		}
		n := 0
		for _, cl := range clusters {
			if cl.Environment != env.Name {
				continue
			}
			n++
			rows = append(rows, append([]string{label, cl.Name, target(cl)}, probeColumns(cl.LastProbe, cl.Provider, now)...))
			kinds = append(kinds, env.Kind)
		}
		if n == 0 {
			rows = append(rows, []string{label, "-", "", "", "", "", ""})
			kinds = append(kinds, env.Kind)
		}
	}

	widths := make([]int, len(header))
	for _, r := range append([][]string{header}, rows...) {
		for i, cell := range r {
			widths[i] = max(widths[i], utf8.RuneCountInString(cell))
		}
	}
	line := func(r []string, kind inventory.EnvKind) string {
		var b strings.Builder
		for i, cell := range r {
			pad := strings.Repeat(" ", widths[i]-utf8.RuneCountInString(cell))
			if i == 0 && kind != "" {
				cell = a.envColour(kind, cell)
			}
			if i == len(r)-1 {
				b.WriteString(cell)
				break
			}
			b.WriteString(cell + pad + "  ")
		}
		return strings.TrimRight(b.String(), " ") + "\n"
	}
	if _, err := io.WriteString(w, line(header, "")); err != nil {
		return err
	}
	for i, r := range rows {
		if _, err := io.WriteString(w, line(r, kinds[i])); err != nil {
			return err
		}
	}
	return nil
}

// target is where the cluster lives, without credentials.
func target(cl inventory.Cluster) string {
	p := cl.Profile
	if p.Host == "" && p.Service != "" {
		return "service=" + p.Service
	}
	host := p.Host
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "/") {
		host = "[" + host + "]"
	}
	s := host + ":" + strconv.Itoa(int(p.EffectivePort()))
	if p.Database != "" {
		s += "/" + p.Database
	}
	return s
}

func probeColumns(r *inventory.ProbeRecord, provider string, now time.Time) []string {
	if r == nil {
		return []string{"", provider, "", "never"}
	}
	role := "primary"
	if r.InRecovery {
		role = "standby"
	}
	return []string{fmt.Sprintf("%d.%d", r.VersionNum/10000, r.VersionNum%10000), string(r.Provider), role, age(now.Sub(r.At))}
}

// age is a short, rounded duration such as "4m ago".
func age(d time.Duration) string {
	switch {
	case d < 0:
		return "just now"
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

// envRGB holds the environment colours from docs/design.md.
var envRGB = map[inventory.EnvKind][3]int{
	inventory.EnvProd:    {0xE5, 0x48, 0x4D},
	inventory.EnvStaging: {0xF5, 0xA5, 0x24},
	inventory.EnvDev:     {0x30, 0xA4, 0x6C},
	inventory.EnvLocal:   {0x8B, 0x8D, 0x98},
}

func (a *app) envColour(k inventory.EnvKind, s string) string {
	rgb, ok := envRGB[k]
	if !a.colour || !ok {
		return s
	}
	return fmt.Sprintf("\x1b[1;38;2;%d;%d;%dm%s\x1b[0m", rgb[0], rgb[1], rgb[2], s)
}
