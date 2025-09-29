# New API Electron Desktop App

This directory contains the Electron wrapper for New API, allowing it to run as a native desktop application on Windows, macOS, and Linux.

## Architecture

The Electron app consists of:
- **main.js**: Main process that spawns the Go backend server and creates the application window
- **preload.js**: Preload script for secure context isolation
- **package.json**: Electron dependencies and build configuration

## Development

### Prerequisites

1. Build the Go backend first:
```bash
cd ..
go build -o new-api
```

2. Install Electron dependencies:
```bash
cd electron
npm install
```

### Running in Development Mode

```bash
npm start
```

This will:
- Start the Go backend on port 3000
- Open an Electron window pointing to `http://localhost:3000`
- Enable DevTools for debugging

## Building for Production

### Quick Build (Current Platform)

Use the provided build script:
```bash
./build.sh
```

This will:
1. Build the frontend (web/dist)
2. Build the Go binary for your platform
3. Package the Electron app

### Manual Build Steps

1. Build frontend:
```bash
cd ../web
DISABLE_ESLINT_PLUGIN='true' bun run build
```

2. Build backend:
```bash
cd ..
# macOS/Linux
go build -ldflags="-s -w" -o new-api

# Windows
go build -ldflags="-s -w" -o new-api.exe
```

3. Build Electron app:
```bash
cd electron
npm install

# All platforms
npm run build

# Or specific platforms
npm run build:mac    # macOS (DMG, ZIP)
npm run build:win    # Windows (NSIS installer, Portable)
npm run build:linux  # Linux (AppImage, DEB)
```

### Output

Built apps are located in `electron/dist/`:
- **macOS**: `.dmg` and `.zip`
- **Windows**: `.exe` installer and portable `.exe`
- **Linux**: `.AppImage` and `.deb`

## Cross-Platform Building

To build for other platforms:

```bash
# From macOS, build Windows app
npm run build:win

# From macOS, build Linux app
npm run build:linux
```

Note: Building macOS apps requires macOS. Building Windows apps with code signing requires Windows.

## Configuration

### Port

The app uses port 3000 by default. To change:

Edit `electron/main.js`:
```javascript
const PORT = 3000; // Change to your desired port
```

### Data Directory

- **Development**: Uses `data/` in the project root
- **Production**: Uses Electron's `userData` directory:
  - macOS: `~/Library/Application Support/New API/data/`
  - Windows: `%APPDATA%/New API/data/`
  - Linux: `~/.config/New API/data/`

### Window Size

Edit `electron/main.js` in the `createWindow()` function:
```javascript
mainWindow = new BrowserWindow({
  width: 1400,  // Change width
  height: 900,  // Change height
  // ...
});
```

## Troubleshooting

### Server fails to start

Check the console logs in DevTools (Cmd/Ctrl+Shift+I). Common issues:
- Go binary not found (ensure it's built)
- Port 3000 already in use
- Database file permission issues

### Binary not found in production

Ensure the Go binary is built before running `electron-builder`:
```bash
go build -o new-api      # macOS/Linux
go build -o new-api.exe  # Windows
```

The binary must be in the project root, not inside `electron/`.

### Database issues

If you encounter database errors, delete the data directory and restart:
- Dev: `rm -rf data/`
- Prod: Clear Electron's userData folder (see "Data Directory" above)

## Icon

To add a custom icon:
1. Place a 512x512 PNG icon at `electron/icon.png`
2. Rebuild the app with `npm run build`

## Security

- Context isolation is enabled
- Node integration is disabled in renderer process
- Only safe APIs are exposed via preload script
- Backend runs as a local subprocess with no external network access by default