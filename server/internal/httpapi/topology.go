// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/interlace-sh/tuskline/core/ha"
	"github.com/interlace-sh/tuskline/core/inventory"
	"github.com/interlace-sh/tuskline/core/store"
)

// clusterTopology reads the Patroni or CloudNativePG document registered
// on the cluster. It does not connect to Postgres and it does not change
// the topology.
func (a *api) clusterTopology(w http.ResponseWriter, r *http.Request) {
	cl, _, err := a.savedCluster(r)
	if err != nil {
		writeMissing(w, err)
		return
	}
	if cl.Patroni == nil || strings.TrimSpace(cl.Patroni.URL) == "" {
		writeErr(w, http.StatusNotFound, errors.New("this cluster has no Patroni URL"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	body, err := getLimited(ctx, strings.TrimRight(cl.Patroni.URL, "/")+"/cluster")
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	topo, err := parseTopology(cl.Patroni.URL, body)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, topo)
}

type switchRequest struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Confirm string `json:"confirm"`
	Run     bool   `json:"run"`
}

// clusterSwitchover previews a leader move. run sends it after the
// environment's confirmation rule, and records an audit row.
func (a *api) clusterSwitchover(w http.ResponseWriter, r *http.Request) {
	var req switchRequest
	r.Body = http.MaxBytesReader(w, r.Body, 4<<10)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, errors.New("invalid switchover"))
		return
	}
	cl, env, err := a.savedCluster(r)
	if err != nil {
		writeMissing(w, err)
		return
	}
	if cl.Patroni == nil || cl.Patroni.URL == "" {
		writeErr(w, http.StatusNotFound, errors.New("this cluster has no Patroni URL"))
		return
	}
	action := ha.Switchover{
		Environment: cl.Environment,
		Cluster:     cl.Name,
		From:        req.From,
		To:          req.To,
		API:         cl.Patroni.URL,
		Prod:        env.Policy.RequireTypedConfirmation,
		Confirm:     req.Confirm,
	}
	if !req.Run {
		writeJSON(w, http.StatusOK, map[string]any{"preview": action.Preview(), "sent": false})
		return
	}
	if err := action.Check(); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	sendErr := postLimited(ctx, strings.TrimRight(action.API, "/")+"/switchover", action.Body())
	failure := ""
	if sendErr != nil {
		failure = sendErr.Error()
	}
	if err := a.recordAction(ctx, cl.Environment, cl.Name, "switchover", action.Preview(), failure); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if sendErr != nil {
		writeErr(w, http.StatusBadGateway, sendErr)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"preview": action.Preview(), "sent": true})
}

func (a *api) savedCluster(r *http.Request) (inventory.Cluster, inventory.Environment, error) {
	ref, err := inventory.ParseRef(r.PathValue("env") + "/" + r.PathValue("name"))
	if err != nil {
		return inventory.Cluster{}, inventory.Environment{}, err
	}
	inv, err := a.inventory()
	if err != nil {
		return inventory.Cluster{}, inventory.Environment{}, err
	}
	cl, ok := inv.Cluster(ref)
	if !ok {
		return inventory.Cluster{}, inventory.Environment{}, fmt.Errorf("%s: %w", ref, errNotInInventory)
	}
	env, ok := inv.Environment(ref.Env)
	if !ok {
		return inventory.Cluster{}, inventory.Environment{}, fmt.Errorf("%s: environment is missing", ref)
	}
	return cl, env, nil
}

func writeMissing(w http.ResponseWriter, err error) {
	if errors.Is(err, errNotInInventory) {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	writeErr(w, http.StatusBadRequest, err)
}

func parseTopology(rawURL string, body []byte) (ha.Topology, error) {
	if strings.Contains(rawURL, "cnpg") || strings.Contains(string(body), `"instances"`) {
		return ha.ParseCNPG(body)
	}
	return ha.ParsePatroni(body)
}

func getLimited(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	closeErr := res.Body.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("GET %s: %s", rawURL, res.Status)
	}
	return body, nil
}

func postLimited(ctx context.Context, rawURL, body string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(io.Discard, io.LimitReader(res.Body, 1<<20))
	closeErr := res.Body.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if res.StatusCode >= 300 {
		return fmt.Errorf("POST %s: %s", rawURL, res.Status)
	}
	return nil
}

func (a *api) recordAction(ctx context.Context, env, cluster, action, preview, failure string) error {
	db, err := a.openAudit(ctx)
	if err != nil {
		return err
	}
	recordErr := db.RecordAction(ctx, store.ActionAudit{
		Actor: localActor, Environment: env, Cluster: cluster, Action: action, Preview: preview, Error: failure,
	})
	if err := db.Close(); err != nil && recordErr == nil {
		return err
	}
	return recordErr
}
