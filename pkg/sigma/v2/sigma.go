package sigma

// Keyworder implements keywords sigma rule type on arbitrary event
// Should return list of fields that are relevant for rule matching
type Keyworder interface {
	// Keywords implements Keyworder
	Keywords() ([]string, bool)
}

// Selector implements selection sigma rule type
type Selector interface {
	// Select implements Selector
	Select(string) (interface{}, bool)
}

// Event implements sigma rule types by embedding Keyworder and Selector
// Used by rules to extract relevant fields
type Event interface {
	Keyworder
	Selector
}

// Matcher is used for implementing Abstract Syntax Tree for Sigma engine
type Matcher interface {
	// Match implements Matcher
	Match(Event) (bool, bool)
}

// Branch implements Matcher with additional methods for walking and debugging the tree
type Branch interface {
	Matcher

	// Self returns Node or final rule object for debugging and/or walking the tree
	// Must be type switched externally
	// Self() interface{}
}
