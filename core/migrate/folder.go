// SPDX-License-Identifier: Apache-2.0

package migrate

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/interlace-sh/tuskline/core/inventory"
	"github.com/interlace-sh/tuskline/core/lint"
)

// ErrApproval is returned when a production apply has no approval.
var ErrApproval = errors.New("prod requires approval")

// File is one migration script.
type File struct {
	Name     string
	SQL      string
	Checksum string // hex-encoded SHA-256 of SQL
}

// Folder is an ordered set of migration scripts. Nothing has been applied.
type Folder struct {
	Dir   string
	Files []File
}

const maxScript = 1 << 20

// Load reads the .sql files in dir. Names are ordered so that a number
// inside a name sorts by its value: 2_two.sql comes before 10_ten.sql.
// Files in subdirectories are not read.
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
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(strings.ToLower(name), ".sql") {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return Folder{}, fmt.Errorf("%s has no migration scripts", dir)
	}
	sortNames(names)
	files, err := readAll(abs, names)
	if err != nil {
		return Folder{}, err
	}
	return Folder{Dir: abs, Files: files}, nil
}

// Lint runs the lock rules on every script. Table sizes are not collected,
// so SET NOT NULL treats every table as unknown. The report is empty when
// there is nothing to say.
func (f Folder) Lint() (string, error) {
	var b strings.Builder
	for _, file := range f.Files {
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

// List renders each script name and the first 12 characters of its checksum.
func (f Folder) List() string {
	var b strings.Builder
	for _, file := range f.Files {
		sum := file.Checksum
		if len(sum) > 12 {
			sum = sum[:12]
		}
		fmt.Fprintf(&b, "%s  %s\n", file.Name, sum)
	}
	return b.String()
}

// Preview describes the folder in apply order. It does not apply anything.
func (f Folder) Preview() string {
	var b strings.Builder
	if len(f.Files) == 0 {
		b.WriteString("no migration scripts\n")
	}
	b.WriteString(f.List())
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

func readAll(dir string, names []string) ([]File, error) {
	files := make([]File, len(names))
	for i, name := range names {
		if strings.Contains(name, "..") || filepath.IsAbs(name) || strings.ContainsRune(name, filepath.Separator) {
			return nil, fmt.Errorf("migration name %q is not a file name", name)
		}
		sql, err := readScript(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		files[i] = File{Name: name, SQL: sql, Checksum: checksum(sql)}
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

func checksum(sql string) string {
	sum := sha256.Sum256([]byte(sql))
	return hex.EncodeToString(sum[:])
}

func sortNames(names []string) {
	// Insertion sort keeps the implementation obvious for a short list.
	for i := 1; i < len(names); i++ {
		name := names[i]
		j := i
		for j > 0 && compareNatural(names[j-1], name) > 0 {
			names[j] = names[j-1]
			j--
		}
		names[j] = name
	}
}

// compareNatural compares ASCII names, with each digit run compared as a
// number. "2_b.sql" sorts before "10_a.sql".
func compareNatural(a, b string) int {
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if isDigit(a[i]) && isDigit(b[j]) {
			i0, j0 := i, j
			for i < len(a) && isDigit(a[i]) {
				i++
			}
			for j < len(b) && isDigit(b[j]) {
				j++
			}
			if c := compareDigits(a[i0:i], b[j0:j]); c != 0 {
				return c
			}
			continue
		}
		if a[i] != b[j] {
			if a[i] < b[j] {
				return -1
			}
			return 1
		}
		i++
		j++
	}
	switch {
	case i == len(a) && j == len(b):
		return 0
	case i == len(a):
		return -1
	default:
		return 1
	}
}

func compareDigits(a, b string) int {
	a = strings.TrimLeft(a, "0")
	b = strings.TrimLeft(b, "0")
	if a == "" {
		a = "0"
	}
	if b == "" {
		b = "0"
	}
	if len(a) != len(b) {
		if len(a) < len(b) {
			return -1
		}
		return 1
	}
	return strings.Compare(a, b)
}

func isDigit(b byte) bool { return b >= '0' && b <= '9' }
