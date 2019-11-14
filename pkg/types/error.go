package types

import "fmt"

type ErrMissingDetection struct{}

func (e ErrMissingDetection) Error() string { return "sigma rule is missing detection field" }

type ErrEmptyDetection struct{}

func (e ErrEmptyDetection) Error() string { return "sigma rule has detection but is empty" }

type ErrMissingCondition struct{}

func (e ErrMissingCondition) Error() string { return "complex sigma rule is missing condition" }

type ErrIncompleteDetection struct {
	Condition string
	Keys      []string
}

func (e ErrIncompleteDetection) Error() string {
	return fmt.Sprintf(
		"incomplete rule, missing fields from condition. [%s]. Has %+v.",
		e.Condition,
		func() []string {
			if e.Keys != nil {
				return e.Keys
			}
			return []string{}
		}(),
	)
}

type ErrUnsupportedToken struct{ Msg string }

func (e ErrUnsupportedToken) Error() string { return fmt.Sprintf("UNSUPPORTED TOKEN: %s", e.Msg) }
