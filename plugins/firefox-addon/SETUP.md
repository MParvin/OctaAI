# Setup & Installation Guide

Complete step-by-step guide to get the OctaAI Firefox Extension up and running.

## System Requirements

- **Node.js**: 14.0 or higher
- **npm**: 6.0 or higher
- **Firefox**: 60 or higher
- **Operating System**: Linux, macOS, or Windows
- **OctaAI Daemon**: Running on `localhost:8765` (configurable)

**Verify installed**:
```bash
node --version    # Should be v14.0+
npm --version     # Should be 6.0+
firefox --version # Should be 60+
```

---

## Installation Steps

### Step 1: Clone or Download the Project

```bash
cd /home/mparvin/Documents/MyGit/Projects/OctaAI
cd plugins/firefox-addon
```

### Step 2: Install Dependencies

```bash
npm install
```

This installs:
- `webpack` - Module bundler
- `webpack-cli` - CLI for webpack
- `web-ext` - Firefox extension development tool
- `eslint` - Code linter
- `copy-webpack-plugin` - Asset copying

### Step 3: Build the Extension

```bash
npm run build
```

This creates the `dist/` directory with the compiled extension ready to load.

**Output**:
```
dist/
├── manifest.json
├── background/background.js
├── content/content.js
├── popup/popup.html
├── popup/popup.js
├── options/options.html
├── options/options.js
└── icons/
```

### Step 4: Create Icons (Optional)

The extension requires three icon sizes. You can:

**Option A**: Generate with ImageMagick
```bash
cd src/icons
convert -size 96x96 xc:#667eea -font Arial -pointsize 50 \
  -draw "gravity center fill white text 0,0 'OA'" icon-96.png
convert icon-96.png -resize 48x48 icon-48.png
convert icon-96.png -resize 16x16 icon-16.png
cd ../..
npm run build  # Rebuild with icons
```

**Option B**: Download placeholder icons
```bash
# Icons are optional for development
# The extension will load without them
```

### Step 5: Load in Firefox

#### Method 1: Automatic (Recommended for Development)

```bash
npm start
```

This:
1. Opens Firefox automatically
2. Loads the extension in development mode
3. Opens browser console
4. Enables live reload

#### Method 2: Manual Loading

1. **Open Firefox**
2. **Navigate to** `about:debugging`
3. **Click** "This Firefox"
4. **Click** "Load Temporary Add-on"
5. **Select** `/home/mparvin/Documents/MyGit/Projects/OctaAI/plugins/firefox-addon/dist/manifest.json`

The extension appears in your toolbar.

---

## Starting the Daemon

Before testing commands, ensure the OctaAI daemon is running:

```bash
# Terminal 1: Navigate to OctaAI root
cd /home/mparvin/Documents/MyGit/Projects/OctaAI

# Terminal 2: Start the daemon
./bin/octa-agentd --browser-port 8765
```

**Expected Output**:
```
[INFO] OctaAI Daemon started
[INFO] WebSocket server listening on ws://localhost:8765/ws
[INFO] Browser automation enabled on port 8765
```

### Verify Daemon is Running

```bash
# In another terminal
curl http://localhost:8765/health
```

If successful, you'll see a WebSocket upgrade response (connection closed message is normal).

---

## Configuration

### Default Settings

The extension uses these defaults:
- **Server URL**: `ws://localhost:8765/ws`
- **Token**: (empty - only if daemon requires it)
- **Auto-connect**: Enabled
- **Command Timeout**: 30 seconds
- **Typing Delay**: 50ms

### Customize Settings

1. **Click** the OctaAI icon in the toolbar
2. **Click** "Settings" button
3. **Modify** any settings
4. **Test Connection** to verify (optional)
5. **Click** "Save Settings"

Settings are saved automatically.

---

## First Test

### Test 1: Connection

1. Click OctaAI icon
2. Verify status shows "Connected" (green dot)
3. If not connected, click "Connect"

### Test 2: Navigate

```bash
# Terminal with OctaAI CLI
./bin/octa-agent goal "Navigate to https://example.com"

# Check status
./bin/octa-agent status

# View result
./bin/octa-agent logs <goal-id>
```

