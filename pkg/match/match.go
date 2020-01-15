package match

import (
	"github.com/markuskont/go-sigma-rule-engine/pkg/sigma"
)

type Condition int

// Branch implements types.Matcher with additional methods for walking and debugging the tree
type Branch interface {
	sigma.Matcher
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

func (t Tree) Match(obj sigma.EventChecker) bool { return t.Root.Match(obj) }
func (t Tree) Self() interface{}                 { return t.Root }
