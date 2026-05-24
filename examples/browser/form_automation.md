# Form Automation Example

Demonstrates how OctaAI can fill and submit HTML forms using the browser tool.

## Goal

```bash
./bin/octa-agent goal "Go to https://httpbin.org/forms/post, fill out all fields with realistic test data (name: Alice Smith, email: alice@example.com, phone: 555-1234, size: Large, pizza topping: Bacon), then submit the form and extract the response body"
```

## What the agent does

1. `navigate` to the form URL and waits for the page to load.
2. `fill` each input field by CSS selector.
3. Selects a radio button / checkbox with `click`.
4. `submit` the form.
5. `extract` the JSON response body from the resulting page.

## Equivalent browser tool calls (for reference)

```json
{ "action": "navigate", "url": "https://httpbin.org/forms/post" }

{ "action": "fill", "selector": "input[name='custname']",  "value": "Alice Smith" }
{ "action": "fill", "selector": "input[name='custtel']",   "value": "555-1234" }
{ "action": "fill", "selector": "input[name='custemail']", "value": "alice@example.com" }

{ "action": "click", "selector": "input[value='large']" }
{ "action": "click", "selector": "input[value='bacon']" }

{ "action": "fill", "selector": "textarea[name='comments']", "value": "Test comment" }

{ "action": "submit", "selector": "form" }

{ "action": "extract", "selector": "body" }
```

## Notes

- For React/Vue/Angular forms, `fill` dispatches both `input` and `change` events so framework state is updated correctly.
- If a field is hidden until another field is filled, add a `wait_for` between steps.
- Use `xpath` locator when CSS selectors are ambiguous: `"xpath": "//input[@placeholder='Your name']"`.
