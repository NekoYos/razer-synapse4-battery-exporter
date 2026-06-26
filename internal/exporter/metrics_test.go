package exporter

import (
	"strings"
	"testing"

	"github.com/NekoYos/razer-synapse4-battery-exporter/internal/synapse"
)

func TestWriteMetrics(t *testing.T) {
	var out strings.Builder
	WriteMetrics(&out, []synapse.Device{
		{
			SerialNumber:   `serial"1`,
			Name:           `Razer \ Mouse`,
			Category:       "MOUSE",
			BatteryLevel:   89,
			ChargingStatus: "NoCharge_BatteryFull",
		},
		{
			SerialNumber:   "serial-2",
			Name:           "Razer Keyboard",
			Category:       "KEYBOARD",
			BatteryLevel:   62,
			ChargingStatus: "Charging",
		},
	}, 12.345)

	got := out.String()
	want := []string{
		`razer_device_battery_level{name="Razer \\ Mouse",serial="serial\"1",category="MOUSE"} 89`,
		`razer_device_battery_status{name="Razer \\ Mouse",serial="serial\"1",category="MOUSE"} 1`,
		`razer_device_battery_status{name="Razer Keyboard",serial="serial-2",category="KEYBOARD"} 2`,
		`razer_exporter_build_info{version="dev",commit="unknown",build_date="unknown"} 1`,
		`razer_exporter_scrape_duration_milliseconds 12.345`,
	}
	for _, line := range want {
		if !strings.Contains(got, line) {
			t.Fatalf("expected metrics to contain %q, got:\n%s", line, got)
		}
	}
}

func TestWriteBuildInfoMetricEscapesLabels(t *testing.T) {
	var out strings.Builder
	WriteBuildInfoMetric(&out, BuildInfo{
		Version:   `v"1`,
		Commit:    `abc\123`,
		BuildDate: "2026-06-26T00:00:00Z",
	})

	got := out.String()
	want := `razer_exporter_build_info{version="v\"1",commit="abc\\123",build_date="2026-06-26T00:00:00Z"} 1`
	if !strings.Contains(got, want) {
		t.Fatalf("expected build info metric to contain %q, got:\n%s", want, got)
	}
}
