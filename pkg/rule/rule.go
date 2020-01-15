package rule

import "github.com/markuskont/go-sigma-rule-engine/pkg/sigma"

type Config struct {
	LowerCase   bool
	NumToString bool
}

type FieldsList []*Fields

// Match implements sigma Matcher
func (f FieldsList) Match(obj sigma.EventChecker) bool {
	for _, rule := range f {
		if rule.Match(obj) {
			return true
		}
	}
	return false
}

// Self returns Node or final rule object for debugging and/or walking the tree
// Must be type switched externally
//Identifier
func (f FieldsList) Self() interface{} {
	return f
}
