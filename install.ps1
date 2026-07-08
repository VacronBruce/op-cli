# op-cli installer for Windows (PowerShell)
#
# One-line install (no Git Bash / WSL needed):
#   irm https://github.com/VacronBruce/op-cli/releases/latest/download/install.ps1 | iex
#
# The repo is public, so this needs no login or token.

$ErrorActionPreference = "Stop"

$Version   = "0.23.0"
$Repo      = "VacronBruce/op-cli"
$DlUrl     = "https://github.com/$Repo/releases/latest/download"  # GitHub serves the newest release here
$Binary    = "op-windows-amd64.exe"
$OpUrl     = "https://openpr.epochbase.com"
$InstallDir = if ($env:OP_INSTALL_DIR) { $env:OP_INSTALL_DIR } else { "$env:LOCALAPPDATA\Programs\op" }

Write-Host "================================"
Write-Host "  op-cli installer (v$Version)"
Write-Host "================================"
Write-Host ""

# Step 1: Download the binary as op.exe into the install dir
Write-Host "1/4 Installing op binary..."
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
$Target = Join-Path $InstallDir "op.exe"
Write-Host "    Downloading $Binary from the latest GitHub release..."
try {
    Invoke-WebRequest -Uri "$DlUrl/$Binary" -OutFile $Target -UseBasicParsing
} catch {
    Write-Host "    Error: download failed. $_"
    exit 1
}
Write-Host "    Installed to $Target"

# Add the install dir to the user PATH (idempotent) so `op` resolves in new shells
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$InstallDir", "User")
    Write-Host "    Added $InstallDir to your user PATH (restart terminals to pick it up)."
}
# Make `op` usable in THIS session too
$env:Path = "$env:Path;$InstallDir"
Write-Host ""

# Step 2: Config
Write-Host "2/4 Config setup"
$RcPath = Join-Path $env:USERPROFILE ".oprc"
if (Test-Path $RcPath) {
    Write-Host "    Already exists at ~/.oprc, skipping."
} else {
    Write-Host ""
    Write-Host "    You need an OpenProject API key from: $OpUrl"
    Write-Host "    Go to: My Account > Access Tokens > Create new token"
    Write-Host ""
    $ApiKey = Read-Host "    Paste your API key"
    if ([string]::IsNullOrWhiteSpace($ApiKey)) {
        Write-Host "    No API key provided. Edit ~/.oprc later."
        @"
url: $OpUrl
api_key: YOUR_API_KEY_HERE
# project: app
# sprint: "App_05/19/2026"
"@ | Set-Content -Path $RcPath -Encoding UTF8
    } else {
        $DefaultProject = Read-Host "    Default project (leave empty to skip)"
        $DefaultSprint  = Read-Host "    Default sprint  (leave empty to skip)"
        $lines = @("url: $OpUrl", "api_key: $ApiKey")
        if ($DefaultProject) { $lines += "project: $DefaultProject" }
        if ($DefaultSprint)  { $lines += "sprint: `"$DefaultSprint`"" }
        $lines -join "`n" | Set-Content -Path $RcPath -Encoding UTF8
    }
    Write-Host "    Saved to ~/.oprc"
}
Write-Host ""

# Step 3: Shell completion (PowerShell profile, idempotent)
Write-Host "3/4 Shell completion"
if (-not (Test-Path $PROFILE)) {
    New-Item -ItemType File -Force -Path $PROFILE | Out-Null
}
if (Select-String -Path $PROFILE -Pattern "op completion powershell" -Quiet -ErrorAction SilentlyContinue) {
    Write-Host "    Already enabled in your PowerShell profile, skipping."
} else {
    Add-Content -Path $PROFILE -Value "`n# op-cli shell completion`nop completion powershell | Out-String | Invoke-Expression"
    Write-Host "    Enabled in $PROFILE - restart PowerShell to pick it up."
}
Write-Host ""

# Step 4: Claude Code plugin (op:)
Write-Host "4/4 Claude Code plugin (op:)"
if (Get-Command claude -ErrorAction SilentlyContinue) {
    $mpSrc = "https://github.com/$Repo.git"
    claude plugin marketplace add $mpSrc 2>$null; if ($LASTEXITCODE -ne 0) { claude plugin marketplace update op 2>$null }
    claude plugin install op@op --scope user 2>$null
    if ($LASTEXITCODE -eq 0) {
        Write-Host "    Installed the op plugin - use /op:openproject, /op:standup, /op:file-bug, /op:ticket-* ..."
    } else {
        Write-Host "    Plugin install failed. Run: claude plugin install op@op"
    }
} else {
    Write-Host "    Claude Code (claude CLI) not detected, skipping."
}
Write-Host ""

# Verify
Write-Host "--- Verifying ---"
& $Target setup
if ($LASTEXITCODE -eq 0) {
    Write-Host ""
    Write-Host "================================"
    Write-Host "  Setup complete!"
    Write-Host "================================"
    Write-Host ""
    Write-Host "  op projects       # List all projects"
    Write-Host "  op board          # Sprint board"
    Write-Host "  op my             # My items"
    Write-Host "  op show <id>      # View ticket"
    Write-Host "  op --help         # All commands"
    Write-Host ""
    Write-Host "  (Open a NEW terminal so 'op' is on your PATH.)"
} else {
    Write-Host ""
    Write-Host "Some checks failed - each [--] line above shows the fix."
    Write-Host "Re-run 'op setup' anytime to re-check."
}
