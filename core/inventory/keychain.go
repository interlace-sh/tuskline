// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/zalando/go-keyring"
)

// KeychainService is the OS keychain service name for Tuskline secrets.
const KeychainService = "tuskline"

// ErrKeychainNotFound is returned when the named item is not in the keychain.
var ErrKeychainNotFound = errors.New("keychain item not found")

// Keyring is the OS keychain. Tests replace it with UseKeyring.
type Keyring interface {
	Get(service, user string) (string, error)
	Set(service, user, password string) error
	Delete(service, user string) error
}

type osKeyring struct{}

func (osKeyring) Get(service, user string) (string, error) {
	return keyring.Get(service, user)
}

func (osKeyring) Set(service, user, password string) error {
	return keyring.Set(service, user, password)
}

func (osKeyring) Delete(service, user string) error {
	return keyring.Delete(service, user)
}

var (
	keyringMu sync.Mutex
	keyrings  Keyring = osKeyring{}
)

// UseKeyring replaces the keychain backend. A nil keyring restores the OS one.
// Tests use this so they do not touch the session keychain.
func UseKeyring(k Keyring) {
	keyringMu.Lock()
	defer keyringMu.Unlock()
	if k == nil {
		keyrings = osKeyring{}
		return
	}
	keyrings = k
}

func currentKeyring() Keyring {
	keyringMu.Lock()
	defer keyringMu.Unlock()
	return keyrings
}

// KeychainItem is the item name stored in the inventory: tuskline/<env>/<cluster>.
func KeychainItem(env, cluster string) (string, error) {
	if err := ValidateName(env); err != nil {
		return "", fmt.Errorf("keychain environment: %w", err)
	}
	if err := ValidateName(cluster); err != nil {
		return "", fmt.Errorf("keychain cluster: %w", err)
	}
	return "tuskline/" + env + "/" + cluster, nil
}

func validKeychainItem(item string) bool {
	rest, ok := strings.CutPrefix(item, "tuskline/")
	if !ok {
		return false
	}
	env, cluster, ok := strings.Cut(rest, "/")
	if !ok || strings.Contains(cluster, "/") {
		return false
	}
	return ValidateName(env) == nil && ValidateName(cluster) == nil
}

// StorePassword saves secret in the OS keychain and returns the inventory reference.
func StorePassword(env, cluster, secret string) (SecretRef, error) {
	if secret == "" {
		return SecretRef{}, errors.New("keychain: empty password")
	}
	item, err := KeychainItem(env, cluster)
	if err != nil {
		return SecretRef{}, err
	}
	if err := currentKeyring().Set(KeychainService, item, secret); err != nil {
		return SecretRef{}, fmt.Errorf("save password in the OS keychain: %w", err)
	}
	return SecretRef{Keychain: item}, nil
}

// DeletePassword removes a Tuskline keychain item. An unknown item is not an error.
func DeletePassword(item string) error {
	if !validKeychainItem(item) {
		return fmt.Errorf("keychain: %q is not a tuskline/<env>/<cluster> item", item)
	}
	err := currentKeyring().Delete(KeychainService, item)
	if err == nil || errors.Is(err, keyring.ErrNotFound) || errors.Is(err, ErrKeychainNotFound) {
		return nil
	}
	return fmt.Errorf("delete keychain item: %w", err)
}

func keychainGet(item string) (string, error) {
	if !validKeychainItem(item) {
		return "", fmt.Errorf("keychain: %q is not a tuskline/<env>/<cluster> item", item)
	}
	v, err := currentKeyring().Get(KeychainService, item)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) || errors.Is(err, ErrKeychainNotFound) {
			return "", fmt.Errorf("keychain:%s: %w", item, ErrKeychainNotFound)
		}
		return "", fmt.Errorf("keychain:%s: %w", item, err)
	}
	if v == "" {
		return "", fmt.Errorf("keychain:%s: %w", item, ErrKeychainNotFound)
	}
	return v, nil
}

// MemoryKeyring is an in-memory keychain for tests.
type MemoryKeyring struct {
	mu    sync.Mutex
	items map[string]string
}

// NewMemoryKeyring returns an empty in-memory keychain.
func NewMemoryKeyring() *MemoryKeyring {
	return &MemoryKeyring{items: map[string]string{}}
}

func (m *MemoryKeyring) key(service, user string) string { return service + "\x00" + user }

// Get returns a stored secret.
func (m *MemoryKeyring) Get(service, user string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.items[m.key(service, user)]
	if !ok {
		return "", ErrKeychainNotFound
	}
	return v, nil
}

// Set stores a secret.
func (m *MemoryKeyring) Set(service, user, password string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.items == nil {
		m.items = map[string]string{}
	}
	m.items[m.key(service, user)] = password
	return nil
}

// Delete removes a secret.
func (m *MemoryKeyring) Delete(service, user string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items, m.key(service, user))
	return nil
}
