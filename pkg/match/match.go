package match

import "github.com/markuskont/go-sigma-rule-engine/pkg/types"

type Condition int

// Branch implements types.Matcher with additional methods for walking and debugging the tree
type Branch interface {
	types.Matcher
	// Self returns Node or final rule object for debugging and/or walking the tree
	// Must be type switched externally
	Self() interface{}
	//Identifier
}

// Identifier implements ID retreival and modification mechanisms for adding elements to and finding them from a tree
// That way, a node may be created and kept as interface yet knowing where to place it in a tree
type Identifier interface {
	// GetID implements Identifier
	GetID() int
	// SetID implements Identifier
	SetID(int)
}

type Tree struct {
	Root Branch
}

func (t Tree) Match(obj types.EventChecker) bool { return t.Root.Match(obj) }
func (t Tree) Self() interface{}                 { return t.Root }

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
func (n NodeNot) Match(obj types.EventChecker) bool { return !n.Match(obj) }

// Self returns Node or final rule object for debugging and/or walking the tree
// Must be type switched externally
func (n NodeNot) Self() interface{} { return n }
