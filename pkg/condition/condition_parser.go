package condition

import (
	"encoding/json"
	"fmt"

	"github.com/markuskont/go-sigma-rule-engine/pkg/match"
	"github.com/markuskont/go-sigma-rule-engine/pkg/rule"
	"github.com/markuskont/go-sigma-rule-engine/pkg/types"
)

type offsets struct {
	From, To int
}

func parseSearch(t tokens, data types.Detection, c rule.Config) (match.Branch, error) {
	fmt.Printf("Parsing %+v\n", t)

	// seek to LPAR -> store offset set balance as 1
	// seek from offset to end -> increment balance when encountering LPAR, decrement when encountering RPAR
	// increment group count on every decrement
	// stop when balance is 0, error of EOF if balance is positive or negative
	// if group count is > 0, fill sub brances via recursion
	// finally, build branch from identifiers and logic statements

	var balance, found int
	groups := make([]*offsets, 0)

	// pass 1 - discover groups
	// TODO - later run fn recursively to parse all sub-elements
	for i, item := range t {
		switch item.T {
		case SepLpar:
			if balance == 0 {
				groups = append(groups, &offsets{From: i, To: -1})
			}
			balance++
		case SepRpar:
			balance--
			if balance == 0 {
				groups[found].To = i
				found++
			}

		case IdentifierAll:
			return nil, fmt.Errorf("TODO - THEM identifier")
		case IdentifierWithWildcard:
			return nil, fmt.Errorf("TODO - wildcard identifier")
		case StOne, StAll:
			return nil, fmt.Errorf("TODO - X of statement")

		}
	}

	if balance > 0 || balance < 0 {
		return nil, fmt.Errorf("Broken rule group")
	}

	// TODO - debug, remove
	if len(groups) > 0 {
		j, _ := json.Marshal(groups)
		fmt.Printf("%s\n", data["condition"].(string))
		fmt.Printf("got %d groups offsets are %s\n", len(groups), string(j))
		return nil, fmt.Errorf("TODO - implement parsing sub-groups recursively")
	}
	return parseSimpleSearch(t, data, c)
}

type condWrapper struct {
	Token
	match.Branch
}

// simple search == just a valid group sequence with no sub-groups
// maybe will stay, maybe exists just until I figure out the parse logic
func parseSimpleSearch(t tokens, data types.Detection, c rule.Config) (match.Branch, error) {

	for t.len() > 0 {
		rules := make([]match.Branch, 0)
		if idx := t.index(KeywordOr); idx > 0 {
			for i, item := range t.head(idx) {
				switch item.T {
				case Identifier:
					r, err := newRuleMatcherFromIdent(data.Get(item.Val), c.LowerCase)
					if err != nil {
						return nil, err
					}
					rules = append(rules, func() match.Branch {
						if t.head(idx).isNegated(i) {
							return match.NodeNot{Branch: r}
						}
						return r
					}())
				}
			}
		}
	}

	/*
		for i, item := range t {
			switch item.T {
			case Identifier:
				r, err := newRuleMatcherFromIdent(data.Get(item.Val), c.LowerCase)
				if err != nil {
					return nil, err
				}
				group = append(group, &condWrapper{
					Branch: func() match.Branch {
						if t.isNegated(i) {
							return match.NodeNot{Branch: r}
						}
						return r
					}(),
					Token: cond,
				})

				cond = 0
			case KeywordAnd:
				cond = KeywordAnd
			case KeywordOr:
				cond = KeywordOr
			}
		}
		j, _ := json.Marshal(group)
		fmt.Printf("------>>>>>>>%s\n", string(j))
	*/

	return nil, fmt.Errorf("WIP")
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

	// sigma condition rules
	rules []interface{}
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
	fmt.Println("------------------")
	if _, err := parseSearch(p.tokens, p.sigma, rule.Config{}); err != nil {
		return err
	}
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
