// SPDX-License-Identifier: Apache-2.0

package migrate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/interlace-sh/tuskline/core/inventory"
	"github.com/interlace-sh/tuskline/core/lint"
)

// ErrApproval is returned when a production apply has no approval.
var ErrApproval = errors.New("prod requires approval")

// Kind is how a migration folder is organised.
type Kind string

const (
	// KindPlain is a directory of .sql files, applied in name order.
	KindPlain Kind = "plain"
	// KindFlyway is a Flyway directory (V* and R* scripts).
	KindFlyway Kind = "flyway"
	// KindSqitch is a project with a sqitch.plan.
	KindSqitch Kind = "sqitch"
)

// File is one migration script. SQL is empty when the file could not be read.
type File struct {
	Name string
	SQL  string
	Err  error
}

// Folder is an ordered set of migration scripts. Nothing has been applied.
type Folder struct {
	Kind  Kind
	Dir   string
	Files []File
}

const maxScript = 1 << 20

var (
	flywayVersioned = regexp.MustCompile(`^V(\d+(?:\.\d+)*)__.+\.sql$`)
	flywayRepeat    = regexp.MustCompile(`^R__.+\.sql$`)
)

// Load reads dir. A sqitch.plan makes it Sqitch. Versioned V* or R* scripts,
// or a flyway.conf, make it Flyway. Otherwise every .sql file is plain SQL.
func Load(dir string) (Folder, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return Folder{}, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return Folder{}, err
	}
	if !info.IsDir() {
		return Folder{}, fmt.Errorf("%s is not a directory", dir)
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return Folder{}, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		names = append(names, e.Name())
	}
	switch {
	case has(names, "sqitch.plan"):
		return loadSqitch(abs)
	case has(names, "flyway.conf") || anyFlyway(names):
		return loadFlyway(abs, names)
	default:
		return loadPlain(abs, names)
	}
}

// Lint runs the lock rules on every script. Table sizes are not collected,
// so SET NOT NULL treats every table as unknown. The report is empty when
// there is nothing to say.
func (f Folder) Lint() (string, error) {
	var b strings.Builder
	for _, file := range f.Files {
		if file.Err != nil {
			fmt.Fprintf(&b, "%s: %s\n", file.Name, file.Err)
			continue
		}
		if strings.TrimSpace(file.SQL) == "" {
			continue
		}
		found, err := lint.Check(file.SQL, lint.Stats{})
		if err != nil {
			return "", fmt.Errorf("%s: %w", file.Name, err)
		}
		if len(found) == 0 {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "%s\n%s", file.Name, lint.Report(found))
	}
	return b.String(), nil
}

// Preview describes the folder in apply order. It does not apply anything.
func (f Folder) Preview() string {
	var b strings.Builder
	fmt.Fprintf(&b, "kind: %s\n", f.Kind)
	if len(f.Files) == 0 {
		b.WriteString("no migration scripts\n")
	}
	for _, file := range f.Files {
		fmt.Fprintf(&b, "%s\n", file.Name)
	}
	b.WriteString("Nothing has been applied.\n")
	return b.String()
}

// Apply checks the production approval and returns the preview.
// env is an inventory environment kind. Production needs approve.
// The returned text is empty when approval is missing.
func (f Folder) Apply(env string, approve bool) (string, error) {
	kind := inventory.EnvKind(env)
	if !kind.Valid() {
		return "", fmt.Errorf("unknown environment %q", env)
	}
	if kind == inventory.EnvProd && !approve {
		return "", ErrApproval
	}
	return f.Preview(), nil
}

func loadPlain(dir string, names []string) (Folder, error) {
	var sql []string
	for _, name := range names {
		if strings.HasSuffix(strings.ToLower(name), ".sql") {
			sql = append(sql, name)
		}
	}
	sort.Strings(sql)
	if len(sql) == 0 {
		return Folder{}, fmt.Errorf("%s has no migration scripts", dir)
	}
	files, err := readAll(dir, sql)
	if err != nil {
		return Folder{}, err
	}
	return Folder{Kind: KindPlain, Dir: dir, Files: files}, nil
}

func loadFlyway(dir string, names []string) (Folder, error) {
	type item struct {
		name    string
		version string
		repeat  bool
	}
	var items []item
	for _, name := range names {
		if m := flywayVersioned.FindStringSubmatch(name); m != nil {
			items = append(items, item{name: name, version: m[1]})
			continue
		}
		if flywayRepeat.MatchString(name) {
			items = append(items, item{name: name, repeat: true})
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].repeat != items[j].repeat {
			return !items[i].repeat
		}
		if items[i].repeat {
			return items[i].name < items[j].name
		}
		if cmp := compareVersion(items[i].version, items[j].version); cmp != 0 {
			return cmp < 0
		}
		return items[i].name < items[j].name
	})
	if len(items) == 0 {
		return Folder{}, fmt.Errorf("%s has flyway config but no V or R scripts", dir)
	}
	ordered := make([]string, len(items))
	for i, item := range items {
		ordered[i] = item.name
	}
	files, err := readAll(dir, ordered)
	if err != nil {
		return Folder{}, err
	}
	return Folder{Kind: KindFlyway, Dir: dir, Files: files}, nil
}

func loadSqitch(dir string) (Folder, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "sqitch.plan"))
	if err != nil {
		return Folder{}, err
	}
	var names []string
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "%") {
			continue
		}
		name := strings.Fields(line)[0]
		if strings.Contains(name, "..") || filepath.IsAbs(name) {
			return Folder{}, fmt.Errorf("sqitch change %q is not a relative name", name)
		}
		names = append(names, name)
	}
	if len(names) == 0 {
		return Folder{}, fmt.Errorf("%s has a sqitch plan with no changes", dir)
	}
	files := make([]File, len(names))
	for i, name := range names {
		rel := filepath.Join("deploy", name+".sql")
		sql, err := readScript(filepath.Join(dir, rel))
		files[i] = File{Name: rel, SQL: sql, Err: err}
	}
	return Folder{Kind: KindSqitch, Dir: dir, Files: files}, nil
}

func readAll(dir string, names []string) ([]File, error) {
	files := make([]File, len(names))
	for i, name := range names {
		if strings.Contains(name, "..") || filepath.IsAbs(name) {
			return nil, fmt.Errorf("migration name %q is not a relative name", name)
		}
		sql, err := readScript(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		files[i] = File{Name: name, SQL: sql}
	}
	return files, nil
}

func readScript(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.Size() > maxScript {
		return "", fmt.Errorf("%s is larger than %d bytes", filepath.Base(path), maxScript)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func has(names []string, want string) bool {
	for _, name := range names {
		if name == want {
			return true
		}
	}
	return false
}

func anyFlyway(names []string) bool {
	for _, name := range names {
		if flywayVersioned.MatchString(name) || flywayRepeat.MatchString(name) {
			return true
		}
	}
	return false
}

func compareVersion(a, b string) int {
	ap := strings.Split(a, ".")
	bp := strings.Split(b, ".")
	n := len(ap)
	if len(bp) > n {
		n = len(bp)
	}
	for i := 0; i < n; i++ {
		var av, bv int
		if i < len(ap) {
			av, _ = strconv.Atoi(ap[i])
		}
		if i < len(bp) {
			bv, _ = strconv.Atoi(bp[i])
		}
		if av != bv {
			return av - bv
		}
	}
	return 0
}
