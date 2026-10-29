// SPDX-License-Identifier: Apache-2.0

package store

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/parquet-go/parquet-go"

	"github.com/interlace-sh/tuskline/core/collector"
)

// Resolutions of a metrics segment. A 1s file holds one minute, a 1m file
// one hour, and a 1h file one day (ADR 0013).
const (
	Res1s = "1s"
	Res1m = "1m"
	Res1h = "1h"
)

// Kind says how a sample rolls up.
type Kind string

const (
	KindGauge   Kind = "gauge"
	KindCounter Kind = "counter"
)

// Sample is one metric observation. For a gauge, Sum/Count is the average
// and Min/Max bound it. For a counter at 1s resolution, Sum is the absolute
// counter; after a rollup it is the sum of non-negative deltas.
type Sample struct {
	At    time.Time
	Name  string
	Kind  Kind
	Sum   float64
	Min   float64
	Max   float64
	Count int64
}

// Gauge builds a raw gauge sample.
func Gauge(at time.Time, name string, value float64) Sample {
	return Sample{At: at.UTC(), Name: name, Kind: KindGauge, Sum: value, Min: value, Max: value, Count: 1}
}

// Counter builds a raw counter sample. value is the absolute reading.
func Counter(at time.Time, name string, value float64) Sample {
	return Sample{At: at.UTC(), Name: name, Kind: KindCounter, Sum: value, Min: value, Max: value, Count: 1}
}

// Average is Sum/Count, or 0 when nothing was counted.
func (s Sample) Average() float64 {
	if s.Count == 0 {
		return 0
	}
	return s.Sum / float64(s.Count)
}

type point struct {
	At    int64   `parquet:"at"`
	Name  string  `parquet:"name,dict"`
	Kind  string  `parquet:"kind,dict"`
	Sum   float64 `parquet:"sum"`
	Min   float64 `parquet:"min"`
	Max   float64 `parquet:"max"`
	Count int64   `parquet:"count"`
}

func (s Sample) point() point {
	return point{At: s.At.UTC().UnixMicro(), Name: s.Name, Kind: string(s.Kind), Sum: s.Sum, Min: s.Min, Max: s.Max, Count: s.Count}
}

func (p point) sample() Sample {
	return Sample{At: time.UnixMicro(p.At).UTC(), Name: p.Name, Kind: Kind(p.Kind), Sum: p.Sum, Min: p.Min, Max: p.Max, Count: p.Count}
}

// Segments is a directory of Parquet metric files. Now, when set, is the
// clock used to decide which bucket is still open.
type Segments struct {
	Root string
	Now  func() time.Time
}

func (s Segments) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

// TargetKey turns env/cluster into one path segment.
func TargetKey(ref string) (string, error) {
	env, cluster, ok := strings.Cut(ref, "/")
	if !ok || env == "" || cluster == "" || strings.Contains(cluster, "/") || strings.Contains(env, "..") || strings.Contains(cluster, "..") {
		return "", fmt.Errorf("store: target %q is not env/cluster", ref)
	}
	return env + "__" + cluster, nil
}

// Write stores raw samples in the open 1s bucket. Samples from a closed
// minute are rejected, so a finished file is never rewritten.
func (s Segments) Write(ref string, samples []Sample) error {
	if len(samples) == 0 {
		return nil
	}
	key, err := TargetKey(ref)
	if err != nil {
		return err
	}
	open := bucketStart(Res1s, s.now())
	var batch []Sample
	for _, sample := range samples {
		if sample.Name == "" || sample.Count < 1 || (sample.Kind != KindGauge && sample.Kind != KindCounter) {
			return fmt.Errorf("store: bad sample %q", sample.Name)
		}
		if !bucketStart(Res1s, sample.At).Equal(open) {
			return fmt.Errorf("store: sample %s is outside the open minute %s", sample.At.Format(time.RFC3339), open.Format(time.RFC3339))
		}
		batch = append(batch, sample)
	}
	return s.merge(key, Res1s, open, batch)
}

