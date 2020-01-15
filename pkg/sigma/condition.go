package sigma

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Item struct {
	T   Token
	Val string
}

func (i Item) String() string { return i.Val }

// TODO - perhaps we should invoke parse only if we actually need to parse the query statement and simply instantiate a single-branch rule otherwise
func ParseDetection(s Detection) (*Tree, error) {
	if s == nil {
		return nil, ErrMissingDetection{}
	}
	if len(s) < 3 {
		return parseSimpleScenario(s)
	}
	return parseComplexScenario(s)
}

func parseSimpleScenario(s Detection) (*Tree, error) {
	switch len(s) {
	case 1:
		// Simple case - should have only one search field, but should not have a condition field
		if c, ok := s["condition"].(string); ok {
			return nil, ErrIncompleteDetection{Condition: c}
		}
	case 2:
		// Simple case - one condition statement comprised of single IDENT that matches the second field name
		if c, ok := s["condition"].(string); !ok {
			return nil, ErrIncompleteDetection{Condition: "MISSING"}
		} else {
			if _, ok := s[c]; !ok {
				return nil, ErrIncompleteDetection{
					Condition: c,
					Msg:       fmt.Sprintf("Field %s defined in condition missing from map.", c),
					Keys:      s.FieldSlice(),
				}
			}
		}
		delete(s, "condition")
	default:
		return nil, ErrMissingDetection{}
	}
	rx := s.Fields()
	ast := &Tree{}
	r := <-rx
	root, err := newRuleMatcherFromIdent(&r, false)
	if err != nil {
		return nil, err
	}
	ast.Root = root
	return ast, nil
}

func parseComplexScenario(s Detection) (*Tree, error) {
	// Complex case, time to build syntax tree out of condition statement
	raw, ok := s["condition"].(string)
	if !ok {
		return nil, ErrMissingCondition{}
	}
	p := &parser{
		lex:       lex(raw),
		sigma:     s,
		tokens:    make([]Item, 0),
		previous:  TokBegin,
		condition: raw,
	}
	if err := p.run(); err != nil {
		return nil, err
	}
	return &Tree{Root: p.result}, nil
}

