package gotoken

import (
	"reflect"
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
