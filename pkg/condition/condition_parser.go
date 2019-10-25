package condition

import (
	"fmt"
	"strings"

	"github.com/markuskont/go-sigma-rule-engine/pkg/match"
	"github.com/markuskont/go-sigma-rule-engine/pkg/rule"
	"github.com/markuskont/go-sigma-rule-engine/pkg/types"
)

type idents struct {
	// sigma detection map that contains condition query and relevant fields
	sigma map[string]interface{}
	// shortcut to sigma detection keys
	identifiers []string
}

type parser struct {
	lex *lexer

	idents

	// maintain a list of collected and validated tokens
	// ma be useless and therefore nuked
	items []Token

	// memorize last token to validate proper sequence
	// for example, two identifiers have to be joined via logical AND or OR, otherwise the sequence is invalid
	previous Token

	// total number of valid tokens collected
	// to simplify handling trivial cases, like single keywords, selection or one of | all of statement
	total int
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
	p := &parser{
		lex: lex(raw),
		idents: idents{
			sigma: s,
			identifiers: func() []string {
				l := make([]string, 0)
				for k, _ := range s {
					if k == "condition" {
						continue
					}
					l = append(l, k)
				}
				return l
			}(),
		},
		items:    make([]Token, 0),
		previous: TokBegin,
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
	for item := range p.lex.items {

		if !validTokenSequence(p.previous, item.T) {
			return fmt.Errorf(
				"invalid token sequence %s -> %s. Value: %s",
				p.previous,
				item.T,
				item.Val,
			)
		}
		p.previous = item.T

		switch item.T {
		case TokErr:
			return fmt.Errorf("invalid token: %s", item.Val)
		case StOne, StAll:
			if p.total > 0 {
			}

		case SepLpar:
			fmt.Println("group begin")

		case SepRpar:
		case IdentifierAll:
		case Identifier:
		case IdentifierWithWildcard:
			// error here
		case LitEof:
			switch p.total {
			case 1:
				// Should only be IDENT in elements
			case 2:
				// Should only be xOf and IDENT/THEM in elements
			}
		case KeywordAnd, KeywordOr:
		}
		fmt.Println(item)
		p.total++
	}
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
