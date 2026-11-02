// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/interlace-sh/tuskline/core/ha"
	"github.com/interlace-sh/tuskline/core/store"
)

func newHACmd(a *app) *cobra.Command {
	cmd := &cobra.Command{Use: "ha", Short: "Read topology and preview guarded actions"}
	cmd.AddCommand(newHAStatusCmd(), newHASwitchoverCmd(a), newHARestartCmd(a))
	return cmd
}

func newHAStatusCmd() *cobra.Command {
	var file, rawURL string
	c := &cobra.Command{
		Use:   "status",
		Short: "Print Patroni or CloudNativePG topology. Nothing is changed.",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			body, source, err := readBody(c.Context(), file, rawURL)
			if err != nil {
				return err
			}
			var topo ha.Topology
			if source == "cnpg" || strings.Contains(string(body), `"instances"`) {
				topo, err = ha.ParseCNPG(body)
			} else {
				topo, err = ha.ParsePatroni(body)
			}
			if err != nil {
				return err
			}
			fmt.Fprint(c.OutOrStdout(), topo.Text())
			return nil
		},
	}
	c.Flags().StringVar(&file, "file", "", "JSON file from Patroni /cluster or a CloudNativePG status")
	c.Flags().StringVar(&rawURL, "url", "", "URL that returns the JSON")
	return c
}

func newHASwitchoverCmd(a *app) *cobra.Command {
	var action ha.Switchover
	var apply bool
	c := &cobra.Command{
		Use:   "switchover",
		Short: "Preview a switchover. --apply sends it after confirmation on prod.",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			fmt.Fprintln(c.OutOrStdout(), action.Preview())
			if !apply {
				return nil
			}
			if err := action.Check(); err != nil {
				return err
			}
			if err := recordAction(c.Context(), a, action.Environment, action.Cluster, "switchover", action.Preview(), ""); err != nil {
				return err
			}
			return postJSON(c.Context(), strings.TrimRight(action.API, "/")+"/switchover", action.Body())
		},
	}
	c.Flags().StringVar(&action.Environment, "env", "dev", "environment name")
	c.Flags().StringVar(&action.Cluster, "cluster", "", "cluster name")
	c.Flags().StringVar(&action.From, "from", "", "current leader")
	c.Flags().StringVar(&action.To, "to", "", "candidate")
	c.Flags().StringVar(&action.API, "api", "", "Patroni or operator API base URL")
	c.Flags().BoolVar(&action.Prod, "prod", false, "production: --confirm must be the cluster name")
	c.Flags().StringVar(&action.Confirm, "confirm", "", "cluster name, required on prod")
	c.Flags().BoolVar(&apply, "apply", false, "send the request")
	_ = c.MarkFlagRequired("cluster")
	_ = c.MarkFlagRequired("from")
	_ = c.MarkFlagRequired("to")
	_ = c.MarkFlagRequired("api")
	return c
}

func newHARestartCmd(a *app) *cobra.Command {
	var action ha.Restart
	var apply bool
	c := &cobra.Command{
		Use:   "restart",
		Short: "Preview a restart, replicas first. --apply sends it.",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			fmt.Fprintln(c.OutOrStdout(), action.Preview())
			if !apply {
				return nil
			}
			if err := action.Check(); err != nil {
				return err
			}
			if err := recordAction(c.Context(), a, action.Environment, action.Cluster, "restart", action.Preview(), ""); err != nil {
				return err
			}
			when := time.Now()
			if a.now != nil {
				when = a.now()
			}
			return postJSON(c.Context(), strings.TrimRight(action.API, "/")+"/restart", ha.RestartBody(when))
		},
	}
	c.Flags().StringVar(&action.Environment, "env", "dev", "environment name")
	c.Flags().StringVar(&action.Cluster, "cluster", "", "cluster name")
	c.Flags().StringVar(&action.Member, "member", "", "leader to restart after the replicas")
	c.Flags().StringVar(&action.API, "api", "", "Patroni or operator API base URL")
	c.Flags().BoolVar(&action.Prod, "prod", false, "production: --confirm must be the cluster name")
	c.Flags().StringVar(&action.Confirm, "confirm", "", "cluster name, required on prod")
	c.Flags().BoolVar(&apply, "apply", false, "send the request")
	_ = c.MarkFlagRequired("cluster")
	_ = c.MarkFlagRequired("api")
	return c
}

func readBody(ctx context.Context, file, rawURL string) ([]byte, string, error) {
	switch {
	case file != "" && rawURL != "":
		return nil, "", fmt.Errorf("pass --file or --url, not both")
	case file != "":
		body, err := os.ReadFile(file)
		return body, "", err
	case rawURL != "":
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, "", err
		}
		client := &http.Client{Timeout: 10 * time.Second}
		res, err := client.Do(req)
		if err != nil {
			return nil, "", err
		}
		body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
		closeErr := res.Body.Close()
		if err != nil {
			return nil, "", err
		}
		if closeErr != nil {
			return nil, "", closeErr
		}
		if res.StatusCode >= 300 {
			return nil, "", fmt.Errorf("GET %s: %s", rawURL, res.Status)
		}
		source := ""
		if strings.Contains(rawURL, "cnpg") {
			source = "cnpg"
		}
		return body, source, nil
	default:
		return nil, "", fmt.Errorf("pass --file or --url")
	}
}

func postJSON(ctx context.Context, rawURL, body string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 30 * time.Second}
	res, err := client.Do(req)
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

func recordAction(ctx context.Context, a *app, env, cluster, action, preview, failure string) error {
	path, err := store.DefaultPath()
	if err != nil {
		return err
	}
	db, err := store.Open(ctx, path)
	if err != nil {
		return err
	}
	when := time.Now()
	if a.now != nil {
		when = a.now()
	}
	recordErr := db.RecordAction(ctx, store.ActionAudit{
		At: when, Actor: "local", Environment: env, Cluster: cluster, Action: action, Preview: preview, Error: failure,
	})
	if err := db.Close(); err != nil && recordErr == nil {
		return err
	}
	return recordErr
}
