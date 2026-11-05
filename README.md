<div align="center">

# Auth Service

Authentication service: identity, tokens and sessions.

[![CI](https://github.com/dz-market/svc-auth/actions/workflows/ci.yml/badge.svg)](https://github.com/dz-market/svc-auth/actions/workflows/ci.yml)
[![Go version](https://img.shields.io/github/go-mod/go-version/dz-market/svc-auth)](go.mod)
[![Release](https://img.shields.io/github/v/release/dz-market/svc-auth)](https://github.com/dz-market/svc-auth/releases)
[![License](https://img.shields.io/github/license/dz-market/svc-auth)](LICENSE)

</div>

## Run

Required Docker Compose and Task. Clone [infra](https://github.com/dz-market/infra) alongside this repository.

```bash
task --dir ../infra up -- auth
```

## Development

Requires Go, Task and Goose. With infrastructure running, stop the containerized service and run it locally:

```bash
docker compose -f ../infa/local/compose.yml --profile auth stop svc-auth
cp .env.example .env # adjust connection settings
task migrate:up
go run ./cmd/service
```

Signing keys are loaded from Vault using the AppRole credentials configured in `.env`.