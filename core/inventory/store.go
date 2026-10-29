// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"go.yaml.in/yaml/v3"
)

// FileVersion is the inventory file format version.
const FileVersion = 1

// Sentinel errors for inventory lookups and changes.
var (
	ErrNotFound    = errors.New("not found")
	ErrEnvNotEmpty = errors.New("environment still has clusters")
)

// Inventory is the file-backed list of environments and clusters
// ($XDG_CONFIG_HOME/tuskline/inventory.yaml). It holds secret references,
// never secret values.
type Inventory struct {
	Version      int           `json:"version" yaml:"version"`
	Environments []Environment `json:"environments" yaml:"environments"`
	Clusters     []Cluster     `json:"clusters" yaml:"clusters"`
}

// DefaultPath returns $TUSKLINE_INVENTORY if set, else
// $XDG_CONFIG_HOME/tuskline/inventory.yaml, falling back to the OS user
// config directory (~/.config on Linux, ~/Library/Application Support on
// macOS, %AppData% on Windows).
func DefaultPath() (string, error) {
	if p := os.Getenv("TUSKLINE_INVENTORY"); p != "" {
		return p, nil
	}
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		var err error
		if dir, err = os.UserConfigDir(); err != nil {
			return "", fmt.Errorf("inventory path: %w", err)
		}
	}
	return filepath.Join(dir, "tuskline", "inventory.yaml"), nil
}

// Load reads the inventory at path. A missing file is an empty inventory.
// Unknown keys are rejected, so a typo never silently drops a setting, and a
// plaintext secret fails with ErrPlaintextSecret.
func Load(path string) (*Inventory, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return &Inventory{Version: FileVersion}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read inventory %s: %w", path, err)
	}
	inv, err := Parse(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("inventory %s: %w", path, err)
	}
	return inv, nil
}

// Parse decodes and validates an inventory document.
func Parse(r io.Reader) (*Inventory, error) {
	var inv Inventory
	dec := yaml.NewDecoder(r)
	dec.KnownFields(true)
	if err := dec.Decode(&inv); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if inv.Version == 0 {
		inv.Version = FileVersion
	}
	if inv.Version != FileVersion {
		return nil, fmt.Errorf("unsupported inventory version %d (this build reads %d)", inv.Version, FileVersion)
	}
	for i := range inv.Environments {
		inv.Environments[i].Policy = DefaultPolicy(inv.Environments[i].Kind)
	}
	if err := inv.Validate(); err != nil {
		return nil, err
	}
	return &inv, nil
}

// Validate checks names, kinds, references between clusters and
// environments, connection profiles and secret references.
func (inv *Inventory) Validate() error {
	var errs []error
	envs := map[string]bool{}
	for _, e := range inv.Environments {
		if err := ValidateName(e.Name); err != nil {
			errs = append(errs, fmt.Errorf("environment: %w", err))
		}
		if !e.Kind.Valid() {
			errs = append(errs, fmt.Errorf("environment %s: invalid kind %q (local, dev, staging or prod)", e.Name, e.Kind))
		}
		if envs[e.Name] {
			errs = append(errs, fmt.Errorf("environment %s: defined twice", e.Name))
		}
		envs[e.Name] = true
	}
	refs := map[Ref]bool{}
	for _, c := range inv.Clusters {
		if err := ValidateName(c.Name); err != nil {
			errs = append(errs, fmt.Errorf("cluster: %w", err))
		}
		if !envs[c.Environment] {
			errs = append(errs, fmt.Errorf("cluster %s: environment %q: %w", c.Ref(), c.Environment, ErrNotFound))
		}
		if refs[c.Ref()] {
			errs = append(errs, fmt.Errorf("cluster %s: defined twice", c.Ref()))
		}
		refs[c.Ref()] = true
		if err := c.Profile.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("cluster %s: %w", c.Ref(), err))
		}
		if c.Profile.HasPassword() {
			errs = append(errs, fmt.Errorf("cluster %s: %w", c.Ref(), ErrPlaintextSecret))
		}
		if c.Password != nil {
			if err := c.Password.Validate(); err != nil {
				errs = append(errs, fmt.Errorf("cluster %s: %w", c.Ref(), err))
			}
		}
	}
	return errors.Join(errs...)
}

