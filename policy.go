package gotoken

import "math"

// Policy determines the tokenization depth based on segment count.
// A segment is a contiguous run of runes sharing a character class and, for
// letters, a detected language.
// Depth controls how many levels of sub-tokens are generated.
//
// A Tokenizer calls Depth once per token, so an implementation shared by
// concurrent Tokenize calls must be safe for concurrent use. The policies in
// this package are immutable and therefore safe.
type Policy interface {
	// Depth returns the maximum tokenization depth for a token with given segments.
	Depth(segments int) int
}

// FixedPolicy returns a constant depth regardless of segment count.
type FixedPolicy struct {
	depth int
}

// NewFixedPolicy creates a policy with constant depth.
// A negative depth is clamped to 0 (whole token only).
func NewFixedPolicy(depth int) *FixedPolicy {
	if depth < 0 {
		depth = 0
	}
	return &FixedPolicy{depth: depth}
}

// Depth returns the configured constant depth.
func (p *FixedPolicy) Depth(segments int) int {
	return p.depth
}

// LinearPolicy interpolates depth linearly between two points.
// For tokens with few segments (≤ shortLen), it returns shortDepth.
// For tokens with many segments (≥ longLen), it returns longDepth.
// For tokens in between, it linearly interpolates.
type LinearPolicy struct {
	shortLen   int
	shortDepth int
	longLen    int
	longDepth  int
}

// NewLinearPolicy creates a policy that linearly interpolates depth.
//
// Parameters:
//   - shortLen: tokens with this many segments or fewer get shortDepth
//   - shortDepth: depth for short tokens (typically higher)
//   - longLen: tokens with this many segments or more get longDepth
//   - longDepth: depth for long tokens (typically lower)
func NewLinearPolicy(shortLen, shortDepth, longLen, longDepth int) *LinearPolicy {
	return &LinearPolicy{
		shortLen:   shortLen,
		shortDepth: shortDepth,
		longLen:    longLen,
		longDepth:  longDepth,
	}
}

// Depth returns the interpolated depth for the given segment count.
func (p *LinearPolicy) Depth(segments int) int {
	if segments <= p.shortLen {
		return p.shortDepth
	}
	if segments >= p.longLen {
		return p.longDepth
	}

	// Linear interpolation
	ratio := float64(p.longLen-segments) / float64(p.longLen-p.shortLen)
	return int(float64(p.longDepth) + float64(p.shortDepth-p.longDepth)*ratio)
}

// CountPolicy limits the total number of subtokens generated per input token.
// It calculates the optimal depth to stay within the limit while:
//  1. Always including the whole token
//  2. Adding complete levels from smallest to largest subtokens
//  3. Never partially filling a level
//
// The whole token is charged against maxCount, so a token never expands into
// more than maxCount map entries.
type CountPolicy struct {
	maxCount int
}

// NewCountPolicy creates a policy that limits total subtoken count.
//
// The algorithm prioritizes:
//  1. Whole token (always included)
//  2. Single-segment subtokens (depth 1)
//  3. Two-segment subtokens (depth 2)
//  4. And so on, until adding another level would exceed maxCount
func NewCountPolicy(maxCount int) *CountPolicy {
	if maxCount < 1 {
		maxCount = 1
	}
	return &CountPolicy{maxCount: maxCount}
}

// Depth calculates the maximum depth that keeps total subtokens ≤ maxCount.
//
// For N segments at depth D, the sliding window emits D*N - D*(D-1)/2
// subtokens. Tokenize always emits the whole token on top of those whenever the
// window is too narrow to span it, so one slot is reserved for it and the
// largest D satisfying D*N - D*(D-1)/2 ≤ maxCount-1 is returned.
//
// Returns 0 if the remaining budget is too small for even single-segment
// subtokens, meaning only the whole token should be emitted.
func (p *CountPolicy) Depth(segments int) int {
	if segments <= 0 {
		return 0
	}
	if segments == 1 {
		return 1
	}

	n := segments

	// Reserve one slot for the whole token, which is emitted regardless of the
	// depth returned here.
	m := p.maxCount - 1

	// At depth 1, we get N subtokens. If they do not fit in the remaining
	// budget, return 0 (whole token only).
	if m < n {
		return 0
	}

	// Total count at depth d = d*n - d*(d-1)/2
	// Solve: d*n - d*(d-1)/2 <= m
	// Rearranged: d² - (2n+1)*d + 2m >= 0
	// Using quadratic formula: d = [(2n+1) - sqrt((2n+1)² - 8m)] / 2

	discriminant := float64((2*n+1)*(2*n+1) - 8*m)
	if discriminant < 0 {
		// All levels fit within the budget
		return n
	}

	d := (float64(2*n+1) - math.Sqrt(discriminant)) / 2
	depth := int(d)

	// Clamp to valid range [1, n]
	if depth < 1 {
		return 1
	}
	if depth > n {
		return n
	}

	return depth
}
