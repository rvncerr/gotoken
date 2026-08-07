package gotoken

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"sync"
	"testing"
	"unicode"
)

func newTestTokenizer() *Tokenizer {
	return New(
		WithPolicy(NewLinearPolicy(3, 10, 10, 2)),
		WithLanguages(unicode.Latin, unicode.Cyrillic),
	)
}

func TestTokenizer_Simple(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect map[string]TokenInfo
	}{
		{
			name:  "single word",
			input: "hello",
			expect: map[string]TokenInfo{
				"hello": {Language: 0, Base: [2]int{0, 5}},
			},
		},
		{
			name:  "two words",
			input: "hello world",
			expect: map[string]TokenInfo{
				"hello": {Language: 0, Base: [2]int{0, 5}},
				"world": {Language: 0, Base: [2]int{0, 5}},
			},
		},
	}

	tok := newTestTokenizer()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tok.Tokenize(tt.input)
			if !reflect.DeepEqual(got, tt.expect) {
				t.Errorf("Tokenize(%q) = %v, want %v", tt.input, got, tt.expect)
			}
		})
	}
}

func TestTokenizer_Transitions(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect map[string]TokenInfo
	}{
		{
			name:  "latin to cyrillic",
			input: "helloпривет",
			expect: map[string]TokenInfo{
				"hello":       {Language: 0, Base: [2]int{0, 5}},
				"привет":      {Language: 1, Base: [2]int{0, 12}},
				"helloпривет": {Language: 1, Base: [2]int{0, 17}},
			},
		},
		{
			name:  "latin to unknown",
			input: "hello你好",
			expect: map[string]TokenInfo{
				"hello":   {Language: 0, Base: [2]int{0, 5}},
				"你好":      {Language: -1, Base: [2]int{0, 6}},
				"hello你好": {Language: -1, Base: [2]int{0, 11}},
			},
		},
		{
			name:  "unknown to cyrillic",
			input: "你好привет",
			expect: map[string]TokenInfo{
				"你好":       {Language: -1, Base: [2]int{0, 6}},
				"привет":   {Language: 1, Base: [2]int{0, 12}},
				"你好привет": {Language: 1, Base: [2]int{0, 18}},
			},
		},
		{
			name:  "letter to digit",
			input: "hello123",
			expect: map[string]TokenInfo{
				"hello":    {Language: 0, Base: [2]int{0, 5}},
				"123":      {Language: -1, Base: [2]int{0, 0}},
				"hello123": {Language: 0, Base: [2]int{0, 5}},
			},
		},
		{
			name:  "letter to punct",
			input: "hello...",
			expect: map[string]TokenInfo{
				"hello":    {Language: 0, Base: [2]int{0, 5}},
				"...":      {Language: -1, Base: [2]int{0, 0}},
				"hello...": {Language: 0, Base: [2]int{0, 5}},
			},
		},
		{
			name:  "letter to symbol",
			input: "hello☭",
			expect: map[string]TokenInfo{
				"hello":  {Language: 0, Base: [2]int{0, 5}},
				"☭":      {Language: -1, Base: [2]int{0, 0}},
				"hello☭": {Language: 0, Base: [2]int{0, 5}},
			},
		},
		{
			name:  "digit to letter",
			input: "123hello",
			expect: map[string]TokenInfo{
				"123":      {Language: -1, Base: [2]int{0, 0}},
				"hello":    {Language: 0, Base: [2]int{0, 5}},
				"123hello": {Language: 0, Base: [2]int{3, 8}},
			},
		},
		{
			name:  "digit to punct",
			input: "123...",
			expect: map[string]TokenInfo{
				"123":    {Language: -1, Base: [2]int{0, 0}},
				"...":    {Language: -1, Base: [2]int{0, 0}},
				"123...": {Language: -1, Base: [2]int{0, 0}},
			},
		},
		{
			name:  "digit to symbol",
			input: "123☭",
			expect: map[string]TokenInfo{
				"123":  {Language: -1, Base: [2]int{0, 0}},
				"☭":    {Language: -1, Base: [2]int{0, 0}},
				"123☭": {Language: -1, Base: [2]int{0, 0}},
			},
		},
		{
			name:  "punct to letter",
			input: "...hello",
			expect: map[string]TokenInfo{
				"...":      {Language: -1, Base: [2]int{0, 0}},
				"hello":    {Language: 0, Base: [2]int{0, 5}},
				"...hello": {Language: 0, Base: [2]int{3, 8}},
			},
		},
		{
			name:  "punct to digit",
			input: "...123",
			expect: map[string]TokenInfo{
				"...":    {Language: -1, Base: [2]int{0, 0}},
				"123":    {Language: -1, Base: [2]int{0, 0}},
				"...123": {Language: -1, Base: [2]int{0, 0}},
			},
		},
		{
			name:  "punct to symbol",
			input: "...☭",
			expect: map[string]TokenInfo{
				"...":  {Language: -1, Base: [2]int{0, 0}},
				"☭":    {Language: -1, Base: [2]int{0, 0}},
				"...☭": {Language: -1, Base: [2]int{0, 0}},
			},
		},
		{
			name:  "symbol to letter",
			input: "☭hello",
			expect: map[string]TokenInfo{
				"☭":      {Language: -1, Base: [2]int{0, 0}},
				"hello":  {Language: 0, Base: [2]int{0, 5}},
				"☭hello": {Language: 0, Base: [2]int{3, 8}},
			},
		},
		{
			name:  "symbol to digit",
			input: "☭123",
			expect: map[string]TokenInfo{
				"☭":    {Language: -1, Base: [2]int{0, 0}},
				"123":  {Language: -1, Base: [2]int{0, 0}},
				"☭123": {Language: -1, Base: [2]int{0, 0}},
			},
		},
		{
			name:  "symbol to punct",
			input: "☭...",
			expect: map[string]TokenInfo{
				"☭":    {Language: -1, Base: [2]int{0, 0}},
				"...":  {Language: -1, Base: [2]int{0, 0}},
				"☭...": {Language: -1, Base: [2]int{0, 0}},
			},
		},
	}

	tok := newTestTokenizer()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tok.Tokenize(tt.input)
			if !reflect.DeepEqual(got, tt.expect) {
				t.Errorf("Tokenize(%q) = %v, want %v", tt.input, got, tt.expect)
			}
		})
	}
}

