// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"errors"
	"fmt"

	"github.com/interlace-sh/tuskline/core/conn"
	"github.com/interlace-sh/tuskline/core/inventory"
)

// resolvePassword fills p from the secret reference, then from the password
// file when that reference is unset. A reference that fails for any other
// reason (for example an unimplemented keychain) is returned as-is.
func resolvePassword(p conn.Profile, secret *inventory.SecretRef, passfile string) (conn.Profile, error) {
	if p.HasPassword() {
		return p, nil
	}
	var unavailable error
	if secret != nil {
		pw, err := secret.Resolve()
		if err == nil {
			return p.WithPassword(pw), nil
		}
		if !errors.Is(err, inventory.ErrSecretUnavailable) {
			return p, err
		}
		unavailable = err
	}
	pw, err := conn.LookupPassword(passfile, p)
	if err != nil {
		return p, err
	}
	if pw != "" {
		return p.WithPassword(pw), nil
	}
	if unavailable != nil {
		return p, unavailable
	}
	return p, fmt.Errorf("no password for %s@%s", p.User, p.Host)
}
