// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"strings"
	"testing"
)

func TestDiagnoseArguments(t *testing.T) {
	path := seedInventory(t)
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"diagnose", "prod/orders", "-o", "yaml"}, "want text, json or markdown"},
		{[]string{"diagnose", "prod/orders", "--lag", "0"}, "greater than 0"},
		{[]string{"diagnose", "orders"}, "env/cluster"},
		{[]string{"diagnose", "prod/missing"}, "not in the inventory"},
	}
	for _, tc := range cases {
		_, _, err := runApp(t, &app{}, append(tc.args, "--inventory", path)...)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%v: got %v, want %q", tc.args, err, tc.want)
		}
	}
}
