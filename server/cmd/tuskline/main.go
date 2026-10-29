// SPDX-License-Identifier: AGPL-3.0-only

// Command tuskline is the Tuskline server (AGPL-3.0): multi-user API, web UI,
// fleet history and alerting. "tuskline serve" is what the Docker image runs.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/interlace-sh/tuskline/core/version"
	"github.com/interlace-sh/tuskline/server/internal/httpapi"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "serve" {
		if len(os.Args) >= 2 && os.Args[1] == "version" {
			fmt.Println("tuskline", version.String())
			return
		}
		fmt.Fprintln(os.Stderr, "usage: tuskline serve [-addr :8080] | tuskline version")
		os.Exit(2)
	}
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := fs.String("addr", envOr("TUSKLINE_ADDR", "127.0.0.1:8080"), "listen address")
	_ = fs.Parse(os.Args[2:])

	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	srv := &http.Server{Addr: *addr, Handler: httpapi.New(), ReadHeaderTimeout: 10 * time.Second}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(sctx)
	}()
	log.Info("tuskline listening", "addr", *addr, "version", version.Version)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
