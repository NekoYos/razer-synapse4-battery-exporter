package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadDevicesFallsBackToOlderLogWhenNewestHasNoDevices(t *testing.T) {
	localAppData := t.TempDir()
	t.Setenv("LOCALAPPDATA", localAppData)

	logDir := filepath.Join(localAppData, "Razer", "RazerAppEngine", "User Data", "Logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	oldPath := filepath.Join(logDir, "systray_systrayv1.log")
	newPath := filepath.Join(logDir, "systray_systrayv2.log")

	oldLog := `[2026/06/12 20:52:22.168] info: mapDevices ~ devices: [{"serialNumber":"keyboard-1","productName":{"en":"Razer DeathStalker V2 Pro"},"category":"KEYBOARD","powerStatus":{"chargingStatus":"Charging","level":60}}]`
	if err := os.WriteFile(oldPath, []byte(oldLog), 0o644); err != nil {
		t.Fatalf("WriteFile old log returned error: %v", err)
	}
	if err := os.WriteFile(newPath, []byte(`[2026/06/12 20:53:22.168] info: new rotated log`), 0o644); err != nil {
		t.Fatalf("WriteFile new log returned error: %v", err)
	}

	now := time.Now()
	if err := os.Chtimes(oldPath, now.Add(-time.Minute), now.Add(-time.Minute)); err != nil {
		t.Fatalf("Chtimes old log returned error: %v", err)
	}
	if err := os.Chtimes(newPath, now, now); err != nil {
		t.Fatalf("Chtimes new log returned error: %v", err)
	}

	devices, resolvedPath, err := readDevices("")
	if err != nil {
		t.Fatalf("readDevices returned error: %v", err)
	}

	if resolvedPath != oldPath {
		t.Fatalf("expected fallback to old log %q, got %q", oldPath, resolvedPath)
	}
	if len(devices) != 1 {
		t.Fatalf("expected 1 battery device, got %d", len(devices))
	}
	if devices[0].BatteryLevel != 60 {
		t.Fatalf("unexpected battery level: %d", devices[0].BatteryLevel)
	}
}
