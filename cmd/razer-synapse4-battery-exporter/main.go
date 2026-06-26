package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"text/tabwriter"
	"time"

	"github.com/NekoYos/razer-synapse4-battery-exporter/internal/exporter"
	"github.com/NekoYos/razer-synapse4-battery-exporter/internal/synapse"
)

const appName = "Razer Synapse 4 Battery Exporter"

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	logPath := flag.String("log", "", "path to Razer Synapse systray log; empty means auto-detect latest log on every read")
	listen := flag.String("listen", ":9978", "HTTP listen address")
	once := flag.Bool("once", false, "print devices once and exit")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(versionText())
		return
	}

	if *once {
		devices, resolvedLogPath, err := readDevices(*logPath)
		if err != nil {
			exitWithError("%v", err)
		}
		printDevices(resolvedLogPath, devices)
		return
	}

	detachConsole()

	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		devices, _, err := readDevices(*logPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var body bytes.Buffer
		exporter.WriteDeviceMetrics(&body, devices)
		exporter.WriteBuildInfoMetric(&body, currentBuildInfo())
		scrapeDurationMilliseconds := float64(time.Since(startedAt).Microseconds()) / 1000
		exporter.WriteScrapeDurationMetric(&body, scrapeDurationMilliseconds)

		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		if _, err := w.Write(body.Bytes()); err != nil {
			log.Printf("write metrics response: %v", err)
		}
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, appName)
		fmt.Fprintf(w, "Version: %s\n", version)
		fmt.Fprintln(w, "Metrics: /metrics")
	})

	if *logPath == "" {
		log.Printf("reading latest matching Razer Synapse systray log on every scrape")
	} else {
		log.Printf("reading fixed log: %s", *logPath)
	}
	log.Printf("%s %s", appName, version)
	log.Printf("listening on http://%s/metrics", *listen)
	server := &http.Server{
		Addr:              *listen,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		exitWithError("listen: %v", err)
	}
}

func currentBuildInfo() exporter.BuildInfo {
	return exporter.BuildInfo{
		Version:   version,
		Commit:    commit,
		BuildDate: buildDate,
	}
}

func versionText() string {
	return fmt.Sprintf("%s %s (commit %s, built %s)", appName, version, commit, buildDate)
}

func readDevices(logPath string) ([]synapse.Device, string, error) {
	if logPath != "" {
		devices, err := readDevicesFromFile(logPath)
		return devices, logPath, err
	}

	logPaths, err := synapse.DefaultLogPaths()
	if err != nil {
		return nil, "", err
	}

	var lastErr error
	for _, candidatePath := range logPaths {
		devices, err := readDevicesFromFile(candidatePath)
		if err != nil {
			lastErr = err
			continue
		}
		if len(devices) > 0 {
			return devices, candidatePath, nil
		}
	}

	if lastErr != nil {
		return nil, "", lastErr
	}
	return nil, "", fmt.Errorf("no battery devices found in %d Razer Synapse systray logs", len(logPaths))
}

func readDevicesFromFile(logPath string) ([]synapse.Device, error) {
	file, err := os.Open(logPath)
	if err != nil {
		return nil, fmt.Errorf("open log %s: %w", logPath, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat log %s: %w", logPath, err)
	}

	devices, err := synapse.ParseLatestDevicesFromLog(file, info.Size())
	if err != nil {
		return nil, fmt.Errorf("parse log %s: %w", logPath, err)
	}
	return devices, nil
}

func printDevices(logPath string, devices []synapse.Device) {
	if len(devices) == 0 {
		fmt.Printf("No battery devices found in %s\n", logPath)
		return
	}

	fmt.Printf("Log: %s\n\n", logPath)

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "DEVICE\tCATEGORY\tSERIAL\tBATTERY\tSTATUS")
	for _, device := range devices {
		fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%d%%\t%s\n",
			device.Name,
			device.Category,
			device.SerialNumber,
			device.BatteryLevel,
			device.ChargingStatus,
		)
	}
	writer.Flush()
}

func exitWithError(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