// Read returns samples in [from, to) for one resolution.
func (s Segments) Read(ref string, res string, from, to time.Time) ([]Sample, error) {
	key, err := TargetKey(ref)
	if err != nil {
		return nil, err
	}
	files, err := s.files(key, res)
	if err != nil {
		return nil, err
	}
	var out []Sample
	for _, f := range files {
		if !f.start.Before(to) || !bucketEnd(res, f.start).After(from) {
			continue
		}
		samples, err := readFile(f.path)
		if err != nil {
			return nil, err
		}
		for _, sample := range samples {
			if !sample.At.Before(from) && sample.At.Before(to) {
				out = append(out, sample)
			}
		}
	}
	slices.SortFunc(out, compareSamples)
	return out, nil
}

// Retain rolls closed buckets that are past their retention into the next
// resolution, then deletes them. 1s is kept for 1 hour, 1m for 14 days and
// 1h for 13 calendar months.
func (s Segments) Retain(ref string, now time.Time) error {
	key, err := TargetKey(ref)
	if err != nil {
		return err
	}
	now = now.UTC()
	if err := s.compact(key, Res1s, Res1m, now.Add(-time.Hour), now); err != nil {
		return err
	}
	if err := s.compact(key, Res1m, Res1h, now.Add(-14*24*time.Hour), now); err != nil {
		return err
	}
	return s.drop(key, Res1h, now.AddDate(0, -13, 0), now)
}

func (s Segments) compact(key, from, to string, cutoff, now time.Time) error {
	files, err := s.files(key, from)
	if err != nil {
		return err
	}
	open := bucketStart(from, now)
	for _, f := range files {
		if !f.start.Before(open) || !f.start.Before(cutoff) {
			continue
		}
		samples, err := readFile(f.path)
		if err != nil {
			return err
		}
		rolled := rollup(samples, from)
		if err := s.merge(key, to, bucketStart(to, f.start), rolled); err != nil {
			return err
		}
		if err := os.Remove(f.path); err != nil {
			return fmt.Errorf("store: remove %s: %w", f.path, err)
		}
	}
	return nil
}

func (s Segments) drop(key, res string, cutoff, now time.Time) error {
	files, err := s.files(key, res)
	if err != nil {
		return err
	}
	open := bucketStart(res, now)
	for _, f := range files {
		if f.start.Before(open) && f.start.Before(cutoff) {
			if err := os.Remove(f.path); err != nil {
				return fmt.Errorf("store: remove %s: %w", f.path, err)
			}
		}
	}
	return nil
}

func (s Segments) merge(key, res string, start time.Time, samples []Sample) error {
	if len(samples) == 0 {
		return nil
	}
	path := segmentPath(s.Root, key, res, start)
	existing, err := readFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	combined := mergeSamples(existing, samples)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("store: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*")
	if err != nil {
		return fmt.Errorf("store: %w", err)
	}
	tmpName := tmp.Name()
	ok := false
	defer func() {
		if !ok {
			_ = os.Remove(tmpName)
		}
	}()
	if err := writePoints(tmp, combined); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("store: rename %s: %w", path, err)
	}
	ok = true
	return nil
}

func writePoints(w io.Writer, samples []Sample) error {
	points := make([]point, len(samples))
	for i, sample := range samples {
		points[i] = sample.point()
	}
	pw := parquet.NewGenericWriter[point](w)
	if _, err := pw.Write(points); err != nil {
		return fmt.Errorf("store: write segment: %w", err)
	}
	if err := pw.Close(); err != nil {
		return fmt.Errorf("store: close segment: %w", err)
	}
	return nil
}

func readFile(path string) ([]Sample, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	r := parquet.NewGenericReader[point](f)
	defer func() { _ = r.Close() }()
	var out []Sample
	buf := make([]point, 64)
	for {
		n, err := r.Read(buf)
		for _, p := range buf[:n] {
			out = append(out, p.sample())
		}
		if errors.Is(err, io.EOF) {
			return out, nil
		}
		if err != nil {
			return nil, fmt.Errorf("store: read %s: %w", path, err)
		}
	}
}

