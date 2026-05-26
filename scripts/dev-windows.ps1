# Superagent Base - Windows Development Script
# Usage:
#   .\scripts\dev-windows.ps1                          # Start MySQL + Redis middleware + backend
#   .\scripts\dev-windows.ps1 -Action server           # Backend only (middleware already running)
#   .\scripts\dev-windows.ps1 -Action web              # Web frontend only (backend already running)
#   .\scripts\dev-windows.ps1 -Action all              # Middleware + backend + web (three windows)
#   .\scripts\dev-windows.ps1 -Action middleware        # Docker middleware only
#   .\scripts\dev-windows.ps1 -Action down             # Stop middleware containers
#   .\scripts\dev-windows.ps1 -Action clean            # Stop + delete data
param(
    [ValidateSet("middleware", "server", "web", "down", "clean", "all")]
    [string]$Action = "all",
    [string]$EnvFile = "backend\.env.dev"
)

$ErrorActionPreference = "Stop"
$ComposeFile = "docker\docker-compose-dev.yml"
$BackendDir  = "backend"
$WebDir      = "web"

function Test-DockerRunning {
    try { docker info 2>&1 | Out-Null; return $true } catch { return $false }
}

function Start-Middleware {
    Write-Host "Starting dev middleware (MySQL + Redis)..." -ForegroundColor Cyan

    if (-not (Test-DockerRunning)) {
        Write-Host "Docker is not running. Starting Docker Desktop..." -ForegroundColor Yellow
        Start-Process "C:\Program Files\Docker\Docker\Docker Desktop.exe" -ErrorAction SilentlyContinue
        $waited = 0
        while (-not (Test-DockerRunning) -and $waited -lt 60) {
            Write-Host "." -NoNewline; Start-Sleep 2; $waited += 2
        }
        Write-Host ""
        if (-not (Test-DockerRunning)) {
            Write-Error "ERROR: Docker Desktop failed to start. Please start it manually."
        }
    }

    docker compose -f $ComposeFile up -d --wait
    Write-Host "Middleware ready." -ForegroundColor Green
}

function Start-Server {
    Write-Host "Building and running backend on :8888..." -ForegroundColor Cyan

    if (-not (Test-Path $EnvFile)) {
        Write-Error "Env file not found: $EnvFile`nCopy docker\.env.dev to backend\.env.dev and edit MYSQL_DSN / REDIS_ADDR."
    }

    Push-Location $BackendDir
    try {
        $env:APP_ENV = "dev"
        go run .
    } finally {
        Pop-Location
        Remove-Item Env:APP_ENV -ErrorAction SilentlyContinue
    }
}

function Start-Web {
    Write-Host "Starting web frontend on :3000..." -ForegroundColor Cyan

    if (-not (Test-Path $WebDir)) {
        Write-Error "Directory '$WebDir' not found. Run this script from the project root."
    }

    Push-Location $WebDir
    try {
        if (-not (Test-Path "node_modules")) {
            Write-Host "Installing npm dependencies..." -ForegroundColor Yellow
            npm install
        }
        npm run dev
    } finally {
        Pop-Location
    }
}

function Stop-Middleware {
    Write-Host "Stopping dev environment..." -ForegroundColor Cyan
    docker compose -f $ComposeFile down
}

function Remove-DevData {
    Stop-Middleware
    $dataDir = "docker\data\mysql-dev"
    if (Test-Path $dataDir) {
        Remove-Item -Recurse -Force $dataDir
        Write-Host "Cleaned dev data." -ForegroundColor Green
    }
}

switch ($Action) {
    "middleware" { Start-Middleware }
    "server"     { Start-Server }
    "web"        { Start-Web }
    "down"       { Stop-Middleware }
    "clean"      { Remove-DevData }
    "all" {
        Start-Middleware
        # Open backend in a new PowerShell window, web in current terminal
        $backendCmd = "cd '$PWD'; .\scripts\dev-windows.ps1 -Action server -EnvFile '$EnvFile'"
        Start-Process powershell -ArgumentList "-NoExit", "-Command", $backendCmd
        Write-Host "Backend started in a new window. Starting web frontend here..." -ForegroundColor Green
        Start-Web
    }
}