func TestTokenizer_Languages(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect map[string]TokenInfo
	}{
		{
			name:  "similar chars alternating",
			input: "aаaа", // Latin 'a' and Cyrillic 'а'
			expect: map[string]TokenInfo{
				"a":    {Language: 0, Base: [2]int{0, 1}},
				"а":    {Language: 1, Base: [2]int{0, 2}},
				"aа":   {Language: 1, Base: [2]int{0, 3}},
				"аa":   {Language: 0, Base: [2]int{0, 3}},
				"aаa":  {Language: -1, Base: [2]int{0, 0}},
				"аaа":  {Language: -1, Base: [2]int{0, 0}},
				"aаaа": {Language: -1, Base: [2]int{0, 0}},
			},
		},
		{
			name:  "chinese with punctuation",
			input: "你好。再见。",
			expect: map[string]TokenInfo{
				"你好":     {Language: -1, Base: [2]int{0, 6}},
				"再见":     {Language: -1, Base: [2]int{0, 6}},
				"。":      {Language: -1, Base: [2]int{0, 0}},
				"你好。":    {Language: -1, Base: [2]int{0, 6}},
				"再见。":    {Language: -1, Base: [2]int{0, 6}},
				"。再见":    {Language: -1, Base: [2]int{3, 9}},
				"你好。再见":  {Language: -1, Base: [2]int{0, 15}},
				"。再见。":   {Language: -1, Base: [2]int{3, 9}},
				"你好。再见。": {Language: -1, Base: [2]int{0, 0}},
			},
		},
	}

	tok := newTestTokenizer()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tok.Tokenize(tt.input)
			if !reflect.DeepEqual(got, tt.expect) {
				t.Errorf("Tokenize(%q) = %v, want %v", tt.input, got, tt.expect)
			}
		})
	}
}

