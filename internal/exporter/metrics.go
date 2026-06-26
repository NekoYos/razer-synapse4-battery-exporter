package exporter

import (
	"fmt"
	"io"
	"strings"

	"github.com/NekoYos/razer-synapse4-battery-exporter/internal/synapse"
)

type BuildInfo struct {
	Version   string
	Commit    string
	BuildDate string
}

func WriteMetrics(w io.Writer, devices []synapse.Device, scrapeDurationMilliseconds float64) {
	WriteDeviceMetrics(w, devices)
	WriteBuildInfoMetric(w, BuildInfo{
		Version:   "dev",
		Commit:    "unknown",
		BuildDate: "unknown",
	})
	WriteScrapeDurationMetric(w, scrapeDurationMilliseconds)
}

func WriteDeviceMetrics(w io.Writer, devices []synapse.Device) {
	fmt.Fprintln(w, "# HELP razer_device_battery_level Razer device battery level in percent.")
	fmt.Fprintln(w, "# TYPE razer_device_battery_level gauge")
	for _, device := range devices {
		fmt.Fprintf(
			w,
			"razer_device_battery_level{name=\"%s\",serial=\"%s\",category=\"%s\"} %d\n",
			escapeLabel(device.Name),
			escapeLabel(device.SerialNumber),
			escapeLabel(device.Category),
			device.BatteryLevel,
		)
	}

	fmt.Fprintln(w, "# HELP razer_device_battery_status Razer device charging status: 0 unknown/off, 1 full/not charging, 2 charging.")
	fmt.Fprintln(w, "# TYPE razer_device_battery_status gauge")
	for _, device := range devices {
		fmt.Fprintf(
			w,
			"razer_device_battery_status{name=\"%s\",serial=\"%s\",category=\"%s\"} %d\n",
			escapeLabel(device.Name),
			escapeLabel(device.SerialNumber),
			escapeLabel(device.Category),
			synapse.BatteryStatusValue(device.ChargingStatus),
		)
	}
}

func WriteScrapeDurationMetric(w io.Writer, scrapeDurationMilliseconds float64) {
	fmt.Fprintln(w, "# HELP razer_exporter_scrape_duration_milliseconds Time spent handling this metrics scrape inside the exporter.")
	fmt.Fprintln(w, "# TYPE razer_exporter_scrape_duration_milliseconds gauge")
	fmt.Fprintf(w, "razer_exporter_scrape_duration_milliseconds %.3f\n", scrapeDurationMilliseconds)
}

func WriteBuildInfoMetric(w io.Writer, info BuildInfo) {
	fmt.Fprintln(w, "# HELP razer_exporter_build_info Razer Synapse 4 Battery Exporter build information.")
	fmt.Fprintln(w, "# TYPE razer_exporter_build_info gauge")
	fmt.Fprintf(
		w,
		"razer_exporter_build_info{version=\"%s\",commit=\"%s\",build_date=\"%s\"} 1\n",
		escapeLabel(info.Version),
		escapeLabel(info.Commit),
		escapeLabel(info.BuildDate),
	)
}

func escapeLabel(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	return strings.ReplaceAll(value, `"`, `\"`)
}
