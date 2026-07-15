# Security Policy

OctaAI executes shell commands, filesystem operations, SSH, HTTP requests, and browser automation based on LLM output. Treat it as a privileged local agent.

## Supported versions

Security fixes are applied on the `master` branch.

## Reporting a vulnerability

Please report security issues privately to the repository maintainer rather than opening a public GitHub issue.

Include:

- Description of the vulnerability and impact
- Steps to reproduce
- Suggested fix (if any)

## Security configuration

Review `config.example.yaml` before running the daemon:

- `safety.allow_paths` — filesystem and command `cwd` boundaries
- `safety.deny_commands` — blocked shell command patterns (normalized matching)
- `safety.require_confirmation_for` — commands requiring human approval
- `safety.allow_http_hosts` — optional HTTP host allowlist for `http` and `git clone`
- `browser.token` — required WebSocket token when browser automation is enabled
- `browser.browser_domains` — allowed navigation domains for the browser tool
- `ssh.known_hosts_file` — required for SSH host-key verification

## Threat model notes

- The permission manager gates all six built-in tools.
- Docker isolation is opt-in and should be validated before relying on it in production.
- Browser WebSocket connections require a token and restricted origins.
- HTTP tool requests block private/link-local addresses by default (SSRF mitigation).
