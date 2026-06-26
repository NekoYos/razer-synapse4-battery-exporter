# Contributing

Thanks for taking an interest in `Razer Synapse 4 Battery Exporter`.

## Development

Requirements:

- Windows
- Go 1.26 or newer
- Razer Synapse 4 with systray logs under the current user's `%LOCALAPPDATA%`

Run tests:

```powershell
go test ./...
```

Run the exporter in console mode:

```powershell
go run ./cmd/razer-synapse4-battery-exporter
```

Print parsed devices once:

```powershell
go run ./cmd/razer-synapse4-battery-exporter -once
```

Build the background executable:

```powershell
go build -ldflags="-H windowsgui" -o bin\razer-synapse4-battery-exporter.exe ./cmd/razer-synapse4-battery-exporter
```

## Pull Requests

Please keep changes focused. Useful contributions include:

- support for additional observed `chargingStatus` values;
- parser fixtures from other Razer devices;
- better installer behavior;
- documentation improvements;
- tests for Synapse log rotation edge cases.

Avoid committing generated `.exe` files, the local `bin\` directory, or local
Synapse logs.
