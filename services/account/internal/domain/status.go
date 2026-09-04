package domain

type Status string

const (
	StatusActive  Status = "ACTIVE"
	StatusBlocked Status = "BLOCKED"
	StatusClosed  Status = "CLOSED"
)

func (status Status) IsValid() bool {
	switch status {
	case StatusActive, StatusBlocked, StatusClosed:
		return true
	default:
		return false
	}
}

func canTransition(from, to Status) bool {
	switch from {
	case StatusActive:
		return to == StatusBlocked || to == StatusClosed
	case StatusBlocked:
		return to == StatusActive || to == StatusClosed
	default:
		return false
	}
}
