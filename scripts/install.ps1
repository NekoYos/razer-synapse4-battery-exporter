param(
    [ValidateSet("", "install", "uninstall")]
    [string]$Action = "",
    [string]$TaskName = "Razer Synapse 4 Battery Exporter",
    [string]$Listen = ":9978",
    [int]$DelaySeconds = 30,
    [string]$InstallDir = "$env:ProgramFiles\razer-synapse4-battery-exporter",
    [switch]$NoPause
)

$ErrorActionPreference = "Stop"
$LogPath = Join-Path $env:TEMP "razer-synapse4-battery-exporter-install.log"

trap {
    $message = "$(Get-Date -Format o) ERROR: $($_ | Out-String)"
    Add-Content -Path $LogPath -Value $message
    Write-Error $_
    if ((Test-Administrator) -and (-not $NoPause)) {
        Write-Host "Error details written to: $LogPath"
        Wait-ForKey
    }
    exit 1
}

function Test-Administrator {
    $identity = [System.Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = [System.Security.Principal.WindowsPrincipal]::new($identity)
    return $principal.IsInRole([System.Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Quote-Argument {
    param([string]$Value)
    return '"' + ($Value -replace '"', '\"') + '"'
}

function Restart-AsAdministrator {
    param([string]$SelectedAction)

    Remove-Item -LiteralPath $LogPath -Force -ErrorAction SilentlyContinue

    $arguments = @(
        "-NoProfile",
        "-ExecutionPolicy", "Bypass",
        "-File", (Quote-Argument $PSCommandPath),
        "-Action", $SelectedAction,
        "-TaskName", (Quote-Argument $TaskName),
        "-Listen", (Quote-Argument $Listen),
        "-DelaySeconds", $DelaySeconds,
        "-InstallDir", (Quote-Argument $InstallDir),
        "-NoPause"
    )

    $process = Start-Process -FilePath "powershell.exe" -ArgumentList $arguments -Verb RunAs -Wait -PassThru
    $exitCode = $process.ExitCode

    Write-Host ""
    Write-Host "Elevated installer process finished with exit code $exitCode."
    if (Test-Path -LiteralPath $LogPath) {
        Write-Host "Installer log: $LogPath"
    } else {
        Write-Host "Installer log: not created"
    }
    Write-Host ""
    Show-Status

    if ($exitCode -ne 0) {
        if (Test-Path -LiteralPath $LogPath) {
            Write-Host "Elevated installer failed. See log: $LogPath"
        } else {
            Write-Host "Elevated installer failed, but no log file was created."
        }
        exit $exitCode
    }
}

function Wait-ForKey {
    Write-Host ""
    Write-Host "Press any key to continue..."
    $null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
}

function Get-InstalledExePath {
    return Join-Path $InstallDir "razer-synapse4-battery-exporter.exe"
}

function Get-ProjectRoot {
    return Split-Path -Parent $PSScriptRoot
}

function Get-SourceExePath {
    return Join-Path (Get-ProjectRoot) "bin\razer-synapse4-battery-exporter.exe"
}

function Build-SourceExecutable {
    $projectRoot = Get-ProjectRoot
    $sourceExePath = Get-SourceExePath
    $sourceExeDir = Split-Path -Parent $sourceExePath

    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        throw "Go is required to build the exporter, but 'go' was not found in PATH."
    }

    New-Item -ItemType Directory -Path $sourceExeDir -Force | Out-Null

    Write-Host "Building exporter executable..."
    Write-Host "Output: $sourceExePath"

    Push-Location $projectRoot
    try {
        & go build -o $sourceExePath ./cmd/razer-synapse4-battery-exporter
        if ($LASTEXITCODE -ne 0) {
            throw "go build failed with exit code $LASTEXITCODE"
        }
    } finally {
        Pop-Location
    }
}

function Ensure-SourceExecutable {
    $sourceExePath = Get-SourceExePath
    if (Test-Path -LiteralPath $sourceExePath) {
        return
    }

    Write-Host "Source executable not found at $sourceExePath."
    Build-SourceExecutable
}

function Get-MetricsUrl {
    $metricsAddress = $Listen
    if ($metricsAddress.StartsWith(":")) {
        $metricsAddress = "localhost$metricsAddress"
    }
    return "http://$metricsAddress/metrics"
}

function Test-MetricsEndpoint {
    try {
        $response = Invoke-WebRequest -UseBasicParsing -Uri (Get-MetricsUrl) -TimeoutSec 2
        return ($response.StatusCode -ge 200 -and $response.StatusCode -lt 300)
    } catch {
        return $false
    }
}

function Show-Status {
    $task = Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
    $installedExePath = Get-InstalledExePath
    $exeInstalled = Test-Path -LiteralPath $installedExePath
    $metricsAvailable = Test-MetricsEndpoint

    Write-Host "Razer Synapse 4 Battery Exporter"
    Write-Host ""

    if ($task) {
        Write-Host "Task: installed ($($task.State))"
        Write-Host "Task name: $TaskName"
    } else {
        Write-Host "Task: not installed"
    }

    if ($exeInstalled) {
        Write-Host "Executable: installed at $installedExePath"
    } else {
        Write-Host "Executable: not installed at $installedExePath"
    }

    if ($metricsAvailable) {
        Write-Host "Metrics: available at $(Get-MetricsUrl)"
    } else {
        Write-Host "Metrics: not responding at $(Get-MetricsUrl)"
    }

    Write-Host ""
}

function Select-Action {
    Show-Status
    Write-Host "1. Install or update user logon task"
    Write-Host "2. Uninstall user logon task"
    $choice = Read-Host "Select action"

    switch ($choice) {
        "1" { return "install" }
        "2" { return "uninstall" }
        default { throw "Unsupported selection: $choice" }
    }
}

function Stop-ExistingTask {
    $existingTask = Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
    if ($existingTask -and $existingTask.State -eq "Running") {
        Write-Host "Stopping existing scheduled task '$TaskName'..."
        Stop-ScheduledTask -TaskName $TaskName
        $deadline = (Get-Date).AddSeconds(15)
        do {
            Start-Sleep -Milliseconds 300
            $existingTask = Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
        } while ($existingTask -and $existingTask.State -eq "Running" -and (Get-Date) -lt $deadline)

        if ($existingTask -and $existingTask.State -eq "Running") {
            throw "Scheduled task '$TaskName' did not stop within 15 seconds."
        }
    }
}

function Stop-OrphanExporterProcesses {
    $task = Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
    if ($task -and $task.State -eq "Running") {
        return
    }

    $processes = @(Get-ExporterProcesses)
    if ($processes.Count -eq 0) {
        return
    }

    Write-Host "Stopping exporter process not managed by a running scheduled task..."
    Stop-ExporterProcesses
}

function Get-ExporterProcesses {
    $installedExePath = Get-InstalledExePath
    $processName = [System.IO.Path]::GetFileNameWithoutExtension($installedExePath)

    return Get-Process -Name $processName -ErrorAction SilentlyContinue | Where-Object {
        try {
            $_.Path -eq $installedExePath
        } catch {
            $false
        }
    }
}

function Test-FileLocked {
    param([string]$Path)

    if (-not (Test-Path -LiteralPath $Path)) {
        return $false
    }

    try {
        $stream = [System.IO.File]::Open(
            $Path,
            [System.IO.FileMode]::Open,
            [System.IO.FileAccess]::ReadWrite,
            [System.IO.FileShare]::None
        )
        $stream.Close()
        return $false
    } catch {
        return $true
    }
}

function Wait-ForInstalledExeRelease {
    param([int]$TimeoutSeconds = 15)

    $installedExePath = Get-InstalledExePath
    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    do {
        if (-not (Test-FileLocked -Path $installedExePath)) {
            return $true
        }
        Start-Sleep -Milliseconds 300
    } while ((Get-Date) -lt $deadline)

    return $false
}

function Stop-ExporterProcesses {
    $processes = @(Get-ExporterProcesses)
    if ($processes.Count -eq 0) {
        return
    }

    Write-Host "Stopping exporter process still holding the installed executable..."
    foreach ($process in $processes) {
        Stop-Process -Id $process.Id -Force
    }
}

function Wait-ForTaskState {
    param(
        [string]$ExpectedState,
        [int]$TimeoutSeconds = 15
    )

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    do {
        Start-Sleep -Milliseconds 300
        $task = Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
        if ($task -and $task.State.ToString() -eq $ExpectedState) {
            return $true
        }
    } while ((Get-Date) -lt $deadline)

    return $false
}

function Wait-ForMetricsEndpoint {
    param([int]$TimeoutSeconds = 15)

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    do {
        if (Test-MetricsEndpoint) {
            return $true
        }
        Start-Sleep -Milliseconds 500
    } while ((Get-Date) -lt $deadline)

    return $false
}

function Wait-ForMetricsEndpointStop {
    param([int]$TimeoutSeconds = 15)

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    do {
        if (-not (Test-MetricsEndpoint)) {
            return $true
        }
        Start-Sleep -Milliseconds 500
    } while ((Get-Date) -lt $deadline)

    return $false
}

function Install-ExporterTask {
    $sourceExePath = Get-SourceExePath
    $installedExePath = Get-InstalledExePath

    Ensure-SourceExecutable

    Stop-ExistingTask
    Stop-OrphanExporterProcesses

    if (-not (Wait-ForInstalledExeRelease -TimeoutSeconds 15)) {
        Stop-ExporterProcesses
        if (-not (Wait-ForInstalledExeRelease -TimeoutSeconds 10)) {
            throw "Installed executable is still locked and cannot be updated: $installedExePath"
        }
    }

    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Copy-Item -LiteralPath $sourceExePath -Destination $installedExePath -Force

    $currentUser = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name
    $arguments = "-listen `"$Listen`""

    $taskAction = New-ScheduledTaskAction `
        -Execute $installedExePath `
        -Argument $arguments `
        -WorkingDirectory $InstallDir

    $trigger = New-ScheduledTaskTrigger -AtLogOn -User $currentUser
    $trigger.Delay = "PT$($DelaySeconds)S"

    $principal = New-ScheduledTaskPrincipal `
        -UserId $currentUser `
        -LogonType Interactive `
        -RunLevel Limited

    $settings = New-ScheduledTaskSettingsSet `
        -AllowStartIfOnBatteries `
        -DontStopIfGoingOnBatteries `
        -ExecutionTimeLimit (New-TimeSpan -Seconds 0) `
        -Hidden `
        -MultipleInstances IgnoreNew `
        -RestartCount 3 `
        -RestartInterval (New-TimeSpan -Minutes 1) `
        -StartWhenAvailable

    $task = New-ScheduledTask `
        -Action $taskAction `
        -Trigger $trigger `
        -Principal $principal `
        -Settings $settings `
        -Description "Prometheus exporter for Razer Synapse 4 battery levels."

    Register-ScheduledTask `
        -TaskName $TaskName `
        -InputObject $task `
        -Force | Out-Null

    Write-Host "Starting scheduled task '$TaskName'..."
    Start-ScheduledTask -TaskName $TaskName

    if (-not (Wait-ForTaskState -ExpectedState "Running" -TimeoutSeconds 15)) {
        $task = Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
        $state = if ($task) { $task.State } else { "missing" }
        $taskInfo = Get-ScheduledTaskInfo -TaskName $TaskName -ErrorAction SilentlyContinue
        $lastTaskResult = if ($taskInfo) { $taskInfo.LastTaskResult } else { "unknown" }
        throw "Scheduled task '$TaskName' did not enter Running state after start. Current state: $state. LastTaskResult: $lastTaskResult"
    }

    if (-not (Wait-ForMetricsEndpoint -TimeoutSeconds 15)) {
        throw "Scheduled task '$TaskName' is running, but metrics did not respond at $(Get-MetricsUrl)."
    }

    Write-Host "Installed and started scheduled task '$TaskName' for $currentUser."
    Write-Host "Executable: $installedExePath"
    Write-Host "Metrics: $(Get-MetricsUrl)"
}

function Uninstall-ExporterTask {
    $task = Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
    if ($task) {
        if ($task.State -eq "Running") {
            Write-Host "Stopping scheduled task '$TaskName'..."
            Stop-ScheduledTask -TaskName $TaskName
            if (-not (Wait-ForTaskState -ExpectedState "Ready" -TimeoutSeconds 15)) {
                Write-Warning "Scheduled task '$TaskName' did not enter Ready state before unregistering."
            }
        }

        Unregister-ScheduledTask -TaskName $TaskName -Confirm:$false
    } else {
        Write-Host "Scheduled task '$TaskName' is not installed."
    }

    $installedExePath = Get-InstalledExePath
    if (Test-Path -LiteralPath $installedExePath) {
        if (-not (Wait-ForInstalledExeRelease -TimeoutSeconds 15)) {
            Stop-ExporterProcesses
            if (-not (Wait-ForInstalledExeRelease -TimeoutSeconds 10)) {
                throw "Installed executable is still locked and cannot be removed: $installedExePath"
            }
        }

        Remove-Item -LiteralPath $installedExePath -Force
    }

    if (-not (Wait-ForMetricsEndpointStop -TimeoutSeconds 10)) {
        Stop-ExporterProcesses
        if (-not (Wait-ForMetricsEndpointStop -TimeoutSeconds 5)) {
            throw "Metrics endpoint is still responding after uninstall: $(Get-MetricsUrl)"
        }
    }

    if (Test-Path -LiteralPath $InstallDir) {
        $remainingItems = Get-ChildItem -LiteralPath $InstallDir -Force
        if ($remainingItems.Count -eq 0) {
            Remove-Item -LiteralPath $InstallDir -Force
        } else {
            Write-Warning "Install directory is not empty, leaving it in place: $InstallDir"
        }
    }

    Write-Host "Uninstalled scheduled task '$TaskName'."
}

if ($Action -eq "") {
    $Action = Select-Action
}

if (-not (Test-Administrator)) {
    Write-Host "Administrator rights are required to modify Program Files and Task Scheduler."
    Write-Host "Requesting elevation..."
    Restart-AsAdministrator -SelectedAction $Action
    exit 0
}

switch ($Action) {
    "install" { Install-ExporterTask }
    "uninstall" { Uninstall-ExporterTask }
}

if ((Test-Administrator) -and (-not $NoPause)) {
    Wait-ForKey
}
