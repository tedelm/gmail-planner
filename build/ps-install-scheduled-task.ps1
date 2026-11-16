param(
    [string]$appName = "gmail-planner",
    [parameter(Mandatory = $true)][string]$version = "1.0.0",
    [string]$rootPath = "C:\0_Priv\github.com\gmail-planner",
    [string]$taskName = "gmail-planner-weekly-digest",
    [string]$dayOfWeek = "Monday",
    [string]$at = "09:05",
    [string]$arguments = "-digest -ephemeral-auth",
    [switch]$Uninstall
)

$ErrorActionPreference = "Stop"

$exePath = Join-Path $rootPath "build\bin\${appName}_${version}.exe"
$workDir = Join-Path $rootPath "src"
$envFile = Join-Path $workDir ".env"

$existing = Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue

if ($Uninstall) {
    if (-not $existing) {
        Write-Host "Task '$taskName' is not registered; nothing to uninstall."
        exit 0
    }
    Unregister-ScheduledTask -TaskName $taskName -Confirm:$false
    Write-Host "Unregistered scheduled task '$taskName'."
    exit 0
}

if (-not (Test-Path -LiteralPath $exePath)) {
    throw "Executable not found: $exePath. Build first with .\build\ps-build.ps1 -version $version"
}
if (-not (Test-Path -LiteralPath $envFile)) {
    throw ".env not found: $envFile (copy from src\.env-example and fill in values)"
}

$action = New-ScheduledTaskAction -Execute $exePath -Argument $arguments -WorkingDirectory $workDir
$trigger = New-ScheduledTaskTrigger -Weekly -DaysOfWeek $dayOfWeek -At $at
# Interactive logon only so the Google browser window is visible.
$principal = New-ScheduledTaskPrincipal -UserId $env:USERNAME -LogonType Interactive -RunLevel Limited
$settings = New-ScheduledTaskSettingsSet `
    -AllowStartIfOnBatteries `
    -DontStopIfGoingOnBatteries `
    -StartWhenAvailable `
    -ExecutionTimeLimit (New-TimeSpan -Hours 2)

if ($existing) {
    Set-ScheduledTask -TaskName $taskName -Action $action -Trigger $trigger -Principal $principal -Settings $settings | Out-Null
    Write-Host "Updated scheduled task '$taskName'."
}
else {
    Register-ScheduledTask -TaskName $taskName -Action $action -Trigger $trigger -Principal $principal -Settings $settings | Out-Null
    Write-Host "Registered scheduled task '$taskName'."
}

Write-Host "  Exe:       $exePath"
Write-Host "  Arguments: $arguments"
Write-Host "  Start in:  $workDir"
Write-Host "  Schedule:  $dayOfWeek at $at (run only when user is logged on)"
