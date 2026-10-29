module github.com/interlace-sh/tuskline/cli

go 1.25.0

require (
	github.com/interlace-sh/tuskline/agent v0.0.0
	github.com/interlace-sh/tuskline/core v0.0.0
	github.com/spf13/cobra v1.10.2
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.11.0 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/text v0.40.0 // indirect
)

replace (
	github.com/interlace-sh/tuskline/agent => ../agent
	github.com/interlace-sh/tuskline/core => ../core
)
