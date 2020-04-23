package sigma

import (
	"fmt"
	"strings"
)

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

// Matcher represents a branch in AST tree
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

type RawRule struct {
	Author         string   `yaml:"author" json:"author"`
	Description    string   `yaml:"description" json:"description"`
	Falsepositives []string `yaml:"falsepositives" json:"falsepositives"`
	Fields         []string `yaml:"fields" json:"fields"`
	ID             string   `yaml:"id" json:"id"`
	Level          string   `yaml:"level" json:"level"`
	Title          string   `yaml:"title" json:"title"`
	Status         string   `yaml:"status" json:"status"`
	References     []string `yaml:"references" json:"references"`

	Logsource Logsource `yaml:"logsource" json:"logsource"`
	Detection Detection `yaml:"detection" json:"detection"`
	Tags      Tags      `yaml:"tags" json:"tags"`
}

func (r RawRule) Condition() (string, error) {
	if r.Detection == nil || len(r.Detection) == 0 {
		return "", fmt.Errorf("missing detection key")
	}
	if val, ok := r.Detection["condition"].(string); ok {
		return val, nil
	}
	return "", fmt.Errorf("condition key missing or not a string value")
}

func (r RawRule) GetCondition() string {
	if c, err := r.Condition(); err == nil {
		return c
	}
	return ""
}

type Logsource struct {
	Product    string `yaml:"product" json:"product"`
	Category   string `yaml:"category" json:"category"`
	Service    string `yaml:"service" json:"service"`
	Definition string `yaml:"definition" json:"definition"`
}

type Tags []string

type Result struct {
	Tags

	ID, Title string
}

type Results []Result

type SearchExprType int

const (
	ExprUnk SearchExprType = iota
	ExprSelection
	ExprKeywords
)

type SearchExpr struct {
	Name    string
	Type    SearchExprType
	Content interface{}
}

func (s *SearchExpr) Guess() *SearchExpr {
	if strings.HasPrefix(s.Name, "keyword") {
		s.Type = ExprKeywords
	} else {
		s.Type = ExprSelection
	}
	return s
}

type Detection map[string]interface{}

func (d Detection) Length() int { return len(d) }

func (d Detection) Fields() <-chan SearchExpr {
	tx := make(chan SearchExpr, 0)
	go func() {
		defer close(tx)
		for k, v := range d {
			if k != "condition" {
				e := SearchExpr{
					Name:    k,
					Content: v,
				}
				tx <- *e.Guess()
			}
		}
	}()
	return tx
}

func (d Detection) FieldSlice() []string {
	tx := make([]string, 0)
	rx := d.Fields()
	for item := range rx {
		tx = append(tx, item.Name)
	}
	return tx
}

func (d Detection) Get(key string) *SearchExpr {
	if val, ok := d[key]; ok {
		e := &SearchExpr{
			Name:    key,
			Content: val,
		}
		return e.Guess()
	}
	return nil
}

type ErrMissingDetection struct{}

func (e ErrMissingDetection) Error() string { return "sigma rule is missing detection field" }

type ErrEmptyDetection struct{}

func (e ErrEmptyDetection) Error() string { return "sigma rule has detection but is empty" }

type ErrMissingCondition struct{}

func (e ErrMissingCondition) Error() string { return "complex sigma rule is missing condition" }

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

type ErrUnsupportedToken struct{ Msg string }

func (e ErrUnsupportedToken) Error() string { return fmt.Sprintf("UNSUPPORTED TOKEN: %s", e.Msg) }

type ErrWip struct{}

func (e ErrWip) Error() string { return fmt.Sprintf("Work in progress") }
