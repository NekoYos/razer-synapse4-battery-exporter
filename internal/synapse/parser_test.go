package synapse

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseDevicesFromLogReturnsLatestBatteryDevices(t *testing.T) {
	input := strings.Join([]string{
		`[2026/06/12 20:50:00.000] info: other line`,
		`[2026/06/12 20:51:22.168] info: mapDevices ~ devices: [{"serialNumber":"mouse-1","name":{"en":"Razer Naga V2 Pro"},"category":"MOUSE","powerStatus":{"chargingStatus":"NoCharge_BatteryFull","level":92}},{"serialNumber":"mat-1","name":{"en":"Razer Strider Chroma"},"category":"MOUSEMAT"}]`,
		`[2026/06/12 20:52:22.168] info: mapDevices ~ devices: [{"serialNumber":"keyboard-1","productName":{"en":"Razer DeathStalker V2 Pro"},"category":"KEYBOARD","powerStatus":{"chargingStatus":"Charging","level":60}}]`,
	}, "\n")

	devices, err := ParseDevicesFromLog(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseDevicesFromLog returned error: %v", err)
	}

	if len(devices) != 1 {
		t.Fatalf("expected 1 battery device, got %d", len(devices))
	}
	if devices[0].Name != "Razer DeathStalker V2 Pro" {
		t.Fatalf("unexpected device name: %q", devices[0].Name)
	}
	if devices[0].BatteryLevel != 60 {
		t.Fatalf("unexpected battery level: %d", devices[0].BatteryLevel)
	}
}

func TestParseLatestDevicesFromLogReadsFromEnd(t *testing.T) {
	input := strings.Join([]string{
		`[2026/06/12 20:51:22.168] info: mapDevices ~ devices: [{"serialNumber":"old","name":{"en":"Old Mouse"},"category":"MOUSE","powerStatus":{"chargingStatus":"Discharging","level":10}}]`,
		strings.Repeat("padding", 20000),
		`[2026/06/12 20:52:22.168] info: mapDevices ~ devices: [{"serialNumber":"new","name":{"en":"New Keyboard"},"category":"KEYBOARD","powerStatus":{"chargingStatus":"Charging","level":60}}]`,
		`[2026/06/12 20:53:22.168] info: unrelated final line`,
	}, "\n")

	reader := bytes.NewReader([]byte(input))
	devices, err := ParseLatestDevicesFromLog(reader, int64(reader.Len()))
	if err != nil {
		t.Fatalf("ParseLatestDevicesFromLog returned error: %v", err)
	}

	if len(devices) != 1 {
		t.Fatalf("expected 1 battery device, got %d", len(devices))
	}
	if devices[0].SerialNumber != "new" {
		t.Fatalf("unexpected serial number: %q", devices[0].SerialNumber)
	}
}

func TestParseLatestDevicesFromLogSkipsPartialLatestDevicesLine(t *testing.T) {
	input := strings.Join([]string{
		`[2026/06/12 20:52:22.168] info: mapDevices ~ devices: [{"serialNumber":"new","name":{"en":"New Keyboard"},"category":"KEYBOARD","powerStatus":{"chargingStatus":"Charging","level":60}}]`,
		`[2026/06/12 20:53:22.168] info: mapDevices ~ devices: [{"serialNumber":"partial"`,
	}, "\n")

	reader := bytes.NewReader([]byte(input))
	devices, err := ParseLatestDevicesFromLog(reader, int64(reader.Len()))
	if err != nil {
		t.Fatalf("ParseLatestDevicesFromLog returned error: %v", err)
	}

	if len(devices) != 1 {
		t.Fatalf("expected 1 battery device, got %d", len(devices))
	}
	if devices[0].SerialNumber != "new" {
		t.Fatalf("unexpected serial number: %q", devices[0].SerialNumber)
	}
}