type segmentFile struct {
	path  string
	start time.Time
}

func (s Segments) files(key, res string) ([]segmentFile, error) {
	dir := filepath.Join(s.Root, key, res)
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: %w", err)
	}
	var out []segmentFile
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".parquet") {
			continue
		}
		start, err := time.Parse("20060102T150405Z", strings.TrimSuffix(name, ".parquet"))
		if err != nil {
			continue
		}
		out = append(out, segmentFile{path: filepath.Join(dir, name), start: start.UTC()})
	}
	return out, nil
}

func segmentPath(root, key, res string, start time.Time) string {
	name := bucketStart(res, start).UTC().Format("20060102T150405Z") + ".parquet"
	return filepath.Join(root, key, res, name)
}

func bucketStart(res string, at time.Time) time.Time {
	at = at.UTC()
	switch res {
	case Res1m:
		return at.Truncate(time.Hour)
	case Res1h:
		return time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, time.UTC)
	default:
		return at.Truncate(time.Minute)
	}
}

func bucketEnd(res string, start time.Time) time.Time {
	switch res {
	case Res1m:
		return start.Add(time.Hour)
	case Res1h:
		return start.AddDate(0, 0, 1)
	default:
		return start.Add(time.Minute)
	}
}

func mergeSamples(existing, extra []Sample) []Sample {
	type key struct {
		name string
		at   int64
	}
	m := make(map[key]Sample, len(existing)+len(extra))
	for _, sample := range existing {
		m[key{sample.Name, sample.At.UnixMicro()}] = sample
	}
	for _, sample := range extra {
		m[key{sample.Name, sample.At.UTC().UnixMicro()}] = sample
	}
	out := make([]Sample, 0, len(m))
	for _, sample := range m {
		out = append(out, sample)
	}
	slices.SortFunc(out, compareSamples)
	return out
}

func compareSamples(a, b Sample) int {
	if a.Name != b.Name {
		if a.Name < b.Name {
			return -1
		}
		return 1
	}
	return a.At.Compare(b.At)
}

// rollup collapses samples from one finer bucket into one sample per metric.
// Counters at 1s resolution are absolute readings, so the rollup is the sum
// of non-negative deltas. Coarser counters are already deltas and are summed.
func rollup(samples []Sample, from string) []Sample {
	slices.SortFunc(samples, compareSamples)
	type acc struct {
		kind          Kind
		sum, min, max float64
		count         int64
		prev          float64
		seen          bool
		counter       bool
	}
	byName := map[string]*acc{}
	var order []string
	for _, sample := range samples {
		a := byName[sample.Name]
		if a == nil {
			a = &acc{kind: sample.Kind, min: sample.Min, max: sample.Max}
			byName[sample.Name] = a
			order = append(order, sample.Name)
		}
		if sample.Kind == KindCounter && from == Res1s {
			a.counter = true
			if a.seen {
				delta, _ := collector.DeltaFloat(a.prev, sample.Sum)
				a.sum += delta
				a.count++
			}
			a.prev = sample.Sum
			a.seen = true
			continue
		}
		if sample.Kind == KindCounter {
			a.counter = true
			a.sum += sample.Sum
			a.count += sample.Count
			continue
		}
		a.sum += sample.Sum
		a.count += sample.Count
		if sample.Min < a.min {
			a.min = sample.Min
		}
		if sample.Max > a.max {
			a.max = sample.Max
		}
	}
	var at time.Time
	if len(samples) > 0 {
		at = bucketStart(from, samples[0].At)
	}
	out := make([]Sample, 0, len(order))
	for _, name := range order {
		a := byName[name]
		sample := Sample{At: at, Name: name, Kind: a.kind, Sum: a.sum, Count: a.count}
		if a.counter {
			sample.Kind = KindCounter
		} else {
			sample.Min, sample.Max = a.min, a.max
		}
		out = append(out, sample)
	}
	return out
}
