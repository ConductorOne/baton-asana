# baton-asana [![Go Reference](https://pkg.go.dev/badge/github.com/conductorone/baton-asana.svg)](https://pkg.go.dev/github.com/conductorone/baton-asana) ![ci](https://github.com/conductorone/baton-asana/actions/workflows/ci.yaml/badge.svg) ![verify](https://github.com/conductorone/baton-asana/actions/workflows/verify.yaml/badge.svg)

`baton-asana` is a connector for Asana built using the [Baton SDK](https://github.com/conductorone/baton-sdk). It
communicates with the Asana API to sync data about workspaces, users, and teams.

Check out [Baton](https://github.com/conductorone/baton) to learn more the project in general.

# Getting Started

## Prerequisites

1. Authentication Token:
   - Personal Access Token: See more info [here](https://developers.asana.com/docs/personal-access-token)
   - Service Account Token: (For Enterprise Asana customers) Can be used by setting the `--use-service-account` flag or `BATON_USE_SERVICE_ACCOUNT=true` environment variable

## brew

```
brew install conductorone/baton/baton conductorone/baton/baton-asana
baton-asana
baton resources
```

## docker

```
docker run --rm -v $(pwd):/out -e BATON_TOKEN=token public.ecr.aws/conductorone/baton-asana:latest -f "/out/sync.c1z"
docker run --rm -v $(pwd):/out ghcr.io/conductorone/baton:latest -f "/out/sync.c1z" resources
```

## source

```
go install github.com/conductorone/baton/cmd/baton@main
go install github.com/conductorone/baton-asana/cmd/baton-asana@main

BATON_TOKEN=token
baton resources
```

# Data Model

`baton-asana` pulls down information about the following Asana resources:

- Workspaces
- Users
- Teams

# Contributing, Support, and Issues

We started Baton because we were tired of taking screenshots and manually building spreadsheets. We welcome
contributions, and ideas, no matter how small -- our goal is to make identity and permissions sprawl less painful for
everyone. If you have questions, problems, or ideas: Please open a Github Issue!

See [CONTRIBUTING.md](https://github.com/ConductorOne/baton/blob/main/CONTRIBUTING.md) for more details.

# `baton-asana` Command Line Usage

```
baton-asana

Usage:
  baton-asana [flags]
  baton-asana [command]

Available Commands:
  capabilities       Get connector capabilities
  completion         Generate the autocompletion script for the specified shell
  config             Get connector config
  help               Help about any command

Flags:
      --asana-api-url string             Override the default Asana API URL (for testing with a mock server) ($BATON_ASANA_API_URL)
      --client-id string                 The client ID used to authenticate with ConductorOne ($BATON_CLIENT_ID)
      --client-secret string             The client secret used to authenticate with ConductorOne ($BATON_CLIENT_SECRET)
      --default-workspace-id string      The default workspace ID to use for account provisioning ($BATON_DEFAULT_WORKSPACE_ID)
  -f, --file string                      The path to the c1z file to sync with ($BATON_FILE) (default "sync.c1z")
  -h, --help                             help for baton-asana
      --log-format string                The output format for logs: json, console ($BATON_LOG_FORMAT) (default "json")
      --log-level string                 The log level: debug, info, warn, error ($BATON_LOG_LEVEL) (default "info")
      --otel-collector-endpoint string   The endpoint of the OpenTelemetry collector to send observability data to ($BATON_OTEL_COLLECTOR_ENDPOINT)
  -p, --provisioning                     This must be set in order for provisioning actions to be enabled ($BATON_PROVISIONING)
      --skip-full-sync                   This must be set to skip a full sync ($BATON_SKIP_FULL_SYNC)
      --ticketing                        This must be set to enable ticketing support ($BATON_TICKETING)
      --token string                     required: Your Asana API key (Personal Access Token or Service Account Token) ($BATON_TOKEN)
      --use-scim-api                     Set to true to use the Asana SCIM API for enterprise license management and user provisioning ($BATON_USE_SCIM_API)
      --use-service-account              Set to true if using a service account token instead of a personal access token ($BATON_USE_SERVICE_ACCOUNT)
  -v, --version                          version for baton-asana

Use "baton-asana [command] --help" for more information about a command.

```
