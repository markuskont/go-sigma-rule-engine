package sigma

import "fmt"

// ErrInvalidRegex contextualizes broken regular expressions presented by the user
type ErrInvalidRegex struct {
	Pattern string
	Err     error
}

// Error implements error
func (e ErrInvalidRegex) Error() string {
	return fmt.Sprintf("/%s/ %s", e.Pattern, e.Err)
}

// ErrMissingDetection indicates missing detection field
type ErrMissingDetection struct{}

// Error implements error
func (e ErrMissingDetection) Error() string { return "sigma rule is missing detection field" }

// ErrEmptyDetection indicates detection field present but empty
type ErrEmptyDetection struct{}

// Error implements error
func (e ErrEmptyDetection) Error() string { return "sigma rule has detection but is empty" }

// ErrMissingCondition indicates missing condition field
type ErrMissingCondition struct{}

// Error implements error
func (e ErrMissingCondition) Error() string { return "complex sigma rule is missing condition" }

// ErrIncompleteDetection indicates a rule has defined identifiers that are missing in detection map
type ErrIncompleteDetection struct {
	Condition string
	Keys      []string
	Msg       string
}

// Error implements error
func (e ErrIncompleteDetection) Error() string {
	return fmt.Sprintf(
		"incomplete rule, missing fields from condition. [%s]. Has %+v. %s",
		e.Condition,
		func() []string {
			if e.Keys != nil {
				return e.Keys
			}
			return []string{}
		}(),
		e.Msg,
	)
}

// ErrUnsupportedToken is a parser error indicating lexical token that is not yet supported
// Meant to be used as informational warning, rather than application breaking error
type ErrUnsupportedToken struct{ Msg string }

// Error implements error
func (e ErrUnsupportedToken) Error() string { return fmt.Sprintf("UNSUPPORTED TOKEN: %s", e.Msg) }

// ErrWip indicates a rule expression that is currently Work In Progress
// Functions like ErrUnsupportedToken but indicates that feature is under active development
// Non-critical escape hatch while debugging
type ErrWip struct{}

// Error implements error
func (e ErrWip) Error() string { return fmt.Sprintf("Work in progress") }
