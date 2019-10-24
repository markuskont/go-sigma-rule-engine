package types

import "fmt"

// MessageGetter is for implementing keyword matching by string wildcard or regexp
// Event should return whatever list of fields that are relevant for these matches
type MessageGetter interface {
	// GetMessage implements MessageGetter
	GetMessage() []string
}

// SelectionGetter is used for selection matching
type SelectionGetter interface {
	// GetField returns a success status and arbitrary field content if requested map key is present
	GetField(string) (interface{}, bool)
}

// EventChecker is a collection of interfaces required to implement sigma rule matching for an arbitrary event
// EventChecker should be implemented for any struct that is being used as input for sigma rules
type EventChecker interface {
	MessageGetter
	SelectionGetter
}

// Matcher represents either left or right branch of AST matching tree
type Matcher interface {
	// Match implements sigma Matcher
	Match(EventChecker) bool
}

type ErrInvalidRegex struct {
	Pattern string
	Err     error
}

func (e ErrInvalidRegex) Error() string {
	return fmt.Sprintf("/%s/ %s", e.Pattern, e.Err)
}
