package condition

import (
	"fmt"

	"github.com/markuskont/go-sigma-rule-engine/pkg/match"
	"github.com/markuskont/go-sigma-rule-engine/pkg/rule"
	"github.com/markuskont/go-sigma-rule-engine/pkg/types"
)

func parseSearch(t tokens, detect types.Detection, c rule.Config) (match.Branch, error) {
	if t.contains(IdentifierAll) {
		return nil, types.ErrUnsupportedToken{Msg: IdentifierAll.Literal()}
	}
	if t.contains(IdentifierWithWildcard) {
		return nil, types.ErrUnsupportedToken{Msg: IdentifierWithWildcard.Literal()}
	}
	if t.contains(StOne) || t.contains(StAll) {
		return nil, types.ErrUnsupportedToken{Msg: fmt.Sprintf("%s / %s", StAll.Literal(), StOne.Literal())}
	}

	_, ok, err := newGroupOffsetInTokens(t)
	if err != nil {
		return nil, err
	}
	if !ok {
		return parseSimpleSearch(t, detect, c)
	}

	rules := t.splitByOr()

	branch := make([]match.Branch, 0)
	for _, group := range rules {
		if l := len(group); l == 1 || (l == 2 && group.isNegated()) {
			var ident Item
			switch l {
			case 1:
				ident = group[0]
			case 2:
				ident = group[1]
			}
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
	}
	fmt.Println(branch)

	return nil, types.ErrUnsupportedToken{Msg: "GROUP"}
}

// simple search == just a valid group sequence with no sub-groups
func parseSimpleSearch(t tokens, detect types.Detection, c rule.Config) (match.Branch, error) {
	rules := t.splitByOr()

	fmt.Println("------")
	for _, r := range rules {
		fmt.Println(r)
	}
	fmt.Println("******")

	branch := make([]match.Branch, 0)
	for _, group := range rules {
		if l := len(group); l == 1 || (l == 2 && group.isNegated()) {
			var ident Item
			switch l {
			case 1:
				ident = group[0]
			case 2:
				ident = group[1]
			}
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
		for i, item := range group {
			switch item.T {
			case Identifier:
				r, err := newRuleMatcherFromIdent(detect.Get(item.Val), c.LowerCase)
				if err != nil {
					return nil, err
				}
				andGroup = append(andGroup, func() match.Branch {
					if i > 0 && group[i-1:].isNegated() {
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
	// lexer that tokenizes input string
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

	// resulting rule that can be collected later
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
