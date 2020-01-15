package condition

import (
	"fmt"

	"github.com/markuskont/go-sigma-rule-engine/pkg/match"
	"github.com/markuskont/go-sigma-rule-engine/pkg/rule"
	"github.com/markuskont/go-sigma-rule-engine/pkg/sigma"
)

func parseSearch(t tokens, detect sigma.Detection, c rule.Config, entry bool) (match.Branch, error) {
	if t.contains(IdentifierAll) {
		return nil, sigma.ErrUnsupportedToken{Msg: IdentifierAll.Literal()}
	}
	if t.contains(IdentifierWithWildcard) {
		return nil, sigma.ErrUnsupportedToken{Msg: IdentifierWithWildcard.Literal()}
	}
	if t.contains(StOne) || t.contains(StAll) {
		return nil, sigma.ErrUnsupportedToken{Msg: fmt.Sprintf("%s / %s", StAll.Literal(), StOne.Literal())}
	}

	rules := t.splitByToken(KeywordOr)

	branch := make([]match.Branch, 0)
	for _, group := range rules {
		group.discoverSubGroups()
		b, err := newBranchFromGroup(*group, detect, c)
		if err != nil {
			return nil, err
		}
		branch = append(branch, b)
	}

	if len(branch) == 1 {
		return branch[0], nil
	}

	return match.NodeSimpleOr(branch), nil
}

func newBranchFromGroup(group tokensHandler, detect sigma.Detection, c rule.Config) (match.Branch, error) {

	// recursion here
	if group.hasSubGroup {
		branch := make([]match.Branch, 0)
		var offset int

		for i, pos := range group.subGroups {

			// Grab statement before the group
			sub := group.tokens[pos.From+1 : pos.To-1]

			if regular := group.tokens[offset:pos.From]; len(regular) > 0 {
				for i, item := range regular {
					switch t := item.T; {
					case t == Identifier:
						r, err := newRuleMatcherFromIdent(detect.Get(item.Val), c.LowerCase)
						if err != nil {
							return nil, err
						}
						branch = append(branch, func() match.Branch {
							if i > 0 && regular[i-1:].isNegated() {
								return match.NodeNot{Branch: r}
							}
							return r
						}())
					}
				}
				// move offset to group end
				offset = pos.To
			}

			b, err := parseSearch(sub, detect, c, false)
			if err != nil {
				return nil, err
			}
			branch = append(branch, func() match.Branch {
				if pos.From > 0 && group.tokens[pos.From-1].T == KeywordNot {
					return match.NodeNot{Branch: b}
				}
				return b
			}())

			if i == len(group.subGroups)-1 {
				if regular := group.tokens[pos.To:]; len(regular) != 0 {
					for i, item := range regular {
						switch t := item.T; {
						case t == Identifier:
							r, err := newRuleMatcherFromIdent(detect.Get(item.Val), c.LowerCase)
							if err != nil {
								return nil, err
							}
							branch = append(branch, func() match.Branch {
								if i > 0 && regular[i-1:].isNegated() {
									return match.NodeNot{Branch: r}
								}
								return r
							}())
						}
					}
				}
			}
		}
		return match.NodeSimpleAnd(branch), nil
	}

	if group.isSimpleIdent() {
		var ident Item
		switch len(group.tokens) {
		case 1:
			ident = group.tokens[0]
		case 2:
			ident = group.tokens[1]
		}
		r, err := newRuleMatcherFromIdent(detect.Get(ident.Val), c.LowerCase)
		return func() match.Branch {
			if group.isNegated() {
				return match.NodeNot{Branch: r}
			}
			return r
		}(), err
	}

	// TODO - handle everything in this function
	return parseSimpleSearch(group.tokens, detect, c)
}

// simple search == just a valid group sequence with no sub-groups
func parseSimpleSearch(t tokens, detect sigma.Detection, c rule.Config) (match.Branch, error) {
	rules := t.splitByToken(KeywordOr)

	branch := make([]match.Branch, 0)
	for _, group := range rules {
		if l := len(group.tokens); l == 1 || (l == 2 && group.isNegated()) {
			var ident Item
			switch l {
			case 1:
				ident = group.tokens[0]
			case 2:
				ident = group.tokens[1]
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
		} else {
			andGroup := make([]match.Branch, 0)
			for i, item := range group.tokens {
				switch item.T {
				case Identifier:
					r, err := newRuleMatcherFromIdent(detect.Get(item.Val), c.LowerCase)
					if err != nil {
						return nil, err
					}
					andGroup = append(andGroup, func() match.Branch {
						if i > 0 && group.tokens[i-1:].isNegated() {
							return match.NodeNot{Branch: r}
						}
						return r
					}())
				}
			}
			branch = append(branch, match.NodeSimpleAnd(andGroup))
		}
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
	sigma sigma.Detection

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
	b, err := parseSearch(p.tokens, p.sigma, rule.Config{}, true)
	if err != nil {
		return err
	}
	p.result = b
	return nil
}

func (p *parser) collectAndValidateTokenSequences() error {
	for item := range p.lex.items {

		if item.T == TokUnsupp {
			return sigma.ErrUnsupportedToken{Msg: item.Val}
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
