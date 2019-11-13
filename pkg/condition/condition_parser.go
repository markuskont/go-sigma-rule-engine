package condition

import (
	"fmt"
	"strings"

	"github.com/markuskont/go-sigma-rule-engine/pkg/match"
	"github.com/markuskont/go-sigma-rule-engine/pkg/rule"
	"github.com/markuskont/go-sigma-rule-engine/pkg/types"
)

type tokens []Item

func (t tokens) index(tok Token) int {
	for i, item := range t {
		if item.T == tok {
			return i
		}
	}
	return -1
}

func (t tokens) reverseIndex(tok Token) int {
	for i := len(t) - 1; i > 0; i-- {
		if t[i].T == tok {
			return i
		}
	}
	return -1
}

func (t tokens) contains(tok Token) bool {
	for _, item := range t {
		if item.T == tok {
			return true
		}
	}
	return false
}

type parser struct {
	lex *lexer

	// maintain a list of collected and validated tokens
	tokens

	// memorize last token to validate proper sequence
	// for example, two identifiers have to be joined via logical AND or OR, otherwise the sequence is invalid
	previous Token

	// sigma detection map that contains condition query and relevant fields
	sigma map[string]interface{}

	// for debug
	condition string

	// sigma condition rules
	rules []interface{}
}

func Parse(s map[string]interface{}) (*match.Tree, error) {
	// detection should have condition and at least 1 identifier
	if s == nil || len(s) < 2 {
		return nil, fmt.Errorf("sigma rule detection missing or has less than 2 elements")
	}
	raw, ok := s["condition"].(string)
	if !ok {
		return nil, fmt.Errorf("sigma rule condition missing or wrong type")
	}
	delete(s, "condition")
	p := &parser{
		lex:       lex(raw),
		sigma:     s,
		tokens:    make([]Item, 0),
		previous:  TokBegin,
		condition: raw,
	}
	if err := p.run(); err != nil {
		return nil, err
	}
	return nil, nil
}

func (p *parser) run() error {
	if p.lex == nil {
		return fmt.Errorf("cannot run condition parser, lexer not initialized")
	}
	// Pass 1: collect tokens, do basic sequence validation and collect sigma fields
	for item := range p.lex.items {

		if item.T == TokUnsupp {
			return ErrUnsupported{Msg: item.Val}
		}
		if !validTokenSequence(p.previous, item.T) {
			return fmt.Errorf(
				"invalid token sequence %s -> %s. Value: %s",
				p.previous,
				item.T,
				item.Val,
			)
		}
		if item.T != LitEof {
			p.tokens = append(p.tokens, item)
		}
		p.previous = item.T
	}
	if p.previous != LitEof {
		return fmt.Errorf("last element should be EOF, got %s", p.previous.String())
	}

	// Pass 2: find groups
	/*
		for p.contains(SepLpar) {

		}
	*/
	return nil
}

func isKeywords(s string) bool { return strings.HasPrefix(s, "keywords") }

type ruleKeywordBranch struct {
	id int
	rule.Keyword
}

// Match implements sigma Matcher
func (r ruleKeywordBranch) Match(obj types.EventChecker) bool {
	return r.Keyword.Match(obj)
}

// Self returns Node or final rule object for debugging and/or walking the tree
// Must be type switched externally
func (r ruleKeywordBranch) Self() interface{} {
	return r.Keyword.Self()
}

// GetID implements Identifier
func (r ruleKeywordBranch) GetID() int {
	return r.id
}

// SetID implements Identifier
func (r *ruleKeywordBranch) SetID(id int) {
	r.id = id
}

type ruleSelectionBranch struct {
	id int
	rule.Fields
}

// Match implements sigma Matcher
func (r ruleSelectionBranch) Match(obj types.EventChecker) bool {
	return r.Match(obj)
}

// Self returns Node or final rule object for debugging and/or walking the tree
// Must be type switched externally
func (r ruleSelectionBranch) Self() interface{} {
	return r.Fields.Self()
}

// GetID implements Identifier
func (r ruleSelectionBranch) GetID() int {
	return r.id
}

// SetID implements Identifier
func (r *ruleSelectionBranch) SetID(id int) {
	r.id = id
}

type ErrUnsupported struct {
	Msg string
}

func (e ErrUnsupported) Error() string { return fmt.Sprintf("UNSUPPORTED TOKEN: %s", e.Msg) }
