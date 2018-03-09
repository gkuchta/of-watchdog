package config

const (
	// ModeStreaming streams the values live to the caller as they are printed by the process.
	ModeStreaming = 1

	// ModeSerializing reads all the response and buffers before returning
	ModeSerializing = 2

	// ModeAfterBurn for performance tuning
	ModeAfterBurn = 3

	//ModeHTTP for routing requests over HTTP
	ModeHTTP = 4

	ModeTargetHTTP = 5

	ModeTargetStreaming = 6
)

// WatchdogModeConst as a const int
func WatchdogModeConst(mode string) int {
	switch mode {
	case "streaming":
		return ModeStreaming
	case "afterburn":
		return ModeAfterBurn
	case "serializing":
		return ModeSerializing
	case "http":
		return ModeHTTP
	case "target-http":
		return ModeTargetHTTP
	case "target-streaming":
		return ModeTargetStreaming
	default:
		return 0
	}
}

// WatchdogMode as a string
func WatchdogMode(mode int) string {
	switch mode {
	case ModeStreaming:
		return "streaming"
	case ModeAfterBurn:
		return "afterburn"
	case ModeSerializing:
		return "serializing"
	case ModeHTTP:
		return "http"
	case ModeTargetHTTP:
		return "target-http"
	case ModeTargetStreaming:
		return "target-streaming"
	default:
		return "unknown"
	}
}
