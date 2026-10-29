// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"errors"
	"fmt"
	"os"
	"regexp"

	"go.yaml.in/yaml/v3"
)

// ErrPlaintextSecret is returned when an inventory file holds a secret value
// instead of a reference to one.
var ErrPlaintextSecret = errors.New("plaintext secrets are not allowed in the inventory; use a reference such as {env: PGPASSWORD_ORDERS}")

// ErrSecretUnavailable is returned when a reference cannot be resolved.
var ErrSecretUnavailable = errors.New("secret unavailable")

// ErrKeychainNotImplemented is returned for keychain references until the OS
// keychain provider lands (roadmap M8.3).
var ErrKeychainNotImplemented = errors.New("OS keychain secrets are not implemented yet (roadmap M8.3)")

// SecretRef points at a secret held elsewhere. Exactly one field is set.
// The inventory stores only the reference, never the value.
type SecretRef struct {
	// Env names an environment variable holding the secret.
	Env string `json:"env,omitempty" yaml:"env,omitempty"`
	// Keychain names an OS keychain item (placeholder until M8.3).
	Keychain string `json:"keychain,omitempty" yaml:"keychain,omitempty"`
}

var envNameRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Validate checks that exactly one well-formed reference is set.
func (r SecretRef) Validate() error {
	switch {
	case r.Env != "" && r.Keychain != "":
		return errors.New("secret reference: set env or keychain, not both")
	case r.Env != "":
		if !envNameRE.MatchString(r.Env) {
			return fmt.Errorf("secret reference: %q is not a valid environment variable name", r.Env)
		}
	case r.Keychain == "":
		return errors.New("secret reference: set env or keychain")
	}
	return nil
}

// String describes the reference without resolving it.
func (r SecretRef) String() string {
	if r.Env != "" {
		return "env:" + r.Env
	}
	return "keychain:" + r.Keychain
}

// Resolve returns the secret value. Errors name the reference, never a
// value.
func (r SecretRef) Resolve() (string, error) {
	if err := r.Validate(); err != nil {
		return "", err
	}
	if r.Keychain != "" {
		return "", fmt.Errorf("%s: %w", r, ErrKeychainNotImplemented)
	}
	v, ok := os.LookupEnv(r.Env)
	if !ok || v == "" {
		return "", fmt.Errorf("%s is not set: %w", r, ErrSecretUnavailable)
	}
	return v, nil
}

// UnmarshalYAML accepts only the mapping form, so `password: hunter2` fails
// with ErrPlaintextSecret instead of being stored or echoed in an error.
func (r *SecretRef) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind != yaml.MappingNode {
		return fmt.Errorf("line %d: %w", n.Line, ErrPlaintextSecret)
	}
	type plain SecretRef
	var p plain
	dec := n.Decode
	if err := dec(&p); err != nil {
		return fmt.Errorf("line %d: invalid secret reference", n.Line)
	}
	*r = SecretRef(p)
	return nil
}