func TestTokenizer_ThreeSegmentLetterBaseIsSymmetric(t *testing.T) {
	tok := New(
		WithPolicy(NewFixedPolicy(3)),
		WithLanguages(unicode.Latin, unicode.Cyrillic),
	)

	tests := []struct {
		input string
		want  TokenInfo
	}{
		{"aб.", TokenInfo{Language: 1, Base: [2]int{0, 3}}},
		{".aб", TokenInfo{Language: 1, Base: [2]int{1, 4}}},
	}

	for _, tt := range tests {
		if got := tok.Tokenize(tt.input)[tt.input]; got != tt.want {
			t.Errorf("Tokenize(%q)[%q] = %+v, want %+v", tt.input, tt.input, got, tt.want)
		}
	}
}

func TestTokenizer_CountPolicy(t *testing.T) {
	// Test with CountPolicy to verify depth limiting
	tests := []struct {
		name     string
		maxCount int
		input    string
		minLen   int // minimum number of tokens expected
		maxLen   int // maximum number of tokens expected
	}{
		{
			name:     "whole token only",
			maxCount: 1,
			input:    "aaa.bbb.ccc",
			minLen:   1,
			maxLen:   1,
		},
		{
			name:     "limited subtokens",
			maxCount: 10,
			input:    "a.b.c.d.e",
			minLen:   5,
			maxLen:   10,
		},
		{
			name:     "all subtokens",
			maxCount: 1000,
			input:    "a.b.c",
			minLen:   10,
			maxLen:   100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok := New(
				WithPolicy(NewCountPolicy(tt.maxCount)),
				WithLanguages(unicode.Latin),
			)
			got := tok.Tokenize(tt.input)
			if len(got) < tt.minLen || len(got) > tt.maxLen {
				t.Errorf("Tokenize(%q) with maxCount=%d produced %d tokens, want [%d, %d]",
					tt.input, tt.maxCount, len(got), tt.minLen, tt.maxLen)
			}
		})
	}
}

func TestTokenizer_WholeTokenOnly(t *testing.T) {
	// With CountPolicy maxCount < segments, only whole token is returned
	tok := New(
		WithPolicy(NewCountPolicy(1)),
		WithLanguages(unicode.Latin),
	)

	tests := []struct {
		input string
	}{
		{"hello.world"},
		{"a.b.c.d"},
		{"foo-bar-baz"},
	}

	for _, tt := range tests {
		got := tok.Tokenize(tt.input)
		if len(got) != 1 {
			t.Errorf("Tokenize(%q) = %d tokens, want 1 (whole token only)", tt.input, len(got))
		}
		if _, ok := got[tt.input]; !ok {
			t.Errorf("Tokenize(%q) missing whole token", tt.input)
		}
	}
}

func TestRuneClassValues(t *testing.T) {
	// The class constants are exported and may be persisted by callers, so
	// renumbering them is a breaking change.
	values := []struct {
		name  string
		class RuneClass
		want  RuneClass
	}{
		{"ClassLetter", ClassLetter, 1},
		{"ClassDigit", ClassDigit, 2},
		{"ClassPunct", ClassPunct, 3},
		{"ClassOther", ClassOther, 4},
	}

	for _, v := range values {
		if v.class != v.want {
			t.Errorf("%s = %d, want %d", v.name, v.class, v.want)
		}
	}

	// Zero stays reserved, so an unset RuneClass is never a real class.
	var unset RuneClass
	if unset != classUndef {
		t.Errorf("zero value = %d, want the unassigned marker (%d)", unset, classUndef)
	}

	s := newScanner(nil, nil)
	for _, r := range []rune{'a', 'Z', 'п', '你', '1', '٣', '.', '。', '☭', ' ', '\n', 0} {
		if got := s.classifyRune(r); got == classUndef {
			t.Errorf("classifyRune(%q) = unassigned, want a real class", r)
		}
	}
}