// Save validates inv and writes it to path atomically with mode 0600,
// creating the directory with mode 0700.
func (inv *Inventory) Save(path string) error {
	if err := inv.Validate(); err != nil {
		return err
	}
	inv.sort()
	var buf bytes.Buffer
	buf.WriteString("# Tuskline inventory. Holds secret references, never secret values.\n")
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(inv); err != nil {
		return fmt.Errorf("encode inventory: %w", err)
	}
	if err := enc.Close(); err != nil {
		return fmt.Errorf("encode inventory: %w", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".inventory-*.yaml")
	if err != nil {
		return fmt.Errorf("write inventory: %w", err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }() // no-op after a successful rename
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write inventory: %w", err)
	}
	if _, err := tmp.Write(buf.Bytes()); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write inventory: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("write inventory: %w", err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("write inventory: %w", err)
	}
	return nil
}

func (inv *Inventory) sort() {
	sort.Slice(inv.Environments, func(i, j int) bool { return inv.Environments[i].Name < inv.Environments[j].Name })
	sort.Slice(inv.Clusters, func(i, j int) bool {
		a, b := inv.Clusters[i], inv.Clusters[j]
		if a.Environment != b.Environment {
			return a.Environment < b.Environment
		}
		return a.Name < b.Name
	})
}

// Environment returns the named environment.
func (inv *Inventory) Environment(name string) (Environment, bool) {
	for _, e := range inv.Environments {
		if e.Name == name {
			return e, true
		}
	}
	return Environment{}, false
}

// PutEnvironment adds an environment or changes its kind. The policy is
// always the kind's default.
func (inv *Inventory) PutEnvironment(name string, kind EnvKind) (Environment, error) {
	if err := ValidateName(name); err != nil {
		return Environment{}, err
	}
	if !kind.Valid() {
		return Environment{}, fmt.Errorf("environment %s: invalid kind %q (local, dev, staging or prod)", name, kind)
	}
	e := Environment{Name: name, Kind: kind, Policy: DefaultPolicy(kind)}
	for i := range inv.Environments {
		if inv.Environments[i].Name == name {
			inv.Environments[i] = e
			return e, nil
		}
	}
	inv.Environments = append(inv.Environments, e)
	return e, nil
}

// RemoveEnvironment deletes an environment that has no clusters.
func (inv *Inventory) RemoveEnvironment(name string) error {
	for _, c := range inv.Clusters {
		if c.Environment == name {
			return fmt.Errorf("environment %s: %w", name, ErrEnvNotEmpty)
		}
	}
	for i, e := range inv.Environments {
		if e.Name == name {
			inv.Environments = append(inv.Environments[:i], inv.Environments[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("environment %s: %w", name, ErrNotFound)
}

// Cluster returns the cluster at ref.
func (inv *Inventory) Cluster(ref Ref) (Cluster, bool) {
	for _, c := range inv.Clusters {
		if c.Ref() == ref {
			return c, true
		}
	}
	return Cluster{}, false
}

// PutCluster adds a cluster or replaces the one with the same reference.
// Its environment must exist, and its profile must not hold a password.
func (inv *Inventory) PutCluster(c Cluster) error {
	if err := ValidateName(c.Name); err != nil {
		return err
	}
	if _, ok := inv.Environment(c.Environment); !ok {
		return fmt.Errorf("cluster %s: environment %q: %w", c.Ref(), c.Environment, ErrNotFound)
	}
	if c.Profile.HasPassword() {
		return fmt.Errorf("cluster %s: %w", c.Ref(), ErrPlaintextSecret)
	}
	if err := c.Profile.Validate(); err != nil {
		return fmt.Errorf("cluster %s: %w", c.Ref(), err)
	}
	if c.Password != nil {
		if err := c.Password.Validate(); err != nil {
			return fmt.Errorf("cluster %s: %w", c.Ref(), err)
		}
	}
	for i := range inv.Clusters {
		if inv.Clusters[i].Ref() == c.Ref() {
			inv.Clusters[i] = c
			return nil
		}
	}
	inv.Clusters = append(inv.Clusters, c)
	return nil
}

// RemoveCluster deletes the cluster at ref.
func (inv *Inventory) RemoveCluster(ref Ref) error {
	for i, c := range inv.Clusters {
		if c.Ref() == ref {
			inv.Clusters = append(inv.Clusters[:i], inv.Clusters[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("cluster %s: %w", ref, ErrNotFound)
}

// List returns every cluster sorted by environment and name.
func (inv *Inventory) List() []Cluster {
	out := append([]Cluster(nil), inv.Clusters...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Environment != out[j].Environment {
			return out[i].Environment < out[j].Environment
		}
		return out[i].Name < out[j].Name
	})
	return out
}
