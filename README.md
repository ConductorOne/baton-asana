`baton-asana` is a connector for Asana built using the [Baton SDK](https://github.com/conductorone/baton-sdk). It communicates with the Asana API to sync data about workspaces, users, and teams.

Check out [Baton](https://github.com/conductorone/baton) to learn more the project in general.

# Getting Started

## Prerequisites

1. Personal Acess Token. See more info [here](https://developers.asana.com/docs/personal-access-token).

## brew

```
brew install conductorone/baton/baton conductorone/baton/baton-asana
baton-asana
baton resources
```

## docker

```
docker run --rm -v $(pwd):/out -e BATON_TOKEN=token ghcr.io/conductorone/baton-asana:latest -f "/out/sync.c1z"
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

We started Baton because we were tired of taking screenshots and manually building spreadsheets. We welcome contributions, and ideas, no matter how small -- our goal is to make identity and permissions sprawl less painful for everyone. If you have questions, problems, or ideas: Please open a Github Issue!

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
  help               Help about any command

Flags:
      --client-id string       The client ID used to authenticate with ConductorOne ($BATON_CLIENT_ID)
      --client-secret string   The client secret used to authenticate with ConductorOne ($BATON_CLIENT_SECRET)
  -f, --file string            The path to the c1z file to sync with ($BATON_FILE) (default "sync.c1z")
  -h, --help                   help for baton-asana
      --log-format string      The output format for logs: json, console ($BATON_LOG_FORMAT) (default "json")
      --log-level string       The log level: debug, info, warn, error ($BATON_LOG_LEVEL) (default "info")
  -p, --provisioning           This must be set in order for provisioning actions to be enabled. ($BATON_PROVISIONING)
      --token string           The Asana personal access token used to connect to the Asana API. ($BATON_TOKEN)
  -v, --version                version for baton-asana

Use "baton-asana [command] --help" for more information about a command.

```
