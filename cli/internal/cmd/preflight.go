// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/interlace-sh/tuskline/core/provision"
)

// probeCompose is replaced in tests so up can fail pre-flight without Docker.
var probeCompose = probeLocal

func probeLocal(ctx context.Context, need provision.Need, dir string) (provision.Report, error) {
	var report provision.Report
	report.Resume = journalExists(dir)
	disk, err := existingDir(dir)
	if err != nil {
		return report, err
	}
	report.FreeBytes, err = provision.FreeBytes(disk)
	if err != nil {
		return report, err
	}
	report.BusyPorts = map[int]bool{}
	for _, port := range need.Ports {
		free, err := provision.PortFree(ctx, port)
		if err != nil {
			return report, fmt.Errorf("port %d: %w", port, err)
		}
		if !free {
			report.BusyPorts[port] = true
		}
	}
	server, err := dockerOutput(ctx, "version", "--format", "{{.Server.Version}}")
	if err != nil {
		report.DockerError = err.Error()
		return report, nil
	}
	report.DockerServer = server
	compose, err := dockerOutput(ctx, "compose", "version", "--short")
	if err != nil {
		report.DockerError = err.Error()
		return report, nil
	}
	report.DockerCompose = compose
	names, err := dockerOutput(ctx, "volume", "ls", "--format", "{{.Name}}")
	if err != nil {
		report.DockerError = err.Error()
		return report, nil
	}
	report.Volumes = strings.Fields(names)
	ports, err := dockerOutput(ctx, "ps", "--filter", "label=com.docker.compose.project="+need.Name, "--format", "{{.Ports}}")
	if err != nil {
		report.DockerError = err.Error()
		return report, nil
	}
	report.OwnPorts = provision.PublishedPorts(ports)
	return report, nil
}

func journalExists(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, ".tuskline-apply.jsonl"))
	return err == nil && info.Mode().IsRegular() && info.Size() > 0
}

func existingDir(dir string) (string, error) {
	cur := dir
	for {
		info, err := os.Stat(cur)
		if err == nil && info.IsDir() {
			return cur, nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", fmt.Errorf("output directory %s has no existing parent", dir)
		}
		cur = parent
	}
}

func dockerOutput(ctx context.Context, args ...string) (string, error) {
	if _, err := exec.LookPath("docker"); err != nil {
		return "", fmt.Errorf("docker is not installed")
	}
	cmd := exec.CommandContext(ctx, "docker", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", errors.New(msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}