func TestTokenizer_WholeTokenOnlyMatchesFullDepth(t *testing.T) {
	// The whole-token-only path runs the same language detection as the sliding
	// window, so a letter-free token reports Base [0,0] ("undetermined") rather
	// than a span covering the token.
	tests := []struct {
		input string
		want  TokenInfo
	}{
		{"hello", TokenInfo{Language: 0, Base: [2]int{0, 5}}},
		{"hello.world", TokenInfo{Language: 0, Base: [2]int{0, 11}}},
		{"123hello", TokenInfo{Language: 0, Base: [2]int{3, 8}}},
		{"123...", TokenInfo{Language: -1, Base: [2]int{0, 0}}},
		{"...", TokenInfo{Language: -1, Base: [2]int{0, 0}}},
		{"a.b.c.d.e", TokenInfo{Language: -1, Base: [2]int{0, 0}}},
	}

	wholeOnly := New(
		WithPolicy(NewFixedPolicy(0)),
		WithLanguages(unicode.Latin, unicode.Cyrillic),
	)
	fullDepth := New(
		WithPolicy(NewFixedPolicy(100)),
		WithLanguages(unicode.Latin, unicode.Cyrillic),
	)

	for _, tt := range tests {
		got := wholeOnly.Tokenize(tt.input)
		if len(got) != 1 {
			t.Errorf("Tokenize(%q) = %d tokens, want 1 (whole token only)", tt.input, len(got))
			continue
		}
		if got[tt.input] != tt.want {
			t.Errorf("whole-token-only Tokenize(%q)[%q] = %+v, want %+v",
				tt.input, tt.input, got[tt.input], tt.want)
		}

		// The sliding window must resolve the same token identically.
		if full := fullDepth.Tokenize(tt.input)[tt.input]; full != tt.want {
			t.Errorf("full-depth Tokenize(%q)[%q] = %+v, want %+v",
				tt.input, tt.input, full, tt.want)
		}
	}
}

func TestCountSegments(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"a", 1},
		{"hello", 1},
		{"123", 1},
		{"a.b", 3},         // a | . | b
		{"a.b,c", 5},       // a | . | b | , | c
		{"hello.world", 3}, // hello | . | world
		{"hello123", 2},    // hello | 123
		{"...hello", 2},    // ... | hello
		{"helloпривет", 2}, // Latin | Cyrillic
		{"你好。再见", 3},       // 你好 | 。 | 再见
	}

	s := newScanner([]*unicode.RangeTable{unicode.Latin, unicode.Cyrillic}, nil)
	for _, tt := range tests {
		if got := s.countSegments(tt.input); got != tt.want {
			t.Errorf("countSegments(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

// distinctSegmentToken builds a token of 2k-1 segments in which every segment
// is a different rune, so no two subtokens collapse into the same map entry.
func distinctSegmentToken(k int) string {
	const (
		letters = "abcdefghijklmnopqrstuvwxyz"
		puncts  = ".,;:!?-_'/\\()[]{}*&#@%+=<>~"
	)

	var b strings.Builder
	for i := 0; i < k; i++ {
		if i > 0 {
			b.WriteByte(puncts[(i-1)%len(puncts)])
		}
		b.WriteByte(letters[i%len(letters)])
	}
	return b.String()
}

func TestTokenizer_CountPolicyBound(t *testing.T) {
	// CountPolicy charges the always-emitted whole token against maxCount, so
	// the result map must never exceed it. Segments are all distinct here so
	// deduplication cannot mask an overrun.
	for k := 1; k <= 8; k++ {
		input := distinctSegmentToken(k)
		for maxCount := 1; maxCount <= 60; maxCount++ {
			tok := New(
				WithPolicy(NewCountPolicy(maxCount)),
				WithLanguages(unicode.Latin),
			)
			if got := len(tok.Tokenize(input)); got > maxCount {
				t.Errorf("Tokenize(%q) with maxCount=%d produced %d tokens",
					input, maxCount, got)
			}
		}
	}
}

func TestTokenizer_WholeTokenAlwaysIncluded(t *testing.T) {
	// Depths large enough to skip the whole-token-only path but too small for
	// the sliding window to span the token. Before the fix the whole token was
	// silently dropped in this case.
	tests := []struct {
		name   string
		policy Policy
		input  string
		want   TokenInfo
	}{
		{
			name:   "base spans the token",
			policy: NewFixedPolicy(1),
			input:  "hello.world",
			want:   TokenInfo{Language: 0, Base: [2]int{0, 11}},
		},
		{
			name:   "base undetermined",
			policy: NewCountPolicy(10),
			input:  "a.b.c.d.e",
			want:   TokenInfo{Language: -1, Base: [2]int{0, 0}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok := New(WithPolicy(tt.policy), WithLanguages(unicode.Latin))

			got, ok := tok.Tokenize(tt.input)[tt.input]
			if !ok {
				t.Fatalf("Tokenize(%q) missing whole token", tt.input)
			}
			if got != tt.want {
				t.Errorf("Tokenize(%q)[%q] = %+v, want %+v",
					tt.input, tt.input, got, tt.want)
			}

			// The backfilled value must match what the sliding window emits
			// when it is wide enough to span the token itself.
			fullDepth := New(WithPolicy(NewFixedPolicy(100)), WithLanguages(unicode.Latin))
			if full := fullDepth.Tokenize(tt.input)[tt.input]; full != got {
				t.Errorf("backfilled %+v, but full depth gives %+v", got, full)
			}
		})
	}
}

// negativePolicy always reports a negative depth, exercising the guard in
// extractSubtokens that keeps the ring-buffer capacity from dropping below 1.
type negativePolicy struct{}

func (negativePolicy) Depth(int) int { return -3 }

func TestTokenizer_NonPositiveDepth(t *testing.T) {
	policies := map[string]Policy{
		"fixed zero":      NewFixedPolicy(0),
		"fixed negative":  NewFixedPolicy(-5),
		"always negative": negativePolicy{},
	}

	for name, p := range policies {
		t.Run(name, func(t *testing.T) {
			tok := New(WithPolicy(p), WithLanguages(unicode.Latin))

			var got map[string]TokenInfo
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("Tokenize panicked: %v", r)
					}
				}()
				got = tok.Tokenize("hello.world")
			}()

			if len(got) != 1 {
				t.Errorf("got %d tokens, want 1 (whole token only): %v", len(got), got)
			}
			if _, ok := got["hello.world"]; !ok {
				t.Errorf("missing whole token; got %v", got)
			}
		})
	}
}