func newRuleMatcherFromIdent(v *SearchExpr, toLower bool) (Branch, error) {
	if v == nil {
		return nil, fmt.Errorf("Missing rule search expression")
	}
	switch v.Type {
	case ExprKeywords:
		return NewKeywordFromInterface(toLower, v.Content)
	case ExprSelection:
		switch m := v.Content.(type) {
		case map[string]interface{}:
			return NewFields(m, toLower, false)
		case []interface{}:
			// might be a list of selections where each entry is a distinct selection rule joined by logical OR
			branch := make(FieldsList, 0)
			for _, raw := range m {
				var (
					elem *Fields
					err  error
				)
				switch expr := raw.(type) {
				case map[interface{}]interface{}:
					m2, err := interfaceMapToStringInterfaceMap(expr)
					if err != nil {
						return nil, err
					}
					elem, err = NewFields(m2, toLower, false)
				case map[string]interface{}:
					elem, err = NewFields(expr, toLower, false)
				default:
					return nil, fmt.Errorf("Unhandled rule search expression type")
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
			return NewFields(m2, toLower, false)
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

func parseSearch(t tokens, detect Detection, c RuleConfig, entry bool) (Branch, error) {
	if t.contains(IdentifierAll) {
		return nil, ErrUnsupportedToken{Msg: IdentifierAll.Literal()}
	}
	if t.contains(IdentifierWithWildcard) {
		return nil, ErrUnsupportedToken{Msg: IdentifierWithWildcard.Literal()}
	}
	if t.contains(StOne) || t.contains(StAll) {
		return nil, ErrUnsupportedToken{Msg: fmt.Sprintf("%s / %s", StAll.Literal(), StOne.Literal())}
	}

	rules := t.splitByToken(KeywordOr)

	branch := make([]Branch, 0)
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

	return NodeSimpleOr(branch), nil
}

func newBranchFromGroup(group tokensHandler, detect Detection, c RuleConfig) (Branch, error) {

	// recursion here
	if group.hasSubGroup {
		branch := make([]Branch, 0)
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
						branch = append(branch, func() Branch {
							if i > 0 && regular[i-1:].isNegated() {
								return NodeNot{Branch: r}
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
			branch = append(branch, func() Branch {
				if pos.From > 0 && group.tokens[pos.From-1].T == KeywordNot {
					return NodeNot{Branch: b}
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
							branch = append(branch, func() Branch {
								if i > 0 && regular[i-1:].isNegated() {
									return NodeNot{Branch: r}
								}
								return r
							}())
						}
					}
				}
			}
		}
		return NodeSimpleAnd(branch), nil
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
		return func() Branch {
			if group.isNegated() {
				return NodeNot{Branch: r}
			}
			return r
		}(), err
	}

	// TODO - handle everything in this function
	return parseSimpleSearch(group.tokens, detect, c)
}

// simple search == just a valid group sequence with no sub-groups
func parseSimpleSearch(t tokens, detect Detection, c RuleConfig) (Branch, error) {
	rules := t.splitByToken(KeywordOr)

	branch := make([]Branch, 0)
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
			branch = append(branch, func() Branch {
				if group.isNegated() {
					return NodeNot{Branch: r}
				}
				return r
			}())
		} else {
			andGroup := make([]Branch, 0)
			for i, item := range group.tokens {
				switch item.T {
				case Identifier:
					r, err := newRuleMatcherFromIdent(detect.Get(item.Val), c.LowerCase)
					if err != nil {
						return nil, err
					}
					andGroup = append(andGroup, func() Branch {
						if i > 0 && group.tokens[i-1:].isNegated() {
							return NodeNot{Branch: r}
						}
						return r
					}())
				}
			}
			branch = append(branch, NodeSimpleAnd(andGroup))
		}
	}
	if len(branch) == 1 {
		return branch[0], nil
	}

	return NodeSimpleOr(branch), nil
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
	sigma Detection

	// for debug
	condition string

	// resulting rule that can be collected later
	result Branch
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
	b, err := parseSearch(p.tokens, p.sigma, RuleConfig{}, true)
	if err != nil {
		return err
	}
	p.result = b
	return nil
}

func (p *parser) collectAndValidateTokenSequences() error {
	for item := range p.lex.items {

		if item.T == TokUnsupp {
			return ErrUnsupportedToken{Msg: item.Val}
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

func interfaceMapToStringInterfaceMap(m map[interface{}]interface{}) (map[string]interface{}, error) {
	m2 := make(map[string]interface{})
	for k, v := range m {
		sk, ok := k.(string)
		if !ok {
			return m2, fmt.Errorf("failed to create selection rule from interface")
		}
		m2[sk] = v
	}
	return m2, nil
}

func isKeywords(s string) bool { return strings.HasPrefix(s, "keywords") }

func checkKeyWord(in string) Token {
	if len(in) == 0 {
		return TokNil
	}
	switch strings.ToLower(in) {
	case KeywordAnd.Literal():
		return KeywordAnd
	case KeywordOr.Literal():
		return KeywordOr
	case KeywordNot.Literal():
		return KeywordNot
	case "sum", "min", "max", "count", "avg":
		return KeywordAgg
	case IdentifierAll.Literal():
		return IdentifierAll
	default:
		if strings.Contains(in, "*") {
			return IdentifierWithWildcard
		}
		return Identifier
	}
}

var eof = rune(0)

type Token int

const (
	TokErr Token = iota

	// Helpers for internal stuff
	TokUnsupp
	TokBegin
	TokNil

	// user-defined word
	Identifier
	IdentifierWithWildcard
	IdentifierAll

	// Literals
	LitEof

	// Separators
	SepLpar
	SepRpar
	SepPipe

	// Operators
	OpEq
	OpGt
	OpGte
	OpLt
	OpLte

	// Keywords
	KeywordAnd
	KeywordOr
	KeywordNot
	KeywordAgg

	// TODO
	KeywordNear
	KeywordBy

	// Statements
	StOne
	StAll
)

func (t Token) String() string {
	switch t {
	case Identifier, IdentifierWithWildcard:
		return "IDENT"
	case IdentifierAll:
		return "THEM"
	case SepLpar:
		return "LPAR"
	case SepRpar:
		return "RPAR"
	case SepPipe:
		return "PIPE"
	case OpEq:
		return "EQ"
	case OpGt:
		return "GT"
	case OpGte:
		return "GTE"
	case OpLt:
		return "LT"
	case OpLte:
		return "LTE"
	case KeywordAnd:
		return "AND"
	case KeywordOr:
		return "OR"
	case KeywordNot:
		return "NOT"
	case StAll:
		return "ALL"
	case StOne:
		return "ONE"
	case KeywordAgg:
		return "AGG"
	case LitEof:
		return "EOF"
	case TokErr:
		return "ERR"
	case TokUnsupp:
		return "UNSUPPORTED"
	case TokBegin:
		return "BEGINNING"
	case TokNil:
		return "NIL"
	default:
		return "Unk"
	}
}

func (t Token) Literal() string {
	switch t {
	case Identifier, IdentifierWithWildcard:
		return "keywords"
	case IdentifierAll:
		return "them"
	case SepLpar:
		return "("
	case SepRpar:
		return ")"
	case SepPipe:
		return "|"
	case OpEq:
		return "="
	case OpGt:
		return ">"
	case OpGte:
		return ">="
	case OpLt:
		return "<"
	case OpLte:
		return "<="
	case KeywordAnd:
		return "and"
	case KeywordOr:
		return "or"
	case KeywordNot:
		return "not"
	case StAll:
		return "all of"
	case StOne:
		return "1 of"
	case LitEof, TokNil:
		return ""
	default:
		return "Err"
	}
}

func (t Token) Rune() rune {
	switch t {
	case SepLpar:
		return '('
	case SepRpar:
		return ')'
	case SepPipe:
		return '|'
	default:
		return eof
	}
}

// detect invalid token sequences
func validTokenSequence(t1, t2 Token) bool {
	switch t2 {
	case StAll, StOne:
		switch t1 {
		case TokBegin, SepLpar, KeywordAnd, KeywordOr, KeywordNot:
			return true
		}
	case IdentifierAll:
		switch t1 {
		case StAll, StOne:
			return true
		}
	case Identifier, IdentifierWithWildcard:
		switch t1 {
		case SepLpar, TokBegin, KeywordAnd, KeywordOr, KeywordNot, StOne, StAll:
			return true
		}
	case KeywordAnd, KeywordOr:
		switch t1 {
		case Identifier, IdentifierAll, IdentifierWithWildcard, SepRpar:
			return true
		}
	case KeywordNot:
		switch t1 {
		case KeywordAnd, KeywordOr, SepLpar, TokBegin:
			return true
		}
	case SepLpar:
		switch t1 {
		case KeywordAnd, KeywordOr, KeywordNot, TokBegin, SepLpar:
			return true
		}
	case SepRpar:
		switch t1 {
		case Identifier, IdentifierAll, IdentifierWithWildcard, SepLpar, SepRpar:
			return true
		}
	case LitEof:
		switch t1 {
		case Identifier, IdentifierAll, IdentifierWithWildcard, SepRpar:
			return true
		}
	case SepPipe:
		switch t1 {
		case Identifier, IdentifierAll, IdentifierWithWildcard, SepRpar:
			return true
		}
	}
	return false
}

type tokensHandler struct {
	tokens
	hasSubGroup bool
	subGroups   []offsets
	err         error
}

func (t *tokensHandler) discoverSubGroups() *tokensHandler {
	groups, ok, err := newGroupOffsetInTokens(t.tokens)
	if err != nil {
		t.err = err
	}
	if ok {
		t.hasSubGroup = true
		t.subGroups = groups
	} else {
		t.hasSubGroup = false
		t.subGroups = make([]offsets, 0)
	}
	return t
}

func (t tokensHandler) isSimpleIdent() bool {
	l := len(t.tokens)
	return l == 1 || (l == 2 && t.isNegated())
}

type tokens []Item

func (t tokens) splitByToken(tok Token) []*tokensHandler {

	if !t.contains(KeywordOr) {
		return []*tokensHandler{&tokensHandler{tokens: t}}
	}

	var start, groupBalance int

	//rules := &tokensHandler{ tokens: make(tokens, 0), }
	rules := make([]*tokensHandler, 0)
	last := len(t) - 1

	for pos, item := range t {

		var hasSubGroup bool
		switch v := item.T; {
		case v == SepLpar:
			groupBalance++
		case v == SepRpar:
			groupBalance--
			if groupBalance == 0 {
				hasSubGroup = true
			}
		}

		if (item.T == tok && groupBalance == 0) || pos == last {
			switch pos {
			case last:
				rules = append(rules, func() *tokensHandler {
					if last > 0 && t[pos-1].T == KeywordNot {
						return &tokensHandler{
							tokens:      t[pos-1:],
							hasSubGroup: hasSubGroup,
						}
					}
					return &tokensHandler{
						tokens:      t[start:],
						hasSubGroup: hasSubGroup,
					}
				}().discoverSubGroups())
			default:
				g := &tokensHandler{
					tokens:      t[start:pos],
					hasSubGroup: hasSubGroup,
				}
				rules = append(rules, g.discoverSubGroups())
				start = pos + 1
			}
		}

	}
	return rules
}

func (t tokens) len() int { return len(t) }
func (t tokens) lastIdx() int {
	return t.len() - 1
}
func (t tokens) tail(i int) tokens {
	if i < 0 || i > t.lastIdx() {
		return t
	}
	return t[i:]
}

func (t tokens) head(i int) tokens {
	if i < 0 || i > t.lastIdx() {
		return t
	}
	return t[:i]
}

func (t tokens) nextKeyword() (int, Token) {
	for i, item := range t {
		if item.T == KeywordAnd || item.T == KeywordOr {
			return i, item.T
		}
	}
	return -1, TokNil
}

func (t tokens) getTokenIndices(tok Token) []int {
	out := make([]int, 0)
	for i, item := range t {
		if item.T == tok {
			out = append(out, i)
		}
	}
	return out
}

func (t tokens) countAnd() int {
	var c int
	for _, item := range t {
		if item.T == KeywordAnd {
			c++
		}
	}
	return c
}

func (t tokens) countOr() int {
	var c int
	for _, item := range t {
		if item.T == KeywordOr {
			c++
		}
	}
	return c
}

func (t tokens) count(tok ...Token) []int {
	c := make([]int, len(t))
	for _, item := range t {
		for i, token := range tok {
			if item.T == token {
				c[i]++
			}
		}
	}
	return c
}

func (t tokens) isNegated() bool {
	if len(t) > 1 && t[0].T == KeywordNot {
		return true
	}
	return false
}

func (t tokens) index(tok Token) int {
	for i, item := range t {
		if item.T == tok {
			return i
		}
	}
	return -1
}

func (t tokens) reverseIndex(tok Token) int {
	for i := len(t) - 1; i > 0; i-- {
		if t[i].T == tok {
			return i
		}
	}
	return -1
}

func (t tokens) contains(tok Token) bool {
	for _, item := range t {
		if item.T == tok {
			return true
		}
	}
	return false
}

type offsets struct {
	From, To int
}

func (o *offsets) SetFrom(i int) *offsets {
	o.From = i
	return o
}
func (o *offsets) SetTo(i int) *offsets {
	o.To = i
	return o
}

func newGroupOffsetInTokens(t tokens) ([]offsets, bool, error) {
	if t == nil || t.len() == 0 {
		return nil, false, nil
	}
	if !t.contains(SepLpar) {
		return nil, false, nil
	}
	groups := make([]offsets, 0)
	var balance, found int
	for i, item := range t {
		switch item.T {
		case SepLpar:
			if balance == 0 {
				groups = append(groups, offsets{From: i, To: -1})
			}
			balance++
		case SepRpar:
			balance--
			if balance == 0 {
				groups[found].SetTo(i + 1)
				found++
			}
		}
	}
	if balance > 0 || balance < 0 {
		return groups, false, fmt.Errorf("Broken rule group")
	}
	return groups, true, nil
}

type lexer struct {
	input    string    // we'll store the string being parsed
	start    int       // the position we started scanning
	position int       // the current position of our scan
	width    int       // we'll be using runes which can be double byte
	items    chan Item // the channel we'll use to communicate between the lexer and the parser
}

// lex creates a lexer and starts scanning the provided input.
func lex(input string) *lexer {
	l := &lexer{
		input: input,
		items: make(chan Item, 0),
	}
	go l.scan()
	return l
}

// ignore resets the start position to the current scan position effectively
// ignoring any input.
func (l *lexer) ignore() {
	l.start = l.position
}

// next advances the lexer state to the next rune.
func (l *lexer) next() (r rune) {
	if l.position >= len(l.input) {
		l.width = 0
		return eof
	}

	r, l.width = utf8.DecodeRuneInString(l.todo())
	l.position += l.width
	return r
}

// backup allows us to step back one rune which is helpful when you've crossed
// a boundary from one state to another.
func (l *lexer) backup() {
	l.position = l.position - 1
}

// scan will step through the provided text and execute state functions as
// state changes are observed in the provided input.
func (l *lexer) scan() {
	// When we begin processing, let's assume we're going to process text.
	// One state function will return another until `nil` is returned to signal
	// the end of our process.
	for fn := lexCondition; fn != nil; {
		fn = fn(l)
	}
	close(l.items)
}

func (l *lexer) unsuppf(format string, args ...interface{}) stateFn {
	msg := fmt.Sprintf(format, args...)
	l.items <- Item{T: TokUnsupp, Val: msg}
	return nil
}

func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	msg := fmt.Sprintf(format, args...)
	l.items <- Item{T: TokErr, Val: msg}
	return nil
}

// emit sends a item over the channel so the parser can collect and manage
// each segment.
func (l *lexer) emit(k Token) {
	i := Item{T: k, Val: l.input[l.start:l.position]}
	l.items <- i
	l.ignore() // reset our scanner now that we've dispatched a segment
}

func (l lexer) collected() string { return l.input[l.start:l.position] }
func (l lexer) todo() string      { return l.input[l.position:] }

// stateFn is a function that is specific to a state within the string.
type stateFn func(*lexer) stateFn

// lexCondition scans what is expected to be text.
func lexCondition(l *lexer) stateFn {
	for {
		// TODO - run these cheks only if we have accumulated a word, not on every char
		if strings.HasPrefix(l.todo(), StOne.Literal()) {
			return lexOneOf
		}
		if strings.HasPrefix(l.todo(), StAll.Literal()) {
			return lexAllOf
		}
		switch r := l.next(); {
		case r == eof:
			return lexEOF
		case r == SepRpar.Rune():
			return lexRparWithTokens
		case r == SepLpar.Rune():
			return lexLpar
		case r == SepPipe.Rune():
			return lexPipe
		case unicode.IsSpace(r):
			return lexAccumulateBeforeWhitespace
		}
	}
}

func lexStatement(l *lexer) stateFn {
	return lexCondition
}

func lexOneOf(l *lexer) stateFn {
	l.position += len(StOne.Literal())
	l.emit(StOne)
	return lexCondition
}

func lexAllOf(l *lexer) stateFn {
	l.position += len(StAll.Literal())
	l.emit(StAll)
	return lexCondition
}

func lexAggs(l *lexer) stateFn {
	return l.unsuppf("aggregation not supported yet [%s]", l.input)
}

func lexEOF(l *lexer) stateFn {
	if l.position > l.start {
		l.emit(checkKeyWord(l.collected()))
	}
	l.emit(LitEof)
	return nil
}

func lexPipe(l *lexer) stateFn {
	l.emit(SepPipe)
	return lexAggs
}

func lexLpar(l *lexer) stateFn {
	l.emit(SepLpar)
	return lexCondition
}

func lexRparWithTokens(l *lexer) stateFn {
	// emit any text we've accumulated.
	if l.position > l.start {
		l.backup()
		// There may be N whitespace chars between token RPAR
		// TODO - may be a more concise way to do this, right now loops like this are everywhere

		if t := checkKeyWord(l.collected()); t != TokNil {
			l.emit(t)
		}

		for {
			switch r := l.next(); {
			case r == eof:
				return lexEOF
			case unicode.IsSpace(r):
				l.ignore()
			default:
				return lexRpar
			}
		}
	}
	return lexRpar
}

func lexRpar(l *lexer) stateFn {
	l.emit(SepRpar)
	return lexCondition
}

func lexAccumulateBeforeWhitespace(l *lexer) stateFn {
	l.backup()
	// emit any text we've accumulated.
	if l.position > l.start {
		l.emit(checkKeyWord(l.collected()))
	}
	return lexWhitespace
}

// lexWhitespace scans what is expected to be whitespace.
func lexWhitespace(l *lexer) stateFn {
	for {
		switch r := l.next(); {
		case r == eof:
			return lexEOF
		case !unicode.IsSpace(r):
			l.backup()
			return lexCondition
		default:
			l.ignore()
		}
	}
}
