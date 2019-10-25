package condition

import (
	"fmt"
	"strings"

	"github.com/markuskont/go-sigma-rule-engine/pkg/match"
)

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
		lex:   lex(raw),
		sigma: s,
		items: make([]Token, 0),
	}
	if err := p.run(); err != nil {
		return nil, err
	}
	return nil, nil
}

type parser struct {
	lex      *lexer
	sigma    map[string]interface{}
	items    []Token
	previous Token
	// only one ident with EOF?
	// only one of X | all of X with EOF
	total int
}

func (p *parser) run() error {
	if p.lex == nil {
		return fmt.Errorf("cannot run condition parser, lexer not initialized")
	}
	fmt.Println("------------")
	for item := range p.lex.items {
		if p.total == 0 {
			p.total++
			p.previous = item.T
			continue
		}
		switch item.T {
		case Identifier, IdentifierWithWildcard, IdentifierAll:
			switch item.T {
			case Identifier:
				// Check if is present
			case IdentifierWithWildcard:
			}
			// is it keyword or selection?
			if _, ok := p.sigma[item.Val]; ok {

			}
			// error here
		case LitEof:
			switch p.total {
			case 1:
				// Should only be IDENT in elements
			case 2:
				// Should only be xOf and IDENT/THEM in elements
			}
		case SepLpar:
		case SepRpar:
		case KeywordAnd, KeywordOr:
		}
		fmt.Println(item)
		p.total++
	}
	return nil
}

func isKeywords(s string) bool { return strings.HasPrefix(s, "keywords") }

type group []Token
type query []group
