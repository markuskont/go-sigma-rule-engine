package condition

import (
	"fmt"

	"github.com/markuskont/go-sigma-rule/pkg/match"
)

func Parse(raw string) (*match.Tree, error) {
	p := &parser{lex: lex(raw)}
	p.run()
	return match.NewTree(nil), nil
}

type parser struct {
	result interface{}
	lex    *lexer
	errors []error
	sigma  map[string]interface{}
}

func (p *parser) run() error {
	if p.lex == nil {
		return fmt.Errorf("cannot run condition parser, lexer not initialized")
	}
	for item := range p.lex.items {
		switch item.T {
		case Identifier:
			if _, ok := p.sigma[item.Val]; ok {

			}
		}
		fmt.Println(item)
	}
	return nil
}
