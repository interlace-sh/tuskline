// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/interlace-sh/tuskline/core/conn"
	"github.com/interlace-sh/tuskline/core/inventory"
)

// connectRequest is the wizard's save. Password, when set, is used for this
// probe only. It is never written to the inventory or the metadata store,
// and it is never copied into the response.
type connectRequest struct {
	Environment string `json:"environment"`
	Name        string `json:"name"`
	Host        string `json:"host"`
	Port        uint16 `json:"port"`
	Database    string `json:"database"`
	User        string `json:"user"`
	TLS         string `json:"tls"`
	PasswordEnv string `json:"password_env"`
	Password    string `json:"password"`
}

type connectResponse struct {
	clusterView
	Saved bool   `json:"saved"`
	Note  string `json:"note,omitempty"`
}

// saveCluster probes the server the operator named and, only after that
// succeeds, writes the inventory the way `tl connect` does. A typed password
// is not saved. password_env is stored as a reference.
func (a *api) saveCluster(w http.ResponseWriter, r *http.Request) {
	var req connectRequest
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, errors.New("invalid connection"))
		return
	}
	secret := strings.TrimSpace(req.Password)
	req.Password = ""

	ref, profile, pwRef, err := connectTarget(req)
	if err != nil {
		writeErr(w, http.StatusBadRequest, redact(err, secret))
		return
	}
	profile, err = connectPassword(profile, secret, pwRef)
	if err != nil {
		writeErr(w, http.StatusBadRequest, redact(err, secret))
		return
	}
	inv, err := a.inventory()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, redact(err, secret))
		return
	}
	kind, err := connectKind(inv, ref.Env)
	if err != nil {
		writeErr(w, http.StatusBadRequest, redact(err, secret))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), clusterCallTimeout)
	defer cancel()
	info, err := conn.ProbeSession(ctx, profile, inventory.DefaultPolicy(kind).Session())
	if err != nil {
		writeErr(w, http.StatusBadGateway, redact(err, secret))
		return
	}

	view, note, err := a.recordCluster(ref, kind, profile, pwRef, info, secret != "" && pwRef == nil)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, redact(err, secret))
		return
	}
	writeJSON(w, http.StatusOK, connectResponse{clusterView: view, Saved: true, Note: note})
}

func connectTarget(req connectRequest) (inventory.Ref, conn.Profile, *inventory.SecretRef, error) {
	ref := inventory.Ref{Env: strings.TrimSpace(req.Environment), Cluster: strings.TrimSpace(req.Name)}
	if err := inventory.ValidateName(ref.Env); err != nil {
		return inventory.Ref{}, conn.Profile{}, nil, fmt.Errorf("environment: %w", err)
	}
	if err := inventory.ValidateName(ref.Cluster); err != nil {
		return inventory.Ref{}, conn.Profile{}, nil, fmt.Errorf("cluster: %w", err)
	}
	if strings.TrimSpace(req.Host) == "" || strings.TrimSpace(req.Database) == "" || strings.TrimSpace(req.User) == "" {
		return inventory.Ref{}, conn.Profile{}, nil, errors.New("host, database and user are required")
	}
	if req.Port == 0 {
		return inventory.Ref{}, conn.Profile{}, nil, errors.New("port must be 1–65535")
	}
	mode, err := conn.ParseTLSMode(req.TLS)
	if err != nil {
		return inventory.Ref{}, conn.Profile{}, nil, err
	}
	var pwRef *inventory.SecretRef
	if env := strings.TrimSpace(req.PasswordEnv); env != "" {
		pwRef = &inventory.SecretRef{Env: env}
		if err := pwRef.Validate(); err != nil {
			return inventory.Ref{}, conn.Profile{}, nil, err
		}
	}
	profile := conn.Profile{
		Host: strings.TrimSpace(req.Host), Port: req.Port,
		Database: strings.TrimSpace(req.Database), User: strings.TrimSpace(req.User), TLS: mode,
	}
	return ref, profile, pwRef, nil
}

// connectKind uses an existing environment's kind. A new environment takes
// its name as the kind, the same default as `tl connect`.
func connectKind(inv *inventory.Inventory, env string) (inventory.EnvKind, error) {
	if existing, ok := inv.Environment(env); ok {
		return existing.Kind, nil
	}
	kind := inventory.EnvKind(env)
	if !kind.Valid() {
		return "", fmt.Errorf("environment %s: give a kind local, dev, staging or prod", env)
	}
	return kind, nil
}

// connectPassword fills the in-memory password: the one-time value, then the
// env reference, then ~/.pgpass.
func connectPassword(p conn.Profile, once string, ref *inventory.SecretRef) (conn.Profile, error) {
	if once != "" {
		return p.WithPassword(once), nil
	}
	return resolvePassword(p, ref, conn.PassfilePath())
}

func (a *api) recordCluster(ref inventory.Ref, kind inventory.EnvKind, profile conn.Profile, pwRef *inventory.SecretRef, info conn.ServerInfo, onceOnly bool) (clusterView, string, error) {
	path, err := a.inventoryFile()
	if err != nil {
		return clusterView{}, "", err
	}
	inv, err := inventory.Load(path)
	if err != nil {
		return clusterView{}, "", err
	}
	if existing, ok := inv.Environment(ref.Env); ok {
		kind = existing.Kind
	} else if _, err := inv.PutEnvironment(ref.Env, kind); err != nil {
		return clusterView{}, "", err
	}
	cl, _ := inv.Cluster(ref)
	cl.Name, cl.Environment = ref.Cluster, ref.Env
	cl.Profile = profile.WithPassword("")
	cl.Provider = string(info.Provider)
	if pwRef != nil {
		cl.Password = pwRef
	}
	cl.LastProbe = inventory.RecordProbe(info, time.Now())
	if err := inv.PutCluster(cl); err != nil {
		return clusterView{}, "", err
	}
	if err := inv.Save(path); err != nil {
		return clusterView{}, "", err
	}
	note := ""
	if onceOnly {
		note = "The password was used for this probe only and was not saved. Use a password environment variable or ~/.pgpass."
	}
	return clusterView{
		ID: ref.String(), Environment: ref.Env, Name: ref.Cluster, Kind: string(kind),
		Provider: cl.Provider, Host: cl.Profile.Host, Port: cl.Profile.EffectivePort(),
		Database: cl.Profile.Database, User: cl.Profile.User, TLS: string(cl.Profile.TLSMode()),
		LastProbe: cl.LastProbe,
	}, note, nil
}

func redact(err error, secret string) error {
	if err == nil || secret == "" || !strings.Contains(err.Error(), secret) {
		return err
	}
	return errors.New(strings.ReplaceAll(err.Error(), secret, "[redacted]"))
}
