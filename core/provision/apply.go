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

// SecretStore writes a secret step. The engine does not log the body.
// A nil store refuses every secret step.
type SecretStore interface {
	WriteSecret(ctx context.Context, path, content string) error
}

// Apply writes plan in order. A step whose journal hash still matches is
// skipped, unless the step is marked Rerun. A failed step is not recorded,
// so the next call resumes there. log may be nil. secrets may be nil, in
// which case a secret step fails without writing.
func Apply(ctx context.Context, plan Plan, journal Journal, fs FS, exec Executor, log func(Event), secrets SecretStore) error {
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
		if !step.Rerun && done[step.ID] == hash {
			emit(log, Event{Step: step.ID, Kind: step.Kind, Status: "skip", Text: "already applied"})
			continue
		}
		emit(log, Event{Step: step.ID, Kind: step.Kind, Status: "start", Text: step.Path})
		if err := applyStep(ctx, step, fs, exec, secrets); err != nil {
			emit(log, Event{Step: step.ID, Kind: step.Kind, Status: "fail", Text: err.Error()})
			return fmt.Errorf("step %s: %w", step.ID, err)
		}
		if journal != nil && !step.Rerun {
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

func applyStep(ctx context.Context, step Step, fs FS, exec Executor, secrets SecretStore) error {
	if step.Secret {
		if secrets == nil {
			return fmt.Errorf("secret store is not configured")
		}
		dest, err := cleanPath(step.Path)
		if err != nil {
			return err
		}
		return secrets.WriteSecret(ctx, dest, step.After)
	}
	switch step.Kind {
	case KindCommand:
		if exec == nil {
			return fmt.Errorf("no executor for a command step")
		}
		if len(step.Argv) == 0 {
			return fmt.Errorf("command step has no argv")
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
