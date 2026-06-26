package synapse

import "testing"

func TestBatteryStatusValue(t *testing.T) {
	tests := map[string]int{
		"NoCharge_BatteryFull": BatteryStatusFull,
		"Charging":             BatteryStatusCharging,
		"off":                  BatteryStatusUnknown,
		"SomethingNew":         BatteryStatusUnknown,
		"":                     BatteryStatusUnknown,
	}

	for input, want := range tests {
		if got := BatteryStatusValue(input); got != want {
			t.Fatalf("BatteryStatusValue(%q) = %d, want %d", input, got, want)
		}
	}
}
