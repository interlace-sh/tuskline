// SPDX-License-Identifier: Apache-2.0

package analyser

// Severity is how urgent a finding is.
type Severity string

const (
	SeverityInfo Severity = "info"
	SeverityWarn Severity = "warn"
)

// Finding is one health-check result.
type Finding struct {
	Check    string   `json:"check"`
	Severity Severity `json:"severity"`
	Summary  string   `json:"summary"`
	Evidence string   `json:"evidence"`
	Fix      string   `json:"fix"`
}

// Index is enough of an index to spot unused and duplicate ones.
type Index struct {
	Schema string
	Table  string
	Name   string
	// Key is the indexed columns, in order, so two indexes with the same
	// key on the same table are duplicates. Predicate distinguishes partial
	// indexes.
	Key       string
	Predicate string
	Scans     int64
	Bytes     int64
}
