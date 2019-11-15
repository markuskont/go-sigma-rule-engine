package condition

import (
	"fmt"
	"reflect"

	"github.com/markuskont/go-sigma-rule-engine/pkg/match"
	"github.com/markuskont/go-sigma-rule-engine/pkg/rule"
	"github.com/markuskont/go-sigma-rule-engine/pkg/types"
)

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
