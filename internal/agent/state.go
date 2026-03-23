package agent

// State represents the agent lifecycle state.
type State int

const (
	StateStarting   State = iota
	StateReady
	StateProcessing
	StateIdle
	StateStopped
)

func (s State) String() string {
	switch s {
	case StateStarting:
		return "starting"
	case StateReady:
		return "ready"
	case StateProcessing:
		return "processing"
	case StateIdle:
		return "idle"
	case StateStopped:
		return "stopped"
	default:
		return "unknown"
	}
}
