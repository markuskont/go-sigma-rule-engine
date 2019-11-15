package condition

import (
	"fmt"
	"reflect"
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

// TODO - perhaps we should invoke parse only if we actually need to parse the query statement and simply instantiate a single-branch rule otherwise
func Parse(s types.Detection) (*match.Tree, error) {
	if s == nil {
		return nil, types.ErrMissingDetection{}
	}
	switch len(s) {
	case 0:
		return nil, types.ErrMissingDetection{}
	case 1:
		// Simple case - should have only one search field, but should not have a condition field
		if c, ok := s["condition"].(string); ok {
			return nil, types.ErrIncompleteDetection{Condition: c}
		}
	case 2:
		// Simple case - one condition statement comprised of single IDENT that matches the second field name
		if c, ok := s["condition"].(string); !ok {
			return nil, types.ErrIncompleteDetection{Condition: "MISSING"}
		} else {
			if _, ok := s[c]; !ok {
				return nil, types.ErrIncompleteDetection{
					Condition: c,
					Msg:       fmt.Sprintf("Field %s defined in condition missing from map.", c),
					Keys:      s.FieldSlice(),
				}
			}
		}
		delete(s, "condition")
	default:
		// Complex case, time to build syntax tree out of condition statement
		raw, ok := s["condition"].(string)
		if !ok {
			return nil, types.ErrMissingCondition{}
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
	// Should only have one element as complex scenario is handled separately
	rx := s.Fields()
	ast := &match.Tree{}
	root, err := newRuleMatcherFromIdent(<-rx, false)
	if err != nil {
		return nil, err
	}
	ast.Root = root
	return ast, nil
}

func interfaceMapToStringInterfaceMap(m map[interface{}]interface{}) (map[string]interface{}, error) {
	m2 := make(map[string]interface{})
	for k, v := range m {
		sk, ok := k.(string)
		if !ok {
			return m2, fmt.Errorf("failed to create selection rule from interface")
		}
		m2[sk] = v
	}
	return m2, nil

}

func newRuleMatcherFromIdent(v types.SearchExpr, toLower bool) (match.Branch, error) {
	switch v.Type {
	case types.ExprKeywords:
		return rule.NewKeywordFromInterface(toLower, v.Content)
	case types.ExprSelection:
		switch m := v.Content.(type) {
		case map[string]interface{}:
			return rule.NewFields(m, toLower, false)
		case []interface{}:
			// might be a list of selections where each entry is a distinct selection rule joined by logical OR
			branch := make(rule.FieldsList, 0)
			for _, raw := range m {
				var (
					elem *rule.Fields
					err  error
				)
				switch expr := raw.(type) {
				case map[interface{}]interface{}:
					m2, err := interfaceMapToStringInterfaceMap(expr)
					if err != nil {
						return nil, err
					}
					elem, err = rule.NewFields(m2, toLower, false)
				case map[string]interface{}:
					elem, err = rule.NewFields(expr, toLower, false)
				default:
					return nil, fmt.Errorf("TODO")
				}
				if err != nil {
					return nil, err
				}
				branch = append(branch, elem)
			}
			return branch, nil
		case map[interface{}]interface{}:
			m2, err := interfaceMapToStringInterfaceMap(m)
			if err != nil {
				return nil, err
			}
			return rule.NewFields(m2, toLower, false)
		default:
			return nil, fmt.Errorf(
				"selection rule %s should be defined as a map, got %s",
				v.Name,
				reflect.TypeOf(v.Content).String(),
			)
		}
	default:
		return nil, fmt.Errorf("unable to parse rule definition")
	}
}

func (p *parser) run() error {
	if p.lex == nil {
		return fmt.Errorf("cannot run condition parser, lexer not initialized")
	}
	// Pass 1: collect tokens, do basic sequence validation and collect sigma fields
	for item := range p.lex.items {

		if item.T == TokUnsupp {
			return types.ErrUnsupportedToken{Msg: item.Val}
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
func (r ruleSelectionBranch) Match(obj types.EventChecker) bool { return r.Match(obj) }

// Self returns Node or final rule object for debugging and/or walking the tree
// Must be type switched externally
func (r ruleSelectionBranch) Self() interface{} { return r.Fields.Self() }

// GetID implements Identifier
func (r ruleSelectionBranch) GetID() int { return r.id }

// SetID implements Identifier
func (r *ruleSelectionBranch) SetID(id int) { r.id = id }
