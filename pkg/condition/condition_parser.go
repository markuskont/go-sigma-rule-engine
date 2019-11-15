package condition

import (
	"fmt"

	"github.com/markuskont/go-sigma-rule-engine/pkg/match"
	"github.com/markuskont/go-sigma-rule-engine/pkg/types"
)

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

func (p *parser) run() error {
	if p.lex == nil {
		return fmt.Errorf("cannot run condition parser, lexer not initialized")
	}
	// Pass 1: collect tokens, do basic sequence validation and collect sigma fields
	if err := p.collectAndValidateTokenSequences(); err != nil {
		return err
	}
	// Pass 2: find groups
	return nil
}

func (p *parser) collectAndValidateTokenSequences() error {
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
	return nil
}
