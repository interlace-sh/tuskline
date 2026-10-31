// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/interlace-sh/tuskline/core/conn"
	"github.com/interlace-sh/tuskline/core/inventory"
)

type connectOpts struct {
	dsn         string
	env         string
	kind        string
	name        string
	passwordEnv string
	output      string
}

// connectResult is the --output json shape. It never holds a password.
type connectResult struct {
	Ref         string          `json:"ref,omitempty"`
	Saved       bool            `json:"saved"`
	Inventory   string          `json:"inventory,omitempty"`
	EnvKind     string          `json:"environment_kind,omitempty"`
	Profile     conn.Profile    `json:"profile"`
	PasswordRef string          `json:"password_ref,omitempty"`
	TLSWarning  string          `json:"tls_warning,omitempty"`
	Server      conn.ServerInfo `json:"server"`
	Missing     []conn.Missing  `json:"missing_extensions"`
}

func newConnectCmd(a *app) *cobra.Command {
	var o connectOpts
	c := &cobra.Command{
		Use:   "connect [env/cluster]",
		Short: "Probe a server and add it to the inventory, or probe an inventory entry",
		Long: `Probe a server and print its summary and missing extensions.

With --dsn, the server is probed directly. Give env/cluster (or --env and
--name) to add it to the inventory or update it; on a terminal you are asked
for anything missing. Without --dsn, env/cluster is looked up in the
inventory and probed again.

Passwords are never stored. A password in the DSN is used for this probe
only; save a reference with --password-env, or use ~/.pgpass.`,
		Example: `  tl connect --dsn "postgres://app@db.example.com/orders" prod/orders
  tl connect --dsn "postgres://tuskline@localhost:5417/tuskline?sslmode=disable" --env dev --name pg17
  tl connect prod/orders --output json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			if err := checkOutput(o.output); err != nil {
				return err
			}
			if o.dsn == "" && len(args) == 0 {
				return errors.New("give env/cluster from the inventory, or --dsn to probe a new server")
			}
			if len(args) == 1 && (o.env != "" || o.name != "") {
				return errors.New("give either env/cluster or --env and --name, not both")
			}
			var ref inventory.Ref
			if len(args) == 1 {
				r, err := inventory.ParseRef(args[0])
				if err != nil {
					return err
				}
				ref = r
			}
			if o.dsn == "" {
				return a.connectExisting(c, ref, o)
			}
			return a.connectDSN(c, ref, o)
		},
	}
	c.Flags().StringVar(&o.dsn, "dsn", "", "libpq URL or keyword/value string to probe")
	c.Flags().StringVar(&o.env, "env", "", "environment to add the server to (with --dsn)")
	c.Flags().StringVar(&o.kind, "kind", "", "kind of a new environment: local, dev, staging or prod (default: the environment name if it is a kind)")
	c.Flags().StringVar(&o.name, "name", "", "cluster name in the inventory (with --dsn)")
	c.Flags().StringVar(&o.passwordEnv, "password-env", "", "environment variable holding the password; stored as a reference, never the value")
	addOutputFlag(c, &o.output)
	return c
}

// connectDSN probes a server given by DSN and, when a target is known,
// adds it to the inventory or updates it.
func (a *app) connectDSN(c *cobra.Command, ref inventory.Ref, o connectOpts) error {
	p, err := conn.ParseDSN(o.dsn)
	if err != nil {
		return err
	}
	if p, err = conn.ResolveService(conn.ServicefilePaths(), p); err != nil {
		return err
	}
	var pwRef *inventory.SecretRef
	if o.passwordEnv != "" {
		pwRef = &inventory.SecretRef{Env: o.passwordEnv}
		if err := pwRef.Validate(); err != nil {
			return err
		}
	}
	dsnHadPassword := p.HasPassword()
	if p, err = withPassword(c, p, pwRef); err != nil {
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

	in := bufio.NewReader(c.InOrStdin())
	if ref.Env == "" {
		ref.Env = o.env
	}
	if ref.Cluster == "" {
		ref.Cluster = o.name
	}
	if a.interactive && (ref.Env == "" || ref.Cluster == "") {
		fmt.Fprintln(c.ErrOrStderr(), "Add this server to the inventory (leave blank to only probe it).")
		if ref.Env == "" {
			ref.Env = prompt(c, in, "Environment", "dev")
		}
		if ref.Cluster == "" {
			ref.Cluster = prompt(c, in, "Cluster name", suggestName(p))
		}
	}
	save := ref.Env != "" && ref.Cluster != ""
	if !save && (ref.Env != "" || ref.Cluster != "") {
		return errors.New("to save the server give both an environment and a cluster name (env/cluster, or --env and --name)")
	}

	settings := conn.SessionSettings{}
	var kind inventory.EnvKind
	if save {
		if err := inventory.ValidateName(ref.Env); err != nil {
			return fmt.Errorf("environment: %w", err)
		}
		if err := inventory.ValidateName(ref.Cluster); err != nil {
			return fmt.Errorf("cluster: %w", err)
		}
		if kind, err = a.envKind(c, in, inv, ref.Env, o.kind); err != nil {
			return err
		}
		settings = inventory.DefaultPolicy(kind).Session()
	}

	warnTLS(c, p)
	info, err := conn.ProbeSession(c.Context(), p, settings)
	if err != nil {
		return err
	}

	res := connectResult{Profile: p.WithPassword(""), Server: info, Missing: info.MissingExtensions(), TLSWarning: p.TLSMode().Warning()}
	if pwRef != nil {
		res.PasswordRef = pwRef.String()
	}
	if save {
		if _, ok := inv.Environment(ref.Env); !ok {
			if _, err := inv.PutEnvironment(ref.Env, kind); err != nil {
				return err
			}
		}
		cl, _ := inv.Cluster(ref)
		cl.Name, cl.Environment = ref.Cluster, ref.Env
		cl.Profile = p.WithPassword("")
		cl.Provider = string(info.Provider)
		if pwRef != nil {
			cl.Password = pwRef
		}
		cl.LastProbe = inventory.RecordProbe(info, a.now())
		if err := inv.PutCluster(cl); err != nil {
			return err
		}
		if err := inv.Save(path); err != nil {
			return err
		}
		res.Ref, res.Saved, res.Inventory, res.EnvKind = ref.String(), true, path, string(kind)
		if dsnHadPassword && pwRef == nil {
			fmt.Fprintln(c.ErrOrStderr(), "note: the password in the DSN was used for this probe only and was not saved; use ~/.pgpass or --password-env")
		}
	}
	return a.printConnect(c, o.output, res)
}

// connectExisting probes a cluster from the inventory under its
// environment's policy and records the result.
func (a *app) connectExisting(c *cobra.Command, ref inventory.Ref, o connectOpts) error {
	if o.kind != "" || o.passwordEnv != "" {
		return errors.New("--kind and --password-env only apply with --dsn")
	}
	sc, err := a.loadCluster(c, ref)
	if err != nil {
		return err
	}
	cl, env, p := sc.cluster, sc.env, sc.profile
	info, err := conn.ProbeSession(c.Context(), p, env.Policy.Session())
	if err != nil {
		return err
	}
	cl.Provider = string(info.Provider)
	cl.LastProbe = inventory.RecordProbe(info, a.now())
	if err := sc.inv.PutCluster(cl); err != nil {
		return err
	}
	if err := sc.inv.Save(sc.path); err != nil {
		return err
	}
	res := connectResult{Ref: ref.String(), Saved: true, Inventory: sc.path, EnvKind: string(env.Kind), Profile: cl.Profile, Server: info, Missing: info.MissingExtensions(), TLSWarning: p.TLSMode().Warning()}
	if cl.Password != nil {
		res.PasswordRef = cl.Password.String()
	}
	return a.printConnect(c, o.output, res)
}

// withPassword fills in the password, in order: already in the profile
// (from the DSN), the secret reference, then ~/.pgpass.
func withPassword(c *cobra.Command, p conn.Profile, ref *inventory.SecretRef) (conn.Profile, error) {
	if p.HasPassword() {
		return p, nil
	}
	if ref != nil {
		pw, err := ref.Resolve()
		if err != nil {
			return p, err
		}
		return p.WithPassword(pw), nil
	}
	pw, err := conn.LookupPassword(conn.PassfilePath(), p)
	if err != nil {
		fmt.Fprintln(c.ErrOrStderr(), "warning:", err)
	}
	return p.WithPassword(pw), nil
}

// envKind decides the kind of the target environment. An existing
// environment keeps its kind: connect never changes it, so a flag cannot
// turn prod into dev.
func (a *app) envKind(c *cobra.Command, in *bufio.Reader, inv *inventory.Inventory, env, flag string) (inventory.EnvKind, error) {
	if e, ok := inv.Environment(env); ok {
		if flag != "" && inventory.EnvKind(flag) != e.Kind {
			return "", fmt.Errorf("environment %s already exists with kind %s; tl connect does not change it", env, e.Kind)
		}
		return e.Kind, nil
	}
	k := inventory.EnvKind(flag)
	if k == "" && inventory.EnvKind(env).Valid() {
		k = inventory.EnvKind(env)
	}
	if k == "" && a.interactive {
		k = inventory.EnvKind(prompt(c, in, fmt.Sprintf("Kind of new environment %s (local, dev, staging, prod)", env), "dev"))
	}
	if k == "" {
		return "", fmt.Errorf("new environment %s: give --kind local, dev, staging or prod", env)
	}
	if !k.Valid() {
		return "", fmt.Errorf("environment kind %q: want local, dev, staging or prod", k)
	}
	return k, nil
}

func warnTLS(c *cobra.Command, p conn.Profile) {
	if w := p.TLSMode().Warning(); w != "" {
		fmt.Fprintln(c.ErrOrStderr(), "warning:", w)
	}
}

// prompt asks on stderr and reads one line; blank input yields def.
func prompt(c *cobra.Command, in *bufio.Reader, label, def string) string {
	if def != "" {
		fmt.Fprintf(c.ErrOrStderr(), "%s [%s]: ", label, def)
	} else {
		fmt.Fprintf(c.ErrOrStderr(), "%s: ", label)
	}
	line, err := in.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return def
	}
	if s := strings.TrimSpace(line); s != "" {
		return s
	}
	return def
}

var nameJunkRE = regexp.MustCompile(`[^a-z0-9-]+`)

// suggestName derives a cluster name from the database or host.
func suggestName(p conn.Profile) string {
	for _, s := range []string{p.Database, strings.Split(p.Host, ".")[0]} {
		n := strings.Trim(nameJunkRE.ReplaceAllString(strings.ToLower(s), "-"), "-")
		if len(n) > 63 {
			n = n[:63]
		}
		if inventory.ValidateName(n) == nil {
			return n
		}
	}
	return ""
}

func (a *app) printConnect(c *cobra.Command, format string, r connectResult) error {
	w := c.OutOrStdout()
	if format == outputJSON {
		if r.Missing == nil {
			r.Missing = []conn.Missing{}
		}
		return writeJSON(w, r)
	}
	s := r.Server
	role := "primary"
	if s.InRecovery {
		role = "standby"
	}
	session := "read-write"
	if s.ReadOnly {
		session = "read-only"
	}
	title := "connected"
	if r.Ref != "" {
		title += ": " + r.Ref
		if r.EnvKind != "" {
			title += " (" + r.EnvKind + ")"
		}
	}
	fmt.Fprintln(w, title)
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "  server\tPostgreSQL %s on %s, %s, %s session\n", serverVersion(s), s.Provider, role, session)
	fmt.Fprintf(tw, "  database\t%s as %s\n", s.Database, s.LoginRole)
	fmt.Fprintf(tw, "  tls\t%s\n", r.Profile.TLSMode())
	fmt.Fprintf(tw, "  privileges\t%s\n", privileges(s.Role))
	preload := "unknown (needs pg_read_all_settings or pg_monitor)"
	if s.PreloadKnown {
		preload = strings.Join(s.SharedPreloadLibraries, ", ")
		if preload == "" {
			preload = "none"
		}
	}
	fmt.Fprintf(tw, "  preload\t%s\n", preload)
	if err := tw.Flush(); err != nil {
		return err
	}
	if len(r.Missing) == 0 {
		fmt.Fprintln(w, "all recommended extensions are ready")
	} else {
		fmt.Fprintln(w, "missing extensions:")
		tw = tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		for _, m := range r.Missing {
			fmt.Fprintf(tw, "  %s\t%s\t%s\n", m.Name, missingStatus(m), m.Unlocks)
		}
		if err := tw.Flush(); err != nil {
			return err
		}
	}
	if r.Saved {
		fmt.Fprintf(w, "saved %s to %s\n", r.Ref, r.Inventory)
	} else {
		fmt.Fprintln(w, "not saved: give env/cluster (or --env and --name) to add it to the inventory")
	}
	return nil
}

func serverVersion(s conn.ServerInfo) string {
	if f := strings.Fields(s.Version); len(f) >= 2 && f[0] == "PostgreSQL" {
		return f[1]
	}
	return fmt.Sprint(s.Major())
}

func privileges(r conn.Role) string {
	var out []string
	switch {
	case r.Superuser:
		out = append(out, "superuser")
	default:
		if r.PGMonitor {
			out = append(out, "pg_monitor")
		}
		if r.PGSignalBackend {
			out = append(out, "pg_signal_backend")
		}
	}
	if len(out) == 0 {
		return "none beyond login (Tuskline sees only this role's own sessions; grant pg_monitor)"
	}
	return strings.Join(out, ", ")
}

func missingStatus(m conn.Missing) string {
	switch {
	case !m.Available:
		return "not available on this server"
	case m.Installed && m.NeedsPreload:
		return "installed; add to shared_preload_libraries (restart)"
	case m.NeedsPreload && !m.Create:
		return "available; add to shared_preload_libraries (restart)"
	case m.NeedsPreload:
		return "available; add to shared_preload_libraries (restart) and CREATE EXTENSION"
	default:
		return "available; CREATE EXTENSION"
	}
}
