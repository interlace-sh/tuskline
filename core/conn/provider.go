// SPDX-License-Identifier: Apache-2.0

package conn

// Provider is where a server runs, as far as the probe can tell.
type Provider string

// Providers Tuskline recognises (docs/compatibility.md).
const (
	ProviderCommunity Provider = "community"
	ProviderRDS       Provider = "rds"
	ProviderAurora    Provider = "aurora"
	ProviderCloudSQL  Provider = "cloudsql"
	ProviderAlloyDB   Provider = "alloydb"
	ProviderAzure     Provider = "azure"
	ProviderNeon      Provider = "neon"
)

// ProviderSignals are the fingerprints the probe collects. Each is something
// a managed service creates and a self-managed server normally lacks.
type ProviderSignals struct {
	AuroraVersion     bool // function aurora_version()
	RDSSuperuser      bool // role rds_superuser
	RDSSettings       bool // GUC rds.extensions
	CloudSQLSuperuser bool // role cloudsqlsuperuser
	CloudSQLSettings  bool // GUC cloudsql.iam_authentication
	AlloyDBSuperuser  bool // role alloydbsuperuser
	AlloyDBSettings   bool // GUC alloydb.iam_authentication
	AzureAdmin        bool // role azure_pg_admin
	AzureSettings     bool // GUC azure.extensions
	NeonSettings      bool // GUC neon.timeline_id
	PreloadsNeon      bool // "neon" in shared_preload_libraries
}

// DetectProvider classifies a server from its signals. More specific
// services win: Aurora also has rds_superuser, and AlloyDB shares Cloud SQL
// conventions. These are heuristics; the inventory may override the result.
func DetectProvider(s ProviderSignals) Provider {
	switch {
	case s.AuroraVersion:
		return ProviderAurora
	case s.RDSSuperuser || s.RDSSettings:
		return ProviderRDS
	case s.AlloyDBSuperuser || s.AlloyDBSettings:
		return ProviderAlloyDB
	case s.CloudSQLSuperuser || s.CloudSQLSettings:
		return ProviderCloudSQL
	case s.AzureAdmin || s.AzureSettings:
		return ProviderAzure
	case s.NeonSettings || s.PreloadsNeon:
		return ProviderNeon
	}
	return ProviderCommunity
}
