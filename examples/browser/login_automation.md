# Login Automation Example

Demonstrates how OctaAI handles authentication flows before scraping protected pages.

## Goal

```bash
./bin/octa-agent goal "Log in to https://example.com/login with username 'admin' and password 'secret', then navigate to /dashboard and extract the account balance element"
```

## What the agent does

1. `navigate` to the login page.
2. `wait_for` the username field to be visible.
3. `fill` username and password.
4. `click` the submit button (or `submit` the form).
5. `wait_for` a post-login element to confirm success (e.g. a nav link only shown when authenticated).
6. `navigate` to the protected page.
7. `extract` the desired data.

## Equivalent browser tool calls (for reference)

```json
{ "action": "navigate", "url": "https://example.com/login" }

{ "action": "wait_for", "selector": "input[name='username']", "timeout": 5000 }

{ "action": "fill", "selector": "input[name='username']", "value": "admin" }
{ "action": "fill", "selector": "input[name='password']", "value": "secret" }

{ "action": "click", "text": "Log in" }

{ "action": "wait_for", "selector": ".dashboard-nav", "timeout": 10000 }

{ "action": "navigate", "url": "https://example.com/dashboard" }

{ "action": "extract", "selector": ".account-balance" }
```

## Persisting the session

Cookies are preserved automatically while the browser tab remains open. The agent can call `get_cookies` to export the session and reuse it across goals:

```json
{ "action": "get_cookies", "domain": "example.com" }
```

## Security notes

- Never hardcode passwords in goal strings that are stored in history. Use environment variables or a secrets manager and reference them in the goal: `"use the password from the APP_PASSWORD environment variable"`.
- Ensure `config.yaml` has a `browser_domains` whitelist when running login automations to prevent the agent from sending credentials to unintended sites.