In Firefox, the page should navigate to example.com.

### Test 3: All Commands

```bash
# Run quick command test
./bin/octa-agent goal "Click any button on the page"

# Monitor popup activity log
# (Updates in real-time)
```

---

## Development Workflow

### Watch & Rebuild

Keep this running while developing:

```bash
npm run dev
```

Changes to `src/` files automatically rebuild.

### Development Servers

**Terminal 1**: Watch & rebuild
```bash
npm run dev
```

**Terminal 2**: OctaAI daemon
```bash
cd ../../
./bin/octa-agentd --browser-port 8765
```

**Terminal 3**: Firefox with extension
```bash
npm start
```

### Check for Errors

1. **Press F12** in Firefox
2. **Click** "Console" tab
3. **Filter** for `[Background]` or `[Content]` logs
4. **Check** for red error messages

---

## Common Setup Issues

### Issue: npm install fails

**Solution**:
```bash
npm cache clean --force
rm -rf node_modules package-lock.json
npm install
```

### Issue: WebSocket connection refused

**Check**:
- [ ] Daemon is running: `ps aux | grep octa-agentd`
- [ ] Server URL correct: Settings page
- [ ] Port accessible: `curl http://localhost:8765`

**Fix**:
```bash
# Verify daemon
./bin/octa-agentd --browser-port 8765

# Update server URL if needed
# Open extension Settings → change URL → Save
```

### Issue: Extension won't load

**Check**:
- [ ] manifest.json exists in dist/
- [ ] No syntax errors: `npm run lint`

**Fix**:
```bash
npm run clean
npm run build
npm start
```

### Issue: Firefox permissions

**Fix**:
1. Firefox may ask for permissions
2. **Accept all** prompts
3. **Refresh** extension if needed

### Issue: Icons missing

**Note**: Icons are optional for development. Extension works without them. To add them:

```bash
cd src/icons
# Generate or download 16x16, 48x48, 96x96 PNG files
npm run build  # Rebuild with icons
```

---

## Verify Installation

Run this checklist to ensure everything works:

- [ ] `node --version` shows 14+
- [ ] `npm --version` shows 6+
- [ ] `npm install` completed without errors
- [ ] `npm run build` creates `dist/` folder
- [ ] Daemon runs: `./bin/octa-agentd --browser-port 8765`
- [ ] Extension loads in Firefox
- [ ] Extension connects to daemon (green dot)
- [ ] Activity log shows events
- [ ] Can navigate to a website

If all pass, you're ready to develop!

---

## Build & Distribution

### For Testing (Local)

```bash
npm run build
```

### For Firefox Add-ons Store

```bash
npm run package
```

Creates `octaai-addon.xpi` - ready to submit to Firefox Add-ons.

---

## Uninstall

### Remove from Firefox

1. **Open** Firefox
2. **Go to** `about:addons`
3. **Find** "OctaAI Agent"
4. **Click** Remove

### Remove Files

```bash
cd /home/mparvin/Documents/MyGit/Projects/OctaAI
rm -rf plugins/firefox-addon
```

---

## Getting Help

| Resource | Link |
|----------|------|
| README | [README.md](README.md) |
| Development | [DEVELOPMENT.md](DEVELOPMENT.md) |
| Architecture | [ARCHITECTURE.md](ARCHITECTURE.md) |
| Testing | [TESTING.md](TESTING.md) |
| Quick Ref | [QUICK_REFERENCE.md](QUICK_REFERENCE.md) |
| Firefox WebExtensions | https://developer.mozilla.org/en-US/docs/Mozilla/Add-ons/WebExtensions |
| OctaAI Docs | ../../docs/ |

---

## Next Steps

1. ✅ Follow installation steps above
2. ✅ Verify connection works
3. ⏭️  Run test commands (see TESTING.md)
4. ⏭️  Develop custom features (see DEVELOPMENT.md)
5. ⏭️  Submit to Firefox Add-ons (see Phase 10)

---

**Setup Complete!** You're ready to automate browser tasks with OctaAI. 🚀

For questions or issues, check [DEVELOPMENT.md](DEVELOPMENT.md) or [README.md](README.md).
