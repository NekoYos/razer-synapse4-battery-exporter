package synapse

const (
	BatteryStatusUnknown  = 0
	BatteryStatusFull     = 1
	BatteryStatusCharging = 2
)

func BatteryStatusValue(status string) int {
	switch status {
	case "NoCharge_BatteryFull":
		return BatteryStatusFull
	case "Charging":
		return BatteryStatusCharging
	default:
		return BatteryStatusUnknown
	}
}
