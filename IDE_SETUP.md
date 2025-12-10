# IDE Setup Guide

## Overview
This guide helps you configure your IDE (VS Code, GoLand, PyCharm, etc.) to work with the Go-Kafkify project.

## Important Note
**The code runs perfectly in Docker containers** - this setup is only for IDE IntelliSense, code completion, and linting.

## Go Services Setup

### Prerequisites
- Go 1.21 or later installed locally
- Run: `go version` to verify

### Setup Steps

1. **Generate go.sum files** (already done):
   ```bash
   cd services/rest-service && go mod tidy
   cd ../grpc-service && go mod tidy
   ```

2. **Configure VS Code** (if using):
   - Install the "Go" extension by Go Team at Google
   - Open Command Palette (Ctrl+Shift+P / Cmd+Shift+P)
   - Run: "Go: Install/Update Tools"
   - Select all tools and install

3. **Configure GoLand/IntelliJ** (if using):
   - Go to Settings → Go → GOROOT
   - Ensure it points to your Go installation
   - Enable "Go Modules" integration
   - The IDE should auto-detect go.mod files

### Verify
Open any `.go` file. You should now see:
- ✅ No import errors
- ✅ Code completion working
- ✅ Go to definition works
- ✅ Hover documentation appears

## Python Service Setup

### Prerequisites
- Python 3.11 or later installed locally
- Run: `python3 --version` to verify

### Setup Steps

1. **Virtual environment created** (already done):
   ```bash
   cd services/python-worker
   python3 -m venv .venv
   source .venv/bin/activate  # On Windows: .venv\Scripts\activate
   pip install -r requirements.txt
   ```

2. **Configure VS Code** (if using):
   - Install the "Python" extension by Microsoft
   - Open Command Palette (Ctrl+Shift+P / Cmd+Shift+P)
   - Run: "Python: Select Interpreter"
   - Choose: `./services/python-worker/.venv/bin/python`

3. **Configure PyCharm** (if using):
   - Go to Settings → Project → Python Interpreter
   - Click gear icon → Add
   - Select "Existing environment"
   - Navigate to: `services/python-worker/.venv/bin/python`

4. **Manual activation** (for terminal work):
   ```bash
   cd services/python-worker
   source .venv/bin/activate
   # Now python and pip use the virtual environment
   ```

### Verify
Open `services/python-worker/main.py`. You should now see:
- ✅ No import errors on kafka, psycopg2, opentelemetry
- ✅ Code completion working
- ✅ Type hints working
- ✅ Hover documentation appears

## VS Code Workspace Settings (Optional)

Create `.vscode/settings.json` in project root:

```json
{
  "go.useLanguageServer": true,
  "go.gopath": "",
  "go.goroot": "",
  "go.toolsGopath": "",
  "go.inferGopath": true,
  "python.defaultInterpreterPath": "${workspaceFolder}/services/python-worker/.venv/bin/python",
  "python.linting.enabled": true,
  "python.linting.pylintEnabled": true,
  "python.formatting.provider": "black",
  "editor.formatOnSave": true,
  "[go]": {
    "editor.formatOnSave": true,
    "editor.codeActionsOnSave": {
      "source.organizeImports": true
    }
  },
  "[python]": {
    "editor.formatOnSave": true,
    "editor.codeActionsOnSave": {
      "source.organizeImports": true
    }
  }
}
```

## Troubleshooting

### Go: "cannot find package"
**Solution:**
```bash
cd services/rest-service  # or grpc-service
go mod download
go mod tidy
```

### Python: "No module named 'kafka'"
**Solution:**
```bash
cd services/python-worker
source .venv/bin/activate
pip install -r requirements.txt
```

### VS Code: Python interpreter not found
**Solution:**
1. Open Command Palette
2. "Python: Select Interpreter"
3. Choose `.venv/bin/python` in python-worker directory

### GoLand: GOROOT not configured
**Solution:**
1. Settings → Go → GOROOT
2. Click "..." and select your Go installation
3. Usually `/usr/local/go` or `/usr/lib/go`

## Quick Reference

### File Structure
```
go-kafkify/
├── services/
│   ├── rest-service/
│   │   ├── go.mod          # Module definition
│   │   ├── go.sum          # Dependency checksums (generated)
│   │   └── *.go
│   ├── grpc-service/
│   │   ├── go.mod
│   │   ├── go.sum
│   │   └── *.go
│   └── python-worker/
│       ├── requirements.txt
│       ├── .venv/          # Virtual environment (local only)
│       └── main.py
```

### Commands Cheat Sheet

**Go:**
```bash
go mod download    # Download dependencies
go mod tidy        # Add missing and remove unused modules
go mod verify      # Verify dependencies
```

**Python:**
```bash
source .venv/bin/activate        # Activate venv (Linux/Mac)
.venv\Scripts\activate           # Activate venv (Windows)
pip list                         # List installed packages
pip install -r requirements.txt # Install dependencies
deactivate                       # Deactivate venv
```

## Important Notes

1. **The `.venv` directory is gitignored** - each developer creates their own
2. **The `go.sum` files are gitignored** - they're generated from `go.mod`
3. **Docker builds generate their own go.sum** during build time
4. **Production runs in containers** - local setup is IDE-only
5. **No need to install PostgreSQL, Kafka locally** - they run in Docker

## Still Having Issues?

Make sure:
- [ ] Go is installed: `go version`
- [ ] Python is installed: `python3 --version`
- [ ] Virtual environment exists: `ls services/python-worker/.venv`
- [ ] Go modules downloaded: `ls services/rest-service/go.sum`
- [ ] IDE restarted after configuration changes
- [ ] Workspace is opened at project root (`go-kafkify/`)

## Success Indicators

When properly configured, you should see:
- ✅ No red squiggly lines under imports
- ✅ Autocomplete suggestions when typing
- ✅ "Go to Definition" (F12) works
- ✅ Hover shows documentation
- ✅ Error highlights for actual code issues

---

**Remember:** The services run perfectly in Docker. This setup is only for a better development experience in your IDE! 🎯
