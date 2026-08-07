package gotoken

import (
	"unicode"

	"github.com/rvncerr/gocircular"
)

// RuneClass categorizes Unicode runes into character classes.
//
// Classes are numbered from 1. Zero is reserved as "no class assigned", so a
// default-initialized RuneClass never reads as a real classification.
type RuneClass int

const (
	classUndef  RuneClass = iota // 0: no class assigned yet
	ClassLetter                  // 1
	ClassDigit                   // 2
	ClassPunct                   // 3
	ClassOther                   // 4
)

// TokenInfo contains metadata about a detected token.
type TokenInfo struct {
	// Language is the detected language index, or -1 if unknown or
	// undetermined.
	Language int

	// Base contains half-open byte offsets [start, end) of the
	// language-relevant portion, relative to the start of the (sub-)token, so
	// subtoken[Base[0]:Base[1]] slices it directly.
	//
	// The zero value [0, 0] is a sentinel meaning "undetermined": it is used
	// for non-letter tokens and for sub-tokens spanning more than
	// maxAnalyzedSegments segments, where no single language span is
	// identified. This holds on every path, including the whole-token-only one
	// taken when a policy reports a depth of zero.
	Base [2]int
}

// Tokenizer implements the SmartToken tokenization algorithm.
// It generates sub-tokens with language detection for multilingual text.
//
// A Tokenizer is immutable once constructed, and each call to Tokenize uses its
// own scanning state, so it is safe for concurrent use by multiple goroutines
// provided the configured Policy is too. All policies in this package are.
//
// The map returned by Tokenize is not shared and carries no such restriction.
type Tokenizer struct {
	languages []*unicode.RangeTable
	policy    Policy
}

// Option configures a Tokenizer.
type Option func(*Tokenizer)

// WithPolicy sets the tokenization depth policy.
//
// A nil Policy selects the default. That covers a nil interface only — an
// interface holding a nil pointer, as in WithPolicy(p) for a nil *FixedPolicy
// p, is itself non-nil and will fault on the first Depth call.
func WithPolicy(p Policy) Option {
	return func(t *Tokenizer) {
		t.policy = p
	}
}

// WithLanguage adds a Unicode range table for language detection.
// The order of added languages determines their index (0, 1, 2, ...).
// A nil range table is ignored.
func WithLanguage(rt *unicode.RangeTable) Option {
	return func(t *Tokenizer) {
		if rt != nil {
			t.languages = append(t.languages, rt)
		}
	}
}

// WithLanguages adds multiple Unicode range tables for language detection.
// Nil range tables are ignored.
func WithLanguages(tables ...*unicode.RangeTable) Option {
	return func(t *Tokenizer) {
		for _, rt := range tables {
			if rt != nil {
				t.languages = append(t.languages, rt)
			}
		}
	}
}

// defaultPolicy is used when no policy is configured (or a nil policy is set).
// LinearPolicy is immutable, so one shared instance serves every Tokenizer.
var defaultPolicy Policy = NewLinearPolicy(10, 10, 18, 2)

// New creates a new Tokenizer with the given options.
func New(opts ...Option) *Tokenizer {
	t := &Tokenizer{}
	for _, opt := range opts {
		opt(t)
	}
	if t.policy == nil {
		t.policy = defaultPolicy
	}
	return t
}

// Tokenize processes the input string and returns all detected tokens with
// their info.
//
// The result is keyed by sub-token text, so identical sub-token strings are
// deduplicated and the last occurrence wins. Map keys are sub-slices of
// source, so the returned map keeps source's backing storage alive.
func (t *Tokenizer) Tokenize(source string) map[string]TokenInfo {
	tokens := make(map[string]TokenInfo)
	s := newScanner(t.languages, t.policy)

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
				s.extractSubtokens(source[offset:i], tokens)
			}
		}
	}

	if state == stateToken {
		s.extractSubtokens(source[offset:], tokens)
	}

	return tokens
}

// scanner holds the per-call state used to tokenize a single source string.
// A fresh scanner is created for every Tokenize call so that Tokenizer itself
// remains immutable and safe for concurrent use.
type scanner struct {
	languages []*unicode.RangeTable
	policy    Policy

	prevLangIndex int
	currLangIndex int
	prevClass     RuneClass
	currClass     RuneClass
}

func newScanner(languages []*unicode.RangeTable, policy Policy) *scanner {
	return &scanner{
		languages:     languages,
		policy:        policy,
		prevLangIndex: -1,
		currLangIndex: -1,
		prevClass:     classUndef,
		currClass:     classUndef,
	}
}

