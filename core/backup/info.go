// SPDX-License-Identifier: Apache-2.0

package backup

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/interlace-sh/tuskline/core/inventory"
)

// Backup is one base backup.
type Backup struct {
	Label string
	Type  string
	Start time.Time
	Stop  time.Time
}

// Span is a contiguous WAL range, named by pgBackRest archive ids.
type Span struct {
	Min string
	Max string
}

// Gap is a missing WAL range between two spans.
type Gap struct {
	After  string
	Before string
}

// Stanza is one pgBackRest stanza.
type Stanza struct {
	Name    string
	Backups []Backup
	Archive []Span
	Gaps    []Gap
}

// Report is the parsed backup view.
type Report struct {
	Stanzas []Stanza
}

// ParseInfo reads `pgbackrest info --output=json`.
func ParseInfo(raw []byte) (Report, error) {
	var docs []struct {
		Name   string `json:"name"`
		Backup []struct {
			Label     string `json:"label"`
			Type      string `json:"type"`
			Timestamp struct {
				Start int64 `json:"start"`
				Stop  int64 `json:"stop"`
			} `json:"timestamp"`
		} `json:"backup"`
		Archive []struct {
			Min string `json:"min"`
			Max string `json:"max"`
		} `json:"archive"`
	}
	if err := json.Unmarshal(raw, &docs); err != nil {
		return Report{}, fmt.Errorf("pgbackrest info: %w", err)
	}
	var report Report
	for _, doc := range docs {
		if doc.Name == "" {
			return Report{}, fmt.Errorf("pgbackrest info: a stanza has no name")
		}
		stanza := Stanza{Name: doc.Name}
		for _, item := range doc.Backup {
			stanza.Backups = append(stanza.Backups, Backup{
				Label: item.Label,
				Type:  item.Type,
				Start: time.Unix(item.Timestamp.Start, 0).UTC(),
				Stop:  time.Unix(item.Timestamp.Stop, 0).UTC(),
			})
		}
		for _, item := range doc.Archive {
			if item.Min == "" || item.Max == "" {
				continue
			}
			stanza.Archive = append(stanza.Archive, Span{Min: item.Min, Max: item.Max})
		}
		stanza.Gaps = gaps(stanza.Archive)
		report.Stanzas = append(report.Stanzas, stanza)
	}
	if len(report.Stanzas) == 0 {
		return Report{}, fmt.Errorf("pgbackrest info: no stanzas")
	}
	return report, nil
}

func gaps(spans []Span) []Gap {
	var out []Gap
	for i := 1; i < len(spans); i++ {
		if hole(spans[i-1].Max, spans[i].Min) {
			out = append(out, Gap{After: spans[i-1].Max, Before: spans[i].Min})
		}
	}
	return out
}

// hole reports whether next starts more than one WAL id after prev.
// Equal or overlapping ranges are not a hole. Unparseable ids are a hole
// only when the text is not contiguous.
func hole(prev, next string) bool {
	if next <= prev {
		return false
	}
	left, okLeft := new(big.Int).SetString(prev, 16)
	right, okRight := new(big.Int).SetString(next, 16)
	if !okLeft || !okRight {
		return true
	}
	left.Add(left, big.NewInt(1))
	return right.Cmp(left) > 0
}

// CNPGBackup is one CloudNativePG backup object.
type CNPGBackup struct {
	Name  string
	Phase string
}

// ParseCNPG reads a list of CloudNativePG backup objects.
func ParseCNPG(raw []byte) ([]CNPGBackup, error) {
	var doc struct {
		Items []struct {
			Name  string `json:"name"`
			Phase string `json:"phase"`
		} `json:"items"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("cnpg backups: %w", err)
	}
	var out []CNPGBackup
	for _, item := range doc.Items {
		if item.Name == "" {
			return nil, fmt.Errorf("cnpg backups: an item has no name")
		}
		out = append(out, CNPGBackup{Name: item.Name, Phase: item.Phase})
	}
	return out, nil
}

// Text renders the report.
func (r Report) Text() string {
	var b strings.Builder
	for _, stanza := range r.Stanzas {
		fmt.Fprintf(&b, "stanza %s\n", stanza.Name)
		for _, item := range stanza.Backups {
			fmt.Fprintf(&b, "  %s %s %s\n", item.Type, item.Label, item.Stop.Format(time.RFC3339))
		}
		for _, gap := range stanza.Gaps {
			fmt.Fprintf(&b, "  gap %s .. %s\n", gap.After, gap.Before)
		}
	}
	return b.String()
}

// Restore is a point-in-time restore preview.
type Restore struct {
	Stanza  string
	To      time.Time
	Prod    bool
	Confirm string
	Cluster string
}

// Preview is the pgBackRest command. It is not run.
func (r Restore) Preview(report Report) (string, error) {
	if err := r.Check(report); err != nil {
		return "", err
	}
	return fmt.Sprintf("pgbackrest --stanza=%s restore --type=time --target=%s\nNothing has been restored.", r.Stanza, r.To.UTC().Format(time.RFC3339)), nil
}

// Check refuses a target inside a WAL gap and a prod restore without the cluster name.
func (r Restore) Check(report Report) error {
	if r.Stanza == "" || r.To.IsZero() {
		return fmt.Errorf("restore needs a stanza and a target time")
	}
	if err := inventory.ValidateName(r.Stanza); err != nil {
		return fmt.Errorf("stanza: %w", err)
	}
	stanza, ok := findStanza(report, r.Stanza)
	if !ok {
		return fmt.Errorf("stanza %s is not in the backup report", r.Stanza)
	}
	if len(stanza.Backups) == 0 {
		return fmt.Errorf("stanza %s has no base backup", r.Stanza)
	}
	var base *Backup
	for i := range stanza.Backups {
		item := &stanza.Backups[i]
		if !item.Stop.After(r.To) {
			if base == nil || item.Stop.After(base.Stop) {
				base = item
			}
		}
	}
	if base == nil {
		return fmt.Errorf("no base backup finishes at or before %s", r.To.UTC().Format(time.RFC3339))
	}
	if len(stanza.Gaps) > 0 {
		return fmt.Errorf("stanza %s has a WAL gap after %s; pick another target", r.Stanza, stanza.Gaps[0].After)
	}
	if r.Prod && r.Confirm != r.Cluster {
		return fmt.Errorf("type the cluster name to confirm the restore")
	}
	return nil
}

func findStanza(report Report, name string) (Stanza, bool) {
	for _, stanza := range report.Stanzas {
		if stanza.Name == name {
			return stanza, true
		}
	}
	return Stanza{}, false
}

// VerifyCommand is the scratch-container restore drill. It does not include a password.
func VerifyCommand(stanza, image string) ([]string, error) {
	if err := inventory.ValidateName(stanza); err != nil {
		return nil, fmt.Errorf("stanza: %w", err)
	}
	if image == "" {
		image = "postgres:17"
	}
	if strings.ContainsAny(image, " \t\n") {
		return nil, fmt.Errorf("image %q is not a single name", image)
	}
	return []string{
		"docker", "run", "--rm", "--name", "tuskline-verify-" + stanza,
		image, "pgbackrest", "--stanza", stanza, "restore", "--type", "immediate",
	}, nil
}
