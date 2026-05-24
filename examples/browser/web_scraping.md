# Web Scraping Example

Demonstrates how to have OctaAI extract structured data from web pages using the browser tool.

## Goal

```bash
./bin/octa-agent goal "Navigate to https://github.com/trending, extract the name and description of the top 10 repositories, and save the results as JSON to /tmp/trending.json"
```

## What the agent does

1. Sends a `navigate` command to open the GitHub Trending page.
2. Uses `extract` with `multiple: true` to collect all repository link texts.
3. Uses a second `extract` call to collect the description paragraphs.
4. Uses `execute` to build and return a JSON array.
5. Writes the result to `/tmp/trending.json` using the `filesystem` tool.

## Equivalent browser tool calls (for reference)

```json
{ "action": "navigate", "url": "https://github.com/trending" }

{ "action": "extract", "selector": "h2.h3 a", "multiple": true }

{ "action": "extract", "selector": "p.col-9", "multiple": true }

{
  "action": "execute",
  "script": "return JSON.stringify(Array.from(document.querySelectorAll('article.Box-row')).slice(0,10).map(r => ({ name: r.querySelector('h2 a')?.textContent.trim(), description: r.querySelector('p')?.textContent.trim() })))"
}
```

## Notes

- `multiple: true` returns `{ results: string[], count: number }`.
- If the page uses infinite scroll, add a `scroll` command first to load more results.
- For login-gated pages, see `login_automation.md`.
