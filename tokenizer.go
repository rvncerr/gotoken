package gotoken

import (
	"unicode"

	"github.com/rvncerr/gocircular"
)

// RuneClass categorizes Unicode runes into character classes.
type RuneClass int

const (
	classUndef  RuneClass = -1
	ClassLetter RuneClass = iota
	ClassDigit
	ClassPunct
	ClassOther
)

// TokenInfo contains metadata about a detected token.
type TokenInfo struct {
	// Language is the detected language index, or -1 if unknown.
	Language int

	// Base contains byte offsets [start, end] of the language-relevant portion.
	Base [2]int
}

// Tokenizer implements the SmartToken tokenization algorithm.
// It generates sub-tokens with language detection for multilingual text.
type Tokenizer struct {
	languages []*unicode.RangeTable
	policy    Policy

	// State for processing
	prevLangIndex int
	currLangIndex int
	prevClass     RuneClass
	currClass     RuneClass
}

// Option configures a Tokenizer.
type Option func(*Tokenizer)

// WithPolicy sets the tokenization depth policy.
func WithPolicy(p Policy) Option {
	return func(t *Tokenizer) {
		t.policy = p
	}
}

// WithLanguage adds a Unicode range table for language detection.
// The order of added languages determines their index (0, 1, 2, ...).
func WithLanguage(rt *unicode.RangeTable) Option {
	return func(t *Tokenizer) {
		t.languages = append(t.languages, rt)
	}
}

// WithLanguages adds multiple Unicode range tables for language detection.
func WithLanguages(tables ...*unicode.RangeTable) Option {
	return func(t *Tokenizer) {
		t.languages = append(t.languages, tables...)
	}
}

