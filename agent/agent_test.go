// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"errors"
	"testing"
)

func TestRunRequiresServerURL(t *testing.T) {
	if err := Run(context.Background(), Config{}); err == nil {
		t.Fatal("want error without server URL")
	}
	if err := Run(context.Background(), Config{ServerURL: "https://example.invalid"}); !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("got %v, want ErrNotImplemented", err)
	}
}
