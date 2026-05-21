# Data Extraction Example

Demonstrates bulk and structured data extraction patterns for research and monitoring tasks.

## Single element

```bash
./bin/octa-agent goal "Go to https://example.com and extract the text of the main heading"
```

```json
{ "action": "navigate", "url": "https://example.com" }
{ "action": "extract", "selector": "h1" }
```

## Multiple elements (list)

```bash
./bin/octa-agent goal "Go to https://news.ycombinator.com and extract all article titles on the front page, save to /tmp/hn.txt"
```

```json
{ "action": "navigate", "url": "https://news.ycombinator.com" }
{ "action": "extract", "selector": ".titleline > a", "multiple": true }
```

Returns `{ "results": ["Title 1", "Title 2", ...], "count": 30 }`.

## Extracting attributes

```bash
./bin/octa-agent goal "Extract all image URLs from https://example.com/gallery"
```

```json
{ "action": "navigate", "url": "https://example.com/gallery" }
{ "action": "extract", "selector": "img", "attribute": "src", "multiple": true }
```

## Structured data via script

When you need to correlate data from multiple elements into objects, use `execute`:

```json
{
  "action": "execute",
  "script": "return JSON.stringify(Array.from(document.querySelectorAll('table tbody tr')).map(row => { const cells = row.querySelectorAll('td'); return { name: cells[0]?.textContent.trim(), price: cells[1]?.textContent.trim(), change: cells[2]?.textContent.trim() }; }))"
}
```

## XPath extraction

XPath is useful when CSS selectors are ambiguous or the structure is deeply nested:

```json
{
  "action": "extract",
  "xpath": "//div[@class='product-price']//span[@itemprop='price']",
  "attribute": "content"
}
```

## Page source

To hand the full HTML to an LLM for further parsing:

```json
{ "action": "get_page_source" }
```

Returns `{ "source": "<html>...</html>" }`.

## Tips

- Always `wait_for` dynamically loaded content before extracting: 
  `{ "action": "wait_for", "selector": ".results-container" }`
- Use `scroll` to trigger infinite-scroll pagination:
  `{ "action": "scroll", "y": 3000 }` then re-extract.
- Chain `navigate` + `extract` in a loop (via `execute`) to paginate across multiple pages.