// New creates a new Tokenizer with the given options.
func New(opts ...Option) *Tokenizer {
	t := &Tokenizer{
		policy:        NewLinearPolicy(10, 10, 18, 2),
		prevLangIndex: -1,
		currLangIndex: -1,
		prevClass:     classUndef,
		currClass:     classUndef,
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// Tokenize processes the input string and returns all detected tokens with their info.
func (t *Tokenizer) Tokenize(source string) map[string]TokenInfo {
	tokens := make(map[string]TokenInfo)

	const (
		stateSpace = iota
		stateToken
	)

	var offset int
	state := stateSpace

	for i, r := range source {
		switch state {
		case stateSpace:
			if !unicode.IsSpace(r) {
				state = stateToken
				offset = i
			}
		case stateToken:
			if unicode.IsSpace(r) {
				state = stateSpace
				t.extractSubtokens(source[offset:i], tokens)
			}
		}
	}

	if state == stateToken {
		t.extractSubtokens(source[offset:], tokens)
	}

	return tokens
}

func (t *Tokenizer) extractSubtokens(token string, tokens map[string]TokenInfo) {
	// First pass: count segments
	segments := t.countSegments(token)
	if segments == 0 {
		return
	}

	// Get depth from policy based on segment count
	policyDepth := t.policy.Depth(segments)

	// Depth 0 means only whole token (no subtokens)
	if policyDepth == 0 {
		t.reset()
		for _, r := range token {
			t.advanceRune(r)
		}
		lang := -1
		if t.currClass == ClassLetter {
			lang = t.currLangIndex
		}
		tokens[token] = TokenInfo{
			Language: lang,
			Base:     [2]int{0, len(token)},
		}
		return
	}

	depth := policyDepth + 1
	if depth > segments+1 {
		depth = segments + 1
	}

	// Second pass: extract subtokens
	t.reset()
	positions := gocircular.New[int](depth)
	classes := gocircular.New[RuneClass](depth - 1)
	langIndices := gocircular.New[int](depth - 1)

	for i, r := range token {
		if t.advanceRune(r) {
			positions.PushBack(i)
			if t.prevClass != classUndef {
				classes.PushBack(t.prevClass)
				if t.prevClass == ClassLetter {
					langIndices.PushBack(t.prevLangIndex)
				} else {
					langIndices.PushBack(-1)
				}
			}
			if positions.Full() {
				t.emitTokens(token, positions, classes, langIndices, tokens)
			}
		}
	}

	// Final position
	positions.PushBack(len(token))
	classes.PushBack(t.currClass)
	if t.currClass == ClassLetter {
		langIndices.PushBack(t.currLangIndex)
	} else {
		langIndices.PushBack(-1)
	}

	// Drain remaining
	for !positions.Empty() {
		t.emitTokens(token, positions, classes, langIndices, tokens)
		positions.PopFront()
		classes.PopFront()
		langIndices.PopFront()
	}
}

func (t *Tokenizer) countSegments(token string) int {
	t.reset()
	count := 0
	for _, r := range token {
		if t.advanceRune(r) {
			count++
		}
	}
	if t.currClass != classUndef {
		count++ // Final segment
	}
	return count
}

func (t *Tokenizer) emitTokens(
	token string,
	positions *gocircular.Buffer[int],
	classes *gocircular.Buffer[RuneClass],
	langIndices *gocircular.Buffer[int],
	tokens map[string]TokenInfo,
) {
	posSlice := positions.ToSlice()
	classSlice := classes.ToSlice()
	langSlice := langIndices.ToSlice()

	left, _ := positions.Front()
	for depth, right := range posSlice[1:] {
		lang, base := t.detectLanguage(posSlice, classSlice[:depth+1], langSlice[:depth+1])
		tokens[token[left:right]] = TokenInfo{
			Language: lang,
			Base:     base,
		}
	}
}

func (t *Tokenizer) detectLanguage(positions []int, classes []RuneClass, langs []int) (int, [2]int) {
	base := func(left, right int) [2]int {
		return [2]int{positions[left] - positions[0], positions[right] - positions[0]}
	}

	n := len(classes)
	switch n {
	case 1:
		if classes[0] == ClassLetter {
			return langs[0], base(0, 1)
		}
		return -1, [2]int{}

	case 2:
		c0, c1 := classes[0], classes[1]
		switch {
		case c0 == ClassLetter && c1 == ClassLetter:
			return langs[1], base(0, 2)
		case c0 == ClassLetter:
			return langs[0], base(0, 1)
		case c1 == ClassLetter:
			return langs[1], base(1, 2)
		}
		return -1, [2]int{}

	case 3:
		c0, c1, c2 := classes[0], classes[1], classes[2]
		switch {
		case c0 == ClassLetter && c1 == ClassLetter && c2 == ClassLetter:
			return -1, [2]int{}
		case c0 == ClassLetter && c1 == ClassLetter:
			return langs[1], base(0, 2)
		case c0 == ClassLetter && c2 == ClassLetter:
			if langs[0] == langs[2] {
				return langs[2], base(0, 3)
			}
			return langs[2], base(2, 3)
		case c1 == ClassLetter && c2 == ClassLetter:
			return langs[2], base(2, 3)
		case c0 == ClassLetter:
			return langs[0], base(0, 1)
		case c1 == ClassLetter:
			return langs[1], base(1, 2)
		case c2 == ClassLetter:
			return langs[2], base(2, 3)
		}
		return -1, [2]int{}
	}

	return -1, [2]int{}
}

func (t *Tokenizer) reset() {
	t.prevClass = classUndef
	t.currClass = classUndef
	t.prevLangIndex = -1
	t.currLangIndex = -1
}

func (t *Tokenizer) advanceRune(r rune) bool {
	newClass := t.classifyRune(r)

	if newClass == ClassLetter {
		newLang := t.detectLangIndex(r)
		if newLang != t.currLangIndex {
			t.prevLangIndex = t.currLangIndex
			t.currLangIndex = newLang
			t.prevClass = t.currClass
			t.currClass = newClass
			return true
		}
		if newClass != t.currClass {
			t.prevLangIndex = t.currLangIndex
			t.currLangIndex = -1
			t.prevClass = t.currClass
			t.currClass = newClass
			return true
		}
	} else if newClass != t.currClass {
		t.prevLangIndex = t.currLangIndex
		t.currLangIndex = -1
		t.prevClass = t.currClass
		t.currClass = newClass
		return true
	}

	return false
}

func (t *Tokenizer) classifyRune(r rune) RuneClass {
	switch {
	case unicode.IsLetter(r):
		return ClassLetter
	case unicode.IsDigit(r):
		return ClassDigit
	case unicode.IsPunct(r):
		return ClassPunct
	default:
		return ClassOther
	}
}

func (t *Tokenizer) detectLangIndex(r rune) int {
	for i, rt := range t.languages {
		if unicode.In(r, rt) {
			return i
		}
	}
	return -1
}

