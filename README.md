# Rules Resolution Service

A Go-based service that resolves multi-dimensional configuration for foreclosure legal workflow steps using a specificity-based override cascade.

> **Pearson Specter Litt** — Senior Backend Engineer Take-Home Assignment

---

## 📋 Overview

Given a case context like:
json
{"state": "FL", "client": "Chase", "investor": "FHA", "caseType": "FC-Judicial"}


The service determines exact deadlines, documents, fees, and templates by applying override records ranked by **specificity** (like CSS specificity for business rules).

### Key Features
- ✅ Specificity-based resolution (1-4 dimension cascade)
- ✅ Effective date filtering with `asOfDate` parameter
- ✅ Debuggable explain traces with SELECTED/SHADOWED outcomes
- ✅ Full CRUD API for override management
- ✅ Conflict detection for same-specificity overlapping rules
- ✅ Audit trail for all override changes
- ✅ Windows PowerShell + Linux/macOS compatible

---

## 🚀 Quick Start

### Prerequisites
- Go 1.21+ ([Download](https://go.dev/dl/))
- PowerShell 5.1+ (Windows) or bash (Linux/macOS)
- Optional: Docker & Docker Compose (for PostgreSQL)

### Option A: Run with In-Memory Storage (Fastest)

powershell
# 1. Navigate to project root
cd C:\Users\maruf\OneDrive\Desktop\rules-resolution-service

# 2. Build the service
go build -o rules-resolution.exe ./cmd/server

# 3. Start the server (default port 8080)
.\rules-resolution.exe

# Or specify a custom port
$env:PORT = "8099"
.\rules-resolution.exe


> ⚠️ Keep this PowerShell window open — it's running the server.  
> Open a **NEW** PowerShell window for testing commands below.

### Option B: Run with PostgreSQL (Production Mode)

powershell
# 1. Start PostgreSQL via Docker Compose
docker-compose up -d postgres

# Wait for database to be ready (~10 seconds)
Start-Sleep -Seconds 10

# 2. Set environment variables
$env:DATABASE_URL = "postgres://spine:spine@localhost:5432/rules_resolution?sslmode=disable"
$env:RUN_MIGRATIONS = "true"
$env:SEED_DATA = "true"
$env:PORT = "8099"

# 3. Build and run
go build -o rules-resolution.exe ./cmd/server
.\rules-resolution.exe


> The server will automatically:
> 1. Apply database migrations from `internal/database/schema.sql`
> 2. Load seed data from `sr_backend_assignment_data/`
> 3. Start listening on the specified port

---

## 🧪 Testing the API

### Health Check
powershell
Invoke-RestMethod -Uri "http://localhost:8099/api/health"

Expected: `{"status":"ok"}`

---

### Resolve Configuration (Core Endpoint)
powershell
# Build request body
$body = @{
    state = "FL"
    client = "Chase"
    investor = "FHA"
    caseType = "FC-Judicial"
} | ConvertTo-Json -Compress

# Send POST request
$response = Invoke-RestMethod -Uri "http://localhost:8099/api/resolve" `
    -Method Post `
    -ContentType "application/json" `
    -Body $body

# View specific resolved trait
Write-Host "file-complaint.slaHours = $($response.steps.'file-complaint'.slaHours.value)"
Write-Host "Source: $($response.steps.'file-complaint'.slaHours.source)"
Write-Host "Override ID: $($response.steps.'file-complaint'.slaHours.overrideId)"


Expected output:

file-complaint.slaHours = 168
Source: override
Override ID: ovr-034


---

### Explain Endpoint (Debugging)
powershell
$body = @{
    stepKey = "file-complaint"
    traitKey = "slaHours"
    state = "FL"
    client = "Chase"
    investor = "FHA"
    caseType = "FC-Judicial"
} | ConvertTo-Json -Compress

$response = Invoke-RestMethod -Uri "http://localhost:8099/api/resolve/explain" `
    -Method Post `
    -ContentType "application/json" `
    -Body $body

# View full response
$response | ConvertTo-Json -Depth 10


Expected output shows:
- `resolvedValue: 168` from `ovr-034` (specificity 3)
- Candidate list with `SELECTED`/`SHADOWED` outcomes
- Full selector details for each candidate

---

### Override CRUD API

#### List All Overrides
powershell
# Get all overrides
$overrides = Invoke-RestMethod -Uri "http://localhost:8099/api/overrides"
Write-Host "Total overrides: $($overrides.count)"

# Filter by state
$flOverrides = Invoke-RestMethod -Uri "http://localhost:8099/api/overrides?state=FL"
Write-Host "Florida overrides: $($flOverrides.count)"

# Filter by status
$active = Invoke-RestMethod -Uri "http://localhost:8099/api/overrides?status=active"
Write-Host "Active overrides: $($active.count)"


#### Get Single Override
powershell
$ovr = Invoke-RestMethod -Uri "http://localhost:8099/api/overrides/ovr-034"
Write-Host "ID: $($ovr.id), Value: $($ovr.value), Specificity: $($ovr.specificity)"


#### Create New Override
powershell
# Use raw JSON for reliable parsing
$jsonBody = @'
{
  "id": "ovr-test-001",
  "stepKey": "file-complaint",
  "traitKey": "slaHours",
  "selector": {
    "state": "CA",
    "client": "TestClient"
  },
  "value": 300,
  "effectiveDate": "2025-01-01",
  "status": "draft",
  "description": "Test override for California",
  "createdBy": "test@example.com"
}
'@

$response = Invoke-RestMethod -Uri "http://localhost:8099/api/overrides" `
    -Method Post `
    -ContentType "application/json" `
    -Body $jsonBody `
    -TimeoutSec 5

Write-Host "✅ Created: $($response.id), Specificity=$($response.specificity)"


#### Update Override
powershell
$updateBody = @'
{
  "id": "ovr-test-001",
  "stepKey": "file-complaint",
  "traitKey": "slaHours",
  "selector": {"state": "CA", "client": "TestClient"},
  "value": 350,
  "effectiveDate": "2025-01-01",
  "status": "active",
  "description": "Updated test value",
  "createdBy": "test@example.com"
}
'@

$response = Invoke-RestMethod -Uri "http://localhost:8099/api/overrides/ovr-test-001" `
    -Method Put `
    -ContentType "application/json" `
    -Body $updateBody

Write-Host "✅ Updated value: $($response.value)"


#### Update Status (Activate/Archive)
powershell
$statusUpdate = '{"status": "archived"}'

$response = Invoke-RestMethod -Uri "http://localhost:8099/api/overrides/ovr-test-001/status" `
    -Method Patch `
    -ContentType "application/json" `
    -Body $statusUpdate

Write-Host "✅ Status: $($response.status)"


#### Check for Conflicts
powershell
$conflicts = Invoke-RestMethod -Uri "http://localhost:8099/api/overrides/conflicts"
Write-Host "Conflicts found: $($conflicts.conflicts.Count)"
$conflicts | ConvertTo-Json -Depth 5


Expected: `{"conflicts":[]}` (seed data is conflict-free)

#### Get Audit History
powershell
$history = Invoke-RestMethod -Uri "http://localhost:8099/api/overrides/ovr-034/history"
Write-Host "History entries: $($history.history.Count)"
$history | ConvertTo-Json -Depth 5


---

## 🧪 Run All Test Scenarios

Save as `test-runner.ps1` and execute:

powershell
# test-runner.ps1
$port = 8099
$baseUrl = "http://localhost:$port/api/resolve"
$passed = 0
$failed = 0

$scenarios = @(
    @{name="FL baseline"; context=@{state="FL"; client="Nationstar"; investor="FannieMae"; caseType="FC-Judicial"}; checks=@(@{step="title-search"; trait="slaHours"; expectedValue=480; expectedSource="override"; expectedId="ovr-048"})},
    @{name="FL Chase"; context=@{state="FL"; client="Chase"; investor="FreddieMac"; caseType="FC-Judicial"}; checks=@(@{step="file-complaint"; trait="slaHours"; expectedValue=240; expectedSource="override"; expectedId="ovr-020"})},
    @{name="FL Chase FHA"; context=@{state="FL"; client="Chase"; investor="FHA"; caseType="FC-Judicial"}; checks=@(@{step="file-complaint"; trait="slaHours"; expectedValue=168; expectedSource="override"; expectedId="ovr-034"})},
    @{name="asOfDate"; context=@{state="FL"; client="Chase"; investor="FHA"; caseType="FC-Judicial"; asOfDate="2025-07-01"}; checks=@(@{step="file-complaint"; trait="slaHours"; expectedValue=240; expectedSource="override"; expectedId="ovr-020"})}
)

foreach ($s in $scenarios) {
    Write-Host "[TEST] $($s.name)"
    $body = $s.context | ConvertTo-Json -Compress
    try {
        $r = Invoke-RestMethod -Uri $baseUrl -Method Post -ContentType "application/json" -Body $body -TimeoutSec 10
        foreach ($c in $s.checks) {
            $t = $r.steps.$($c.step).$($c.trait)
            if ($c.expectedValue -eq $t.value -and $c.expectedSource -eq $t.source -and $c.expectedId -eq $t.overrideId) {
                Write-Host "  [PASS] $($c.step).$($c.trait)" -ForegroundColor Green; $passed++
            } else { Write-Host "  [FAIL] $($c.step).$($c.trait)" -ForegroundColor Red; $failed++ }
        }
    } catch { Write-Host "  [ERROR] $($_.Exception.Message)" -ForegroundColor Red; $failed += $s.checks.Count }
}
Write-Host "`n[SUMMARY] Passed: $passed, Failed: $failed" -ForegroundColor Magenta
if ($failed -eq 0) { Write-Host "[RESULT] All tests passed!" -ForegroundColor Green }


Run it:
powershell
.\test-runner.ps1


Expected: `Passed: 4, Failed: 0`

---

## 🗄️ Database Setup (PostgreSQL)

### Using Docker Compose (Recommended)
powershell
# Start PostgreSQL
docker-compose up -d postgres

# Verify it's running
docker-compose ps

# Connect manually (optional)
docker-compose exec postgres psql -U spine -d rules_resolution


### Manual PostgreSQL Setup
powershell
# Create database
psql -U postgres -c "CREATE DATABASE rules_resolution;"
psql -U postgres -c "CREATE USER spine WITH PASSWORD 'spine';"
psql -U postgres -c "GRANT ALL PRIVILEGES ON DATABASE rules_resolution TO spine;"

# Apply schema
psql -U spine -d rules_resolution -f internal/database/schema.sql

# Load seed data (optional)
# See scripts/seed.go for JSON-based seeding


### Environment Variables
| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP port for the service |
| `DATABASE_URL` | `postgres://spine:spine@localhost:5432/rules_resolution?sslmode=disable` | PostgreSQL connection string |
| `RUN_MIGRATIONS` | `false` | Apply schema migrations on startup |
| `SEED_DATA` | `false` | Load seed data from JSON files on startup |

---

## 📡 API Reference

### POST /api/resolve
Resolve full configuration for a case context.

**Request:**
json
{
  "state": "FL",
  "client": "Chase",
  "investor": "FHA",
  "caseType": "FC-Judicial",
  "asOfDate": "2025-07-01"
}


**Response:**
json
{
  "context": {"state": "FL", "client": "Chase", "investor": "FHA", "caseType": "FC-Judicial"},
  "resolvedAt": "2026-04-05T14:30:00Z",
  "steps": {
    "file-complaint": {
      "slaHours": {
        "value": 240,
        "source": "override",
        "overrideId": "ovr-020",
        "explanation": "FL+Chase override (specificity 2)"
      }
    }
  }
}


### POST /api/resolve/explain
Get detailed resolution trace for debugging.

**Response includes:**
- `resolvedValue`: The final resolved value
- `resolvedFrom`: Details of the winning override
- `candidates`: Array of all matching overrides with `SELECTED`/`SHADOWED` outcomes

### GET /api/overrides
List overrides with optional filters: `stepKey`, `traitKey`, `state`, `client`, `investor`, `caseType`, `status`.

### POST /api/overrides
Create a new override. `specificity` is auto-computed from the selector.

### PUT /api/overrides/{id}
Update an existing override.

### PATCH /api/overrides/{id}/status
Change status: `draft` → `active` → `archived`.

### GET /api/overrides/conflicts
Identify conflicting override pairs (same step/trait, same specificity, overlapping dates, compatible selectors).

### GET /api/overrides/{id}/history
Return audit trail for an override (who changed what, when, before/after).

### GET /api/health
Health check endpoint.

---

## 🏗️ Project Structure


rules-resolution-service/
├── cmd/server/
│   └── main.go              # Entry point with all route handlers
├── internal/
│   ├── config/
│   │   ├── config.go        # Application configuration
│   │   └── database.go      # Database configuration
│   ├── database/
│   │   ├── migrate.go       # Migration runner
│   │   ├── schema.sql       # PostgreSQL schema definition
│   │   └── seed_data.sql    # Seed data loader
│   ├── domain/
│   │   └── models.go        # Domain types (Override, Selector, etc.)
│   ├── handler/
│   │   └── override_handler.go  # CRUD endpoint handlers
│   ├── repository/
│   │   ├── interfaces.go    # Repository interfaces
│   │   └── override_repository.go  # PostgreSQL queries
│   └── service/
│       ├── resolution_service.go  # Core resolution algorithm
│       ├── explain_service.go     # Explain endpoint logic
│       └── conflict_service.go    # Conflict detection
├── pkg/util/
│   └── json.go              # JSON helpers
├── scripts/
│   └── seed.go              # JSON seed data loader
├── test/
│   └── integration/
│       └── resolution_test.go   # Integration tests
├── sr_backend_assignment_data/
│   ├── steps.json           # Canonical workflow steps
│   ├── defaults.json        # Default values (specificity 0)
│   ├── overrides.json       # 49 seed override records
│   └── test_scenarios.json  # 12 acceptance test scenarios
├── docker-compose.yml       # PostgreSQL container config
├── Dockerfile               # Service container config
├── go.mod                   # Go module definition
├── README.md                # This file
└── APPROACH.md              # Design decisions & trade-offs


---

## 🔧 Troubleshooting

### Server Won't Start
powershell
# Check if port is already in use
Get-NetTCPConnection -LocalPort 8099 -ErrorAction SilentlyContinue

# Kill any existing server processes
Get-Process -Name "rules-resolution" -ErrorAction SilentlyContinue | Stop-Process -Force

# Rebuild from scratch
go clean -cache -modcache
go mod tidy
go build -o rules-resolution.exe ./cmd/server


### 404 Errors on CRUD Endpoints
powershell
# Verify routes are registered
Select-String -Path "cmd/server/main.go" -Pattern "overrideHandler\.RegisterRoutes"
# Should show exactly ONE match

# Check server logs in the server window for startup messages
# Look for: "Server starting on :8099" with no errors after


### JSON Decode Errors ("invalid body")
powershell
# Use raw JSON strings instead of ConvertTo-Json for POST/PUT
$json = @'{"id":"test","stepKey":"file-complaint","traitKey":"slaHours","selector":{},"value":100,"effectiveDate":"2025-01-01","status":"draft","createdBy":"test"}'@

# Ensure Content-Type header is set
Invoke-RestMethod -Uri "http://localhost:8099/api/overrides" `
    -Method Post `
    -ContentType "application/json" `
    -Body $json


### PostgreSQL Connection Issues
powershell
# Verify Docker container is running
docker-compose ps

# Check database connectivity
$env:PGPASSWORD = "spine"
psql -h localhost -U spine -d rules_resolution -c "SELECT 1;"

# Reset database (development only)
docker-compose down -v
docker-compose up -d postgres
Start-Sleep -Seconds 10


---

## 📦 Building for Production

powershell
# Build optimized binary
go build -ldflags="-s -w" -o rules-resolution ./cmd/server

# Create minimal Docker image
docker build -t rules-resolution:latest .

# Run with Docker
docker run -p 8080:8080 `
  -e DATABASE_URL="postgres://spine:spine@host.docker.internal:5432/rules_resolution?sslmode=disable" `
  rules-resolution:latest


---

## 📄 License

Internal use — Pearson Specter Litt

---

## 🙋 Support

For issues or questions:
1. Check server logs in the running PowerShell window
2. Verify environment variables are set correctly
3. Ensure PostgreSQL is running if using database mode
4. Review `APPROACH.md` for design decisions and trade-offs

---

> **Note**: The core resolution engine is production-ready and passes all 12 test scenarios. PostgreSQL integration is designed and ready to enable via environment variables. CRUD endpoints use in-memory storage by default for testing simplicity.