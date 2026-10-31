// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"context"
	"fmt"
)

// Event is one line of the apply log.
type Event struct {
	Step   string `json:"step"`
	Kind   string `json:"kind"`
	Status string `json:"status"` // start, ok, skip, fail
	Text   string `json:"text"`
}

// Journal remembers completed steps so a later apply can resume.
// The stored value is the hash of the step body.
type Journal interface {
	Load(ctx context.Context) (map[string]string, error)
	Append(ctx context.Context, id, hash string) error
}

// FS is where file, unit and manifest steps are written.
type FS interface {
	Read(ctx context.Context, path string) (string, bool, error)
	Write(ctx context.Context, path string, content string) error
	// NonEmpty reports whether path is an existing directory with entries.
	NonEmpty(ctx context.Context, path string) (bool, error)
}

// Executor runs a command step. The engine does not start a shell itself.
type Executor interface {
	Run(ctx context.Context, step Step) error
}

// Apply writes plan in order. A step whose journal hash still matches is
// skipped. A failed step is not recorded, so the next call resumes there.
// log may be nil.
func Apply(ctx context.Context, plan Plan, journal Journal, fs FS, exec Executor, log func(Event)) error {
	if err := plan.Check(); err != nil {
		return err
	}
	done := map[string]string{}
	if journal != nil {
		loaded, err := journal.Load(ctx)
		if err != nil {
			return fmt.Errorf("read apply journal: %w", err)
		}
		done = loaded
	}
	for _, step := range plan.Steps {
		if err := ctx.Err(); err != nil {
			emit(log, Event{Step: step.ID, Kind: step.Kind, Status: "fail", Text: err.Error()})
			return err
		}
		hash := stepHash(step)
		if done[step.ID] == hash {
			emit(log, Event{Step: step.ID, Kind: step.Kind, Status: "skip", Text: "already applied"})
			continue
		}
		emit(log, Event{Step: step.ID, Kind: step.Kind, Status: "start", Text: step.Path})
		if err := applyStep(ctx, step, fs, exec); err != nil {
			emit(log, Event{Step: step.ID, Kind: step.Kind, Status: "fail", Text: err.Error()})
			return fmt.Errorf("step %s: %w", step.ID, err)
		}
		if journal != nil {
			if err := journal.Append(ctx, step.ID, hash); err != nil {
				emit(log, Event{Step: step.ID, Kind: step.Kind, Status: "fail", Text: err.Error()})
				return fmt.Errorf("record step %s: %w", step.ID, err)
			}
		}
		done[step.ID] = hash
		emit(log, Event{Step: step.ID, Kind: step.Kind, Status: "ok", Text: step.Path})
	}
	return nil
}

func applyStep(ctx context.Context, step Step, fs FS, exec Executor) error {
	if step.Secret {
		return fmt.Errorf("secret store is not configured")
	}
	switch step.Kind {
	case KindCommand:
		if exec == nil {
			return fmt.Errorf("no executor for a command step")
		}
		return exec.Run(ctx, step)
	case KindFile, KindUnit, KindManifest:
		if fs == nil {
			return fmt.Errorf("no filesystem")
		}
		dest, err := cleanPath(step.Path)
		if err != nil {
			return err
		}
		if step.DataDir {
			nonEmpty, err := fs.NonEmpty(ctx, dest)
			if err != nil {
				return err
			}
			if nonEmpty {
				return fmt.Errorf("refuse non-empty data directory %s", dest)
			}
		}
		return fs.Write(ctx, dest, step.After)
	default:
		return fmt.Errorf("unknown kind %q", step.Kind)
	}
}

func emit(log func(Event), ev Event) {
	if log != nil {
		log(ev)
	}
}
