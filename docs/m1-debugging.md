# Debugging on Mac M1 Architecture

This guide provides specific instructions for setting up and debugging the JWT blacklisting application on Mac machines with Apple Silicon (M1/M2/M3) processors.

## Prerequisites

1. **Install ARM64 version of Go**:
   ```bash
   # Check your current Go architecture
   go version
   # Should show "darwin/arm64" not "darwin/amd64"
   ```

   If your Go shows "darwin/amd64", you need to install the ARM64 version:
   - Download the ARM64 version from [go.dev/dl](https://go.dev/dl/)
   - Look for "go1.xx.x.darwin-arm64.pkg"
   - Install this package

2. **Verify correct architecture**:
   ```bash
   go version
   # Should now show "darwin/arm64"
   ```

3. **Install VS Code ARM64 version**:
   - Download VS Code for Apple Silicon from [code.visualstudio.com](https://code.visualstudio.com/)
   - Ensure you select the Apple Silicon version, not Universal or Intel

## VS Code Setup for Debugging

1. **Create a `.vscode` directory and `launch.json`**:
   ```bash
   mkdir -p .vscode
   touch .vscode/launch.json
   ```

2. **Add this M1-compatible configuration**:
   ```json
   {
       "version": "0.2.0",
       "configurations": [
           {
               "name": "Launch JWT Server (M1)",
               "type": "go",
               "request": "launch",
               "mode": "auto",
               "program": "${workspaceFolder}/cmd/server/main.go",
               "env": {
                   "JWT_SECRET": "debug-secret-key",
                   "ACCESS_TOKEN_EXPIRATION": "15m",
                   "REFRESH_TOKEN_EXPIRATION": "7d",
                   "REDIS_ADDR": "localhost:6379",
                   "GOARCH": "arm64",
                   "GOOS": "darwin"
               },
               "args": []
           },
           {
               "name": "Debug Tests (M1)",
               "type": "go",
               "request": "launch",
               "mode": "test",
               "program": "${fileDirname}",
               "env": {
                   "GOARCH": "arm64",
                   "GOOS": "darwin"
               },
               "args": []
           },
           {
               "name": "Debug Multi-Device Tests (M1)",
               "type": "go",
               "request": "launch",
               "mode": "test",
               "program": "${workspaceFolder}/internal/auth",
               "env": {
                   "GOARCH": "arm64",
                   "GOOS": "darwin"
               },
               "args": ["-test.run", "TestMultiDeviceLogout"]
           }
       ]
   }
   ```

3. **Install Go tools for ARM64**:
   - In VS Code, press `Cmd+Shift+P`
   - Type "Go: Install/Update Tools"
   - Select all tools (especially Delve)
   - This will install ARM64 native versions
