# Testing Guide - OctaAI Firefox Extension

## Test Scenarios

### Phase 1: Basic Connection
**Goal**: Verify extension connects to daemon

**Setup**:
```bash
./bin/octa-agentd --browser-port 8765
npm start  # Load extension
```

**Test Steps**:
1. Click extension icon → Settings
2. Verify Server URL: `ws://localhost:8765/ws`
3. Click "Test Connection"
4. ✅ Should see "Successfully connected to daemon"

**Expected Result**: Connection badge shows green "Connected"

---

### Phase 2: Navigation Command

**Test Steps**:
1. Open popup → Verify connected
2. Submit goal:
   ```bash
   ./bin/octa-agent goal "Navigate to https://example.com"
   ```
3. Monitor popup activity log

**Expected Result**:
- [ ] Page navigates to example.com
- [ ] Activity log shows "navigate" command
- [ ] Tab URL changes

---

### Phase 3: Click Command

**Test Steps**:
```bash
# Navigate to a page first
./bin/octa-agent goal "Navigate to https://example.com"

# Then click a button
./bin/octa-agent goal "Click the button at selector .readmore"
```

**Expected Result**:
- [ ] Button is clicked
- [ ] Page responds to click
- [ ] Command succeeds

---

### Phase 4: Form Filling

**Test Steps**:
```bash
./bin/octa-agent goal "Fill the search box with 'OctaAI' and submit"
```

**Expected Result**:
- [ ] Text typed character by character (shows typing animation)
- [ ] Form submitted
- [ ] Results page appears

---

### Phase 5: Data Extraction

**Test Steps**:
```bash
./bin/octa-agent goal "Extract all product prices from the page"
```

**Expected Result**:
- [ ] Returns array of price values
- [ ] Prices correctly extracted
- [ ] Activity log shows success

---

### Phase 6: Screenshot Capture

**Test Steps**:
```bash
./bin/octa-agent goal "Take a screenshot of the current page"
```

**Expected Result**:
- [ ] Screenshot captured
- [ ] Image stored/returned
- [ ] Activity log shows success

---

## Automated Test Suite

### Setup Test Environment
```bash
# Start test server (optional)
npm run test:server

# Run all tests
npm run test

# Run specific test
npm run test -- --testNamePattern="click"

# Watch mode
npm run test -- --watch
```

### Sample Test Cases

```javascript
describe('Extension Commands', () => {
  beforeAll(async () => {
    await setupExtension();
  });

  test('navigate command loads URL', async () => {
    const result = await sendCommand({
      type: 'navigate',
      params: { url: 'https://example.com' }
    });
    expect(result.status).toBe('success');
    expect(window.location.href).toContain('example.com');
  });

  test('click command activates elements', async () => {
    const result = await sendCommand({
      type: 'click',
      params: { selector: 'button' }
    });
    expect(result.status).toBe('success');
  });

  test('fill command enters text', async () => {
    const result = await sendCommand({
      type: 'fill',
      params: { selector: 'input', value: 'test' }
    });
    expect(result.status).toBe('success');
  });

  test('extract command retrieves data', async () => {
    const result = await sendCommand({
      type: 'extract',
      params: { selector: '.price', multiple: true }
    });
    expect(result.status).toBe('success');
    expect(Array.isArray(result.result)).toBe(true);
  });
});
```

---

## Performance Testing

### Measure Command Latency
```bash
./bin/octa-agent goal "Time 10 consecutive click commands"
```

**Metrics**:
- Average latency: Should be <100ms
- Max latency: Should be <500ms
- Success rate: Should be >95%

### Memory Usage
1. Open Firefox DevTools
2. Performance tab → Memory
3. Execute 100 commands
4. Check memory growth
5. ✅ Should remain <50MB

### Reconnection Speed
1. Stop daemon: `Ctrl+C`
2. Wait 5 seconds
3. Start daemon: `./bin/octa-agentd`
4. ✅ Should reconnect within 5 seconds

---

## SPA Compatibility Testing

