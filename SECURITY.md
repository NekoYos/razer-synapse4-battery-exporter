# Security Policy

## Reporting Issues

Please report issues and propose fixes through public GitHub issues and pull
requests.

Do not publish raw Synapse logs without reviewing and redacting them first. Logs
may contain serial numbers, local file paths, profile names, or other local
machine details.

## Data Access

The exporter reads only local Razer Synapse log files from the current user's
profile and exposes battery metrics over HTTP. By default it listens on all
interfaces at `:9978`.

If you do not want the exporter reachable from other machines, run it with:

```powershell
bin\razer-synapse4-battery-exporter.exe -listen 127.0.0.1:9978
```