func TestNew_NilPolicy(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("nil policy panicked: %v", r)
		}
	}()

	tok := New(WithPolicy(nil), WithLanguages(unicode.Latin))
	got := tok.Tokenize("hello")
	if _, ok := got["hello"]; !ok {
		t.Errorf("Tokenize(%q) missing token; got %v", "hello", got)
	}
}

func TestTokenizer_NilLanguagesIgnored(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("nil language table panicked: %v", r)
		}
	}()

	tok := New(
		WithLanguage(nil),
		WithLanguages(nil, unicode.Latin, nil, unicode.Cyrillic),
	)

	got := tok.Tokenize("aб")
	if got["a"].Language != 0 {
		t.Errorf("Latin language index = %d, want 0", got["a"].Language)
	}
	if got["б"].Language != 1 {
		t.Errorf("Cyrillic language index = %d, want 1", got["б"].Language)
	}
}

func TestTokenizer_MaxDepthDoesNotOverflow(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("maximum policy depth panicked: %v", r)
		}
	}()

	tok := New(
		WithPolicy(NewFixedPolicy(math.MaxInt)),
		WithLanguages(unicode.Latin),
	)
	want := New(
		WithPolicy(NewFixedPolicy(2)),
		WithLanguages(unicode.Latin),
	).Tokenize("a.1")

	if got := tok.Tokenize("a.1"); !reflect.DeepEqual(got, want) {
		t.Errorf("Tokenize with maximum depth = %v, want %v", got, want)
	}
}

func TestTokenizer_Concurrent(t *testing.T) {
	tok := newTestTokenizer()

	inputs := []string{
		"helloпривет", "hello你好", "你好привет", "hello123",
		"aаaа", "你好。再见。", "hello world", "123hello", "...hello",
	}

	// Baseline computed sequentially.
	want := make([]map[string]TokenInfo, len(inputs))
	for i, in := range inputs {
		want[i] = tok.Tokenize(in)
	}

	const goroutines = 32
	var wg sync.WaitGroup
	errs := make(chan error, goroutines)

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i, in := range inputs {
				got := tok.Tokenize(in)
				if !reflect.DeepEqual(got, want[i]) {
					errs <- fmt.Errorf("Tokenize(%q) = %v, want %v", in, got, want[i])
					return
				}
			}
		}()
	}

	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}
