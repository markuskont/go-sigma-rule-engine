package condition

import (
	"fmt"

	"github.com/markuskont/go-sigma-rule-engine/pkg/match"
	"github.com/markuskont/go-sigma-rule-engine/pkg/rule"
	"github.com/markuskont/go-sigma-rule-engine/pkg/types"
)

// TODO - only use this function as wrapper for unsupported conditions
func parseSearch(t tokens, data types.Detection, c rule.Config) (match.Branch, error) {
	fmt.Printf("Parsing %+v\n", t)

	// seek to LPAR -> store offset set balance as 1
	// seek from offset to end -> increment balance when encountering LPAR, decrement when encountering RPAR
	// increment group count on every decrement
	// stop when balance is 0, error of EOF if balance is positive or negative
	// if group count is > 0, fill sub brances via recursion
	// finally, build branch from identifiers and logic statements

	if t.contains(IdentifierAll) {
		return nil, types.ErrUnsupportedToken{Msg: IdentifierAll.Literal()}
	}
	if t.contains(IdentifierWithWildcard) {
		return nil, types.ErrUnsupportedToken{Msg: IdentifierWithWildcard.Literal()}
	}
	if t.contains(StOne) || t.contains(StAll) {
		return nil, types.ErrUnsupportedToken{Msg: fmt.Sprintf("%s / %s", StAll.Literal(), StOne.Literal())}
	}

	// pass 1 - discover groups
	_, ok, err := newGroupOffsetInTokens(t)
	if err != nil {
		return nil, err
	}
	if ok {
		return nil, types.ErrUnsupportedToken{Msg: "GROUP"}
	}

	return parseSimpleSearch(t, data, c)
}

// simple search == just a valid group sequence with no sub-groups
// maybe will stay, maybe exists just until I figure out the parse logic
func parseSimpleSearch(t tokens, detect types.Detection, c rule.Config) (match.Branch, error) {
	rules := make([]tokens, 0)
	branch := make([]match.Branch, 0)

	var start int
	last := len(t) - 1
	for pos, item := range t {
		if item.T == KeywordOr || pos == last {
			switch pos {
			case last:
				rules = append(rules, t[pos:])
			default:
				rules = append(rules, t[start:pos])
				start = pos + 1
			}
		}
	}

	// TODO - recursively parse nested groups
	for _, group := range rules {
		if l := len(group); l == 1 || (l == 2 && group.isNegated()) {
			var ident Item
			switch l {
			case 1:
				ident = group[0]
			case 2:
				ident = group[1]
			}
			// TODO - move to separate fn to reduce redundant code
			r, err := newRuleMatcherFromIdent(detect.Get(ident.Val), c.LowerCase)
			if err != nil {
				return nil, err
			}
			branch = append(branch, func() match.Branch {
				if group.isNegated() {
					return match.NodeNot{Branch: r}
				}
				return r
			}())
			continue
		}
		andGroup := make([]match.Branch, 0)
		for _, item := range group {
			switch item.T {
			case Identifier:
				r, err := newRuleMatcherFromIdent(detect.Get(item.Val), c.LowerCase)
				if err != nil {
					return nil, err
				}
				andGroup = append(andGroup, func() match.Branch {
					if group.isNegated() {
						return match.NodeNot{Branch: r}
					}
					return r
				}())
			}
		}
		branch = append(branch, match.NodeSimpleAnd(andGroup))
	}
	if len(branch) == 1 {
		return branch[0], nil
	}

	return match.NodeSimpleOr(branch), nil
}

type parser struct {
	lex *lexer

	// maintain a list of collected and validated tokens
	tokens

	// memorize last token to validate proper sequence
	// for example, two identifiers have to be joined via logical AND or OR, otherwise the sequence is invalid
	previous Token

	// sigma detection map that contains condition query and relevant fields
	sigma types.Detection

	// for debug
	condition string

	result match.Branch
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
	b, err := parseSearch(p.tokens, p.sigma, rule.Config{})
	if err != nil {
		return err
	}
	p.result = b
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