func (s *scanner) extractSubtokens(token string, tokens map[string]TokenInfo) {
	// First pass: count segments
	segments := s.countSegments(token)
	if segments == 0 {
		return
	}

	// Get depth from policy based on segment count.
	policyDepth := s.policy.Depth(segments)

	// A non-positive depth means "whole token only" (no sub-tokens). Treating
	// negative values the same as zero keeps a misconfigured policy from
	// driving the buffer capacity below 1, which would panic.
	if policyDepth <= 0 {
		lang, base := s.wholeTokenInfo(token, segments)
		tokens[token] = TokenInfo{Language: lang, Base: base}
		return
	}

	// The window is sized in position markers, of which a token with N segments
	// has N+1: one per segment start plus the end of the token. A wider window
	// than that would never fill.
	if policyDepth > segments {
		policyDepth = segments
	}
	depth := policyDepth + 1

	// Second pass: extract subtokens
	s.reset()
	positions := gocircular.New[int](depth)
	classes := gocircular.New[RuneClass](depth - 1)
	langIndices := gocircular.New[int](depth - 1)

	for i, r := range token {
		if s.advanceRune(r) {
			positions.PushBack(i)
			if s.prevClass != classUndef {
				classes.PushBack(s.prevClass)
				if s.prevClass == ClassLetter {
					langIndices.PushBack(s.prevLangIndex)
				} else {
					langIndices.PushBack(-1)
				}
			}
			if positions.Full() {
				s.emitTokens(token, positions, classes, langIndices, tokens)
			}
		}
	}

	// Final position
	positions.PushBack(len(token))
	classes.PushBack(s.currClass)
	if s.currClass == ClassLetter {
		langIndices.PushBack(s.currLangIndex)
	} else {
		langIndices.PushBack(-1)
	}

	// Drain remaining
	for !positions.Empty() {
		s.emitTokens(token, positions, classes, langIndices, tokens)
		positions.PopFront()
		classes.PopFront()
		langIndices.PopFront()
	}

	// Guarantee the whole token is always present. When the policy depth is
	// smaller than the segment count the sliding window never spans the entire
	// token, so it would otherwise be missing. wholeTokenInfo produces the same
	// result the window would have, keeping the value consistent either way.
	if _, ok := tokens[token]; !ok {
		lang, base := s.wholeTokenInfo(token, segments)
		tokens[token] = TokenInfo{Language: lang, Base: base}
	}
}

// countSegments returns the number of segments in token, where a segment is a
// contiguous run of runes sharing a character class and, for letters, a
// detected language.
//
// advanceRune reports true exactly once per segment — at the rune that starts
// it, including the first rune of the token — so counting those reports counts
// the segments. Empty tokens yield 0.
func (s *scanner) countSegments(token string) int {
	s.reset()
	count := 0
	for _, r := range token {
		if s.advanceRune(r) {
			count++
		}
	}
	return count
}

func (s *scanner) emitTokens(
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
		lang, base := s.detectLanguage(posSlice, classSlice[:depth+1], langSlice[:depth+1])
		tokens[token[left:right]] = TokenInfo{
			Language: lang,
			Base:     base,
		}
	}
}

// wholeTokenInfo computes the language and base for the entire token using the
// same segment summary the sliding window would build. It is used both for the
// "whole token only" policy path and to backfill the whole token when the
// window did not span it. segments must be the token's segment count, as
// returned by countSegments.
func (s *scanner) wholeTokenInfo(token string, segments int) (int, [2]int) {
	// A token spanning more segments than detectLanguage resolves is
	// undetermined no matter what its summary contains, so don't build one.
	// This is the common case for long tokens, where the summary would be the
	// bulk of the work.
	if segments < 1 || segments > maxAnalyzedSegments {
		return -1, [2]int{}
	}

	s.reset()

	// Bounded by the guard above, so these stay off the heap.
	var (
		positions = make([]int, 0, maxAnalyzedSegments+1)
		classes   = make([]RuneClass, 0, maxAnalyzedSegments)
		langs     = make([]int, 0, maxAnalyzedSegments)
	)

	for i, r := range token {
		if s.advanceRune(r) {
			positions = append(positions, i)
			if s.prevClass != classUndef {
				classes = append(classes, s.prevClass)
				if s.prevClass == ClassLetter {
					langs = append(langs, s.prevLangIndex)
				} else {
					langs = append(langs, -1)
				}
			}
		}
	}

	positions = append(positions, len(token))
	classes = append(classes, s.currClass)
	if s.currClass == ClassLetter {
		langs = append(langs, s.currLangIndex)
	} else {
		langs = append(langs, -1)
	}

	return s.detectLanguage(positions, classes, langs)
}

// maxAnalyzedSegments is the widest span detectLanguage resolves. Keep it in
// step with the cases below: a span wider than this has no single language
// portion, so it reports "undetermined".
const maxAnalyzedSegments = 3

func (s *scanner) detectLanguage(positions []int, classes []RuneClass, langs []int) (int, [2]int) {
	base := func(left, right int) [2]int {
		return [2]int{positions[left] - positions[0], positions[right] - positions[0]}
	}

	n := len(classes)
	if n > maxAnalyzedSegments {
		return -1, [2]int{}
	}

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
			return langs[2], base(1, 3)
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

func (s *scanner) reset() {
	s.prevClass = classUndef
	s.currClass = classUndef
	s.prevLangIndex = -1
	s.currLangIndex = -1
}

func (s *scanner) advanceRune(r rune) bool {
	newClass := s.classifyRune(r)

	if newClass == ClassLetter {
		newLang := s.detectLangIndex(r)
		if newLang != s.currLangIndex {
			s.prevLangIndex = s.currLangIndex
			s.currLangIndex = newLang
			s.prevClass = s.currClass
			s.currClass = newClass
			return true
		}
		if newClass != s.currClass {
			s.prevLangIndex = s.currLangIndex
			s.currLangIndex = -1
			s.prevClass = s.currClass
			s.currClass = newClass
			return true
		}
	} else if newClass != s.currClass {
		s.prevLangIndex = s.currLangIndex
		s.currLangIndex = -1
		s.prevClass = s.currClass
		s.currClass = newClass
		return true
	}

	return false
}

// classifyRune assigns a rune to a character class. Note that ClassDigit only
// covers decimal digits (unicode.IsDigit); other numerics such as Roman
// numerals or fractions fall into ClassOther.
func (s *scanner) classifyRune(r rune) RuneClass {
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

func (s *scanner) detectLangIndex(r rune) int {
	for i, rt := range s.languages {
		if unicode.In(r, rt) {
			return i
		}
	}
	return -1
}
