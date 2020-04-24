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

func (e ErrMissingDetection) Error() string { return "sigma rule is missing detection field" }

// ErrEmptyDetection indicates detection field present but empty
type ErrEmptyDetection struct{}

func (e ErrEmptyDetection) Error() string { return "sigma rule has detection but is empty" }

// ErrMissingCondition indicates missing condition field
type ErrMissingCondition struct{}

func (e ErrMissingCondition) Error() string { return "complex sigma rule is missing condition" }

// ErrIncompleteDetection indicates a rule has defined identifiers that are missing in detection map
type ErrIncompleteDetection struct {
	Condition string
	Keys      []string
	Msg       string
}

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

func (e ErrUnsupportedToken) Error() string { return fmt.Sprintf("UNSUPPORTED TOKEN: %s", e.Msg) }

// ErrWip indicates a rule expression that is currently Work In Progress
// Functions like ErrUnsupportedToken but indicates that feature is under active development
// Non-critical escape hatch while debugging
type ErrWip struct{}

func (e ErrWip) Error() string { return fmt.Sprintf("Work in progress") }

// ErrParseYaml indicates YAML parsing error
type ErrParseYaml struct {
	Path  string
	Err   error
	Count int
}

func (e ErrParseYaml) Error() string {
	return fmt.Sprintf("%d - File: %s; Err: %s", e.Count, e.Path, e.Err)
}

// ErrGotBrokenYamlFiles is a bulk error handler for dealing with broken sigma rules
// Some rules are bound to fail, no reason to exit entire application
// Individual errors can be collected and returned at the end
// Called decides if they should be only reported or it warrants full exit
type ErrBulkParseYaml struct {
	Errs []ErrParseYaml
}

func (e ErrBulkParseYaml) Error() string {
	return fmt.Sprintf("Got %d broken yaml files", len(e.Errs))
}

// ErrInvalidTokenSeq indicates expression syntax error from rule writer
// For example, two indents should be separated by a logical AND / OR operator
type ErrInvalidTokenSeq struct {
	Prev, Next Item
	Collected  []Item
}

func (e ErrInvalidTokenSeq) Error() string {
	return fmt.Sprintf("Seq error after collecting %d elements. Invalid token sequence %s -> %s. Values: %s -> %s.",
		len(e.Collected), e.Prev.T, e.Next.T, e.Prev.Val, e.Next.Val)
}

// ErrIncompleteTokenSeq is invoked when lex channel drain does not end with EOF
// thus indicating incomplete lexing sequence
type ErrIncompleteTokenSeq struct {
	Expression string
	Items      []Item
	Last       Item
}

func (e ErrIncompleteTokenSeq) Error() string {
	return fmt.Sprintf(
		"last element should be EOF, got token %s with value %s",
		e.Last.T.String(), e.Last.Val,
	)
}
