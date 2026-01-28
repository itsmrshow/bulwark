package executor

import "errors"

// SkipError indicates a safe, intentional skip.
type SkipError struct {
	Reason string
}

func (e *SkipError) Error() string {
	return e.Reason
}

// NewSkipError creates a SkipError.
func NewSkipError(reason string) error {
	return &SkipError{Reason: reason}
}

// IsSkipError checks whether the error is a SkipError.
func IsSkipError(err error) bool {
	var skip *SkipError
	return errors.As(err, &skip)
}

// SkipReason returns the skip reason if present.
func SkipReason(err error) string {
	var skip *SkipError
	if errors.As(err, &skip) {
		return skip.Reason
	}
	return ""
}
