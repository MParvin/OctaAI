# Contributing to OctaAI

Thank you for contributing. This project is a Go CLI + daemon agent for autonomous goal execution.

## Development setup

```bash
make deps
make build
make test
```

## Code style

- Run `make fmt` and `make vet` before opening a PR.
- Keep changes focused; match existing package layout and naming.
- Add tests for safety-critical behavior (`pkg/permission`, `pkg/tools`).

## Pull requests

1. Fork and create a feature branch.
2. Ensure `make test` passes locally.
3. Describe behavior changes and security impact when touching tools or permissions.
4. Do not commit personal `config.yaml` files or SQLite databases.

## Reporting security issues

See [SECURITY.md](SECURITY.md). Do not open public issues for exploitable vulnerabilities.
