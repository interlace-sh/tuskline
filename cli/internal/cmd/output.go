// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// Output formats for --output.
const (
	outputText = "text"
	outputJSON = "json"
)

// addOutputFlag registers --output text|json on c.
func addOutputFlag(c *cobra.Command, dst *string) {
	c.Flags().StringVarP(dst, "output", "o", outputText, "output format: text or json")
}

func checkOutput(format string) error {
	if format != outputText && format != outputJSON {
		return fmt.Errorf("--output %q: want text or json", format)
	}
	return nil
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
