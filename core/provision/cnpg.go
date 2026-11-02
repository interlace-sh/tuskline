// SPDX-License-Identifier: Apache-2.0

package provision

import (
	"fmt"
	"strings"

	"github.com/interlace-sh/tuskline/core/inventory"
)

// CNPGSpec is a CloudNativePG cluster. Unlike Compose, it may be production.
type CNPGSpec struct {
	Name      string
	Namespace string
	Major     int
	Instances int
	Pooler    bool
	// Storage is a size such as 10Gi. Empty means 10Gi.
	Storage string
	// Destination is the Barman object path, such as s3://bucket/name.
	// Credentials are not embedded; the operator secret is named separately.
	Destination string
	SecretName  string
}

// CNPG builds Cluster, Pooler and Barman Cloud manifests. Nothing is applied.
func CNPG(spec CNPGSpec) (Plan, error) {
	spec, err := normaliseCNPG(spec)
	if err != nil {
		return Plan{}, err
	}
	plan := Plan{
		Target: "cnpg",
		Steps: []Step{
			{ID: "cluster", Kind: KindManifest, Path: "cluster.yaml", After: cnpgCluster(spec)},
			{ID: "store", Kind: KindManifest, Path: "objectstore.yaml", After: cnpgStore(spec)},
			{ID: "backup", Kind: KindManifest, Path: "scheduledbackup.yaml", After: cnpgSchedule(spec)},
		},
	}
	if spec.Pooler {
		plan.Steps = append(plan.Steps, Step{ID: "pooler", Kind: KindManifest, Path: "pooler.yaml", After: cnpgPooler(spec)})
	}
	return plan, nil
}

// CNPGApply is CNPG plus a kubectl apply. The manifests are on stdin.
// The command does not contain a secret.
func CNPGApply(spec CNPGSpec) (Plan, error) {
	plan, err := CNPG(spec)
	if err != nil {
		return Plan{}, err
	}
	plan.Steps = append(plan.Steps, Step{
		ID: "apply", Kind: KindCommand, Path: "kubectl apply",
		After: "kubectl apply -f cluster.yaml -f objectstore.yaml -f scheduledbackup.yaml",
		Argv:  []string{"kubectl", "apply", "-f", "cluster.yaml", "-f", "objectstore.yaml", "-f", "scheduledbackup.yaml"},
		Rerun: true,
	})
	if spec.Pooler {
		plan.Steps[len(plan.Steps)-1].Argv = append(plan.Steps[len(plan.Steps)-1].Argv, "-f", "pooler.yaml")
		plan.Steps[len(plan.Steps)-1].After += " -f pooler.yaml"
	}
	return plan, nil
}

func normaliseCNPG(spec CNPGSpec) (CNPGSpec, error) {
	if err := inventory.ValidateName(spec.Name); err != nil {
		return spec, fmt.Errorf("cluster name: %w", err)
	}
	if spec.Namespace == "" {
		spec.Namespace = "default"
	}
	if err := inventory.ValidateName(spec.Namespace); err != nil {
		return spec, fmt.Errorf("namespace: %w", err)
	}
	if spec.Major == 0 {
		spec.Major = 17
	}
	if spec.Major < 14 || spec.Major > 18 {
		return spec, fmt.Errorf("PostgreSQL %d is outside 14–18", spec.Major)
	}
	if spec.Instances == 0 {
		spec.Instances = 3
	}
	if spec.Instances < 1 || spec.Instances > 5 {
		return spec, fmt.Errorf("instances must be from 1 to 5")
	}
	if spec.Storage == "" {
		spec.Storage = "10Gi"
	}
	if strings.ContainsAny(spec.Storage, " \t\n") {
		return spec, fmt.Errorf("storage size %q is not a single token", spec.Storage)
	}
	if spec.Destination == "" {
		spec.Destination = "s3://demo-backups/" + spec.Name
	}
	if strings.Contains(spec.Destination, " ") || !strings.Contains(spec.Destination, "://") {
		return spec, fmt.Errorf("destination must be an object URL")
	}
	if spec.SecretName == "" {
		spec.SecretName = spec.Name + "-backup"
	}
	if err := inventory.ValidateName(spec.SecretName); err != nil {
		return spec, fmt.Errorf("secret name: %w", err)
	}
	return spec, nil
}

func cnpgCluster(spec CNPGSpec) string {
	return fmt.Sprintf(`apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: %s
  namespace: %s
spec:
  instances: %d
  imageName: ghcr.io/cloudnative-pg/postgresql:%d
  storage:
    size: %s
  plugins:
    - name: barman-cloud.cloudnative-pg.io
      isWALArchiver: true
      parameters:
        barmanObjectName: %s-store
`, spec.Name, spec.Namespace, spec.Instances, spec.Major, spec.Storage, spec.Name)
}

func cnpgStore(spec CNPGSpec) string {
	return fmt.Sprintf(`apiVersion: barmancloud.cnpg.io/v1
kind: ObjectStore
metadata:
  name: %s-store
  namespace: %s
spec:
  configuration:
    destinationPath: %s
  retentionPolicy: "14d"
  instanceSidecarConfiguration:
    env:
      - name: AWS_ACCESS_KEY_ID
        valueFrom:
          secretKeyRef:
            name: %s
            key: ACCESS_KEY_ID
      - name: AWS_SECRET_ACCESS_KEY
        valueFrom:
          secretKeyRef:
            name: %s
            key: ACCESS_SECRET_KEY
`, spec.Name, spec.Namespace, spec.Destination, spec.SecretName, spec.SecretName)
}

func cnpgSchedule(spec CNPGSpec) string {
	return fmt.Sprintf(`apiVersion: postgresql.cnpg.io/v1
kind: ScheduledBackup
metadata:
  name: %s-daily
  namespace: %s
spec:
  schedule: "0 0 2 * * *"
  cluster:
    name: %s
  method: plugin
  pluginConfiguration:
    name: barman-cloud.cloudnative-pg.io
`, spec.Name, spec.Namespace, spec.Name)
}

func cnpgPooler(spec CNPGSpec) string {
	return fmt.Sprintf(`apiVersion: postgresql.cnpg.io/v1
kind: Pooler
metadata:
  name: %s-pooler-rw
  namespace: %s
spec:
  cluster:
    name: %s
  instances: 2
  type: rw
  pgbouncer:
    poolMode: transaction
`, spec.Name, spec.Namespace, spec.Name)
}