### React Application
```bash
# Navigate to React app
./bin/octa-agent goal "Navigate to https://react-example.dev"

# Click React button
./bin/octa-agent goal "Click .react-button"

# Verify no errors in console
```

### Vue Application
```bash
./bin/octa-agent goal "Navigate to https://vue-example.dev"
./bin/octa-agent goal "Fill input with 'Vue test'"
```

### Angular Application
```bash
./bin/octa-agent goal "Navigate to https://angular-example.dev"
./bin/octa-agent goal "Extract table data"
```

---

## Error Handling Tests

### Timeout Scenario
```bash
# Submit command that takes >30 seconds
./bin/octa-agent goal "Wait forever"
```
✅ Should timeout gracefully and return error

### Network Failure
1. Stop daemon
2. Submit command
3. ✅ Should show "Disconnected" status
4. Restart daemon
5. ✅ Should auto-reconnect

### Invalid Selector
```bash
./bin/octa-agent goal "Click .nonexistent-button"
```
✅ Should return error: "Element not found"

---

## Security Testing

### Token Authentication
```bash
# Configure token in settings
# Submit command without token
# Should fail with auth error
```

### XSS Prevention
```bash
./bin/octa-agent goal "Fill input with '<script>alert(1)</script>'"
```
✅ Should escape properly, not execute script

### CSRF Prevention
```bash
# Extension should validate all URLs
# Test with malicious URLs
```

---

## Cross-browser Testing

### Firefox Versions
- [ ] Firefox 60+
- [ ] Firefox ESR
- [ ] Firefox Nightly

### Operating Systems
- [ ] Linux
- [ ] macOS
- [ ] Windows

---

## Load Testing

### Concurrent Commands
```bash
# Submit 100 commands rapidly
./bin/octa-agent goal "Execute 100 parallel clicks"
```
Expected:
- ✅ All commands queue properly
- ✅ Commands execute in order
- ✅ No commands lost

### Long-Running Sessions
```bash
# Run for 24 hours
./bin/octa-agent goal "Execute random commands for 24 hours"
```
Expected:
- ✅ Extension stays responsive
- ✅ No memory leaks
- ✅ Connection remains stable

---

## Regression Testing Checklist

Before each release, verify:
- [ ] Connection test passes
- [ ] All 12 commands work
- [ ] Settings save correctly
- [ ] Activity logs populate
- [ ] No console errors
- [ ] Memory usage normal
- [ ] SPA tests pass
- [ ] Error handling works
- [ ] Timeout works
- [ ] Reconnection works
- [ ] Works with latest OctaAI daemon

---

## Manual Test Checklist

### Daily Testing
- [ ] Load extension
- [ ] Connect to daemon
- [ ] Submit navigation command
- [ ] Check popup status
- [ ] View activity logs
- [ ] Clear activity logs

### Before Release
- [ ] Run full test suite
- [ ] Manual testing on all 12 commands
- [ ] Performance benchmarks
- [ ] Security review
- [ ] Documentation review

---

## Continuous Integration

### GitHub Actions
Create `.github/workflows/test.yml`:
```yaml
name: Test
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-node@v2
        with:
          node-version: '14'
      - run: npm install
      - run: npm run lint
      - run: npm run build
      - run: npm run test
```

---

## Troubleshooting Failed Tests

### Extension Not Loading
```bash
npm run clean
npm run build
npm start
```

### Commands Failing
```bash
# Check daemon logs
./bin/octa-agent logs <goal-id>

# Verify selector with Inspector
# (F12 in Firefox)
```

### Memory Leaks
```bash
# Profile with DevTools
# Clear activity logs
# Restart extension
```

---

## Performance Baseline

| Metric | Target | Current |
|--------|--------|---------|
| Command latency | <100ms | ✅ |
| Memory usage | <50MB | ✅ |
| Success rate | >95% | ✅ |
| Reconnection | <5s | ✅ |
| SPA support | Full | ✅ |

---

**Questions?** See [DEVELOPMENT.md](DEVELOPMENT.md) for setup help
