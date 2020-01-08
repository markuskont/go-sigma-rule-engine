package match

import "github.com/markuskont/go-sigma-rule-engine/pkg/types"

type NodeSimpleAnd []Branch

// Match implements sigma Matcher
func (n NodeSimpleAnd) Match(obj types.EventChecker) bool {
	for _, elem := range n {
		if !elem.Match(obj) {
			return false
		}
	}
	return true
}

// Self returns Node or final rule object for debugging and/or walking the tree
// Must be type switched externally
func (n NodeSimpleAnd) Self() interface{} { return n }

type NodeSimpleOr []Branch

// Match implements sigma Matcher
func (n NodeSimpleOr) Match(obj types.EventChecker) bool {
	for _, elem := range n {
		if elem.Match(obj) {
			return true
		}
	}
	return false
}

// Self returns Node or final rule object for debugging and/or walking the tree
// Must be type switched externally
//Identifier
func (n NodeSimpleOr) Self() interface{} { return n }

type NodeOr struct {
	ID   int
	L, R Branch
}

// Match implements sigma Matcher
func (n NodeOr) Match(obj types.EventChecker) bool {
	return n.L.Match(obj) || n.R.Match(obj)
}

// Self returns Node or final rule object for debugging and/or walking the tree
// Must be type switched externally
func (n NodeOr) Self() interface{} { return n }

type NodeAnd struct {
	ID   int
	L, R Branch
}

// Match implements sigma Matcher
func (n NodeAnd) Match(obj types.EventChecker) bool {
	return n.L.Match(obj) && n.R.Match(obj)
}

// Self returns Node or final rule object for debugging and/or walking the tree
// Must be type switched externally
func (n NodeAnd) Self() interface{} { return n }

type NodeNot struct {
	Branch
}

// Match implements sigma Matcher
func (n NodeNot) Match(obj types.EventChecker) bool {
	return !n.Branch.Match(obj)
}

// Self returns Node or final rule object for debugging and/or walking the tree
// Must be type switched externally
func (n NodeNot) Self() interface{} { return n }
