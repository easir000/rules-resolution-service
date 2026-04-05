# Rules Resolution Service

A Go-based service that resolves multi-dimensional configuration for legal workflow steps using a specificity-based override cascade.

## Quick Start

### Prerequisites
- Go 1.21+
- PowerShell (Windows) or bash (Linux/Mac)

### Run the Service

```powershell
# Build
go build -o rules-resolution.exe ./cmd/server

# Run (default port 8080)
.\rules-resolution.exe

# Or specify port
$env:PORT="8082"; .\rules-resolution.exe