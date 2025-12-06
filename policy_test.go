package gotoken

import "testing"

func TestFixedPolicy(t *testing.T) {
	p := NewFixedPolicy(5)

	tests := []int{1, 10, 50, 100, 1000}
	for _, segments := range tests {
		if got := p.Depth(segments); got != 5 {
			t.Errorf("Depth(%d) = %d, want 5", segments, got)
		}
	}
}

func TestLinearPolicy(t *testing.T) {
	const (
		shortLen   = 20
		shortDepth = 100
		longLen    = 120
		longDepth  = 10
	)

	p := NewLinearPolicy(shortLen, shortDepth, longLen, longDepth)

	// Test left boundary (short tokens)
	for i := 1; i <= shortLen; i++ {
		if got := p.Depth(i); got != shortDepth {
			t.Errorf("Depth(%d) = %d, want %d (short tokens)", i, got, shortDepth)
		}
	}

	// Test right boundary (long tokens)
	for i := longLen; i <= longLen+20; i++ {
		if got := p.Depth(i); got != longDepth {
			t.Errorf("Depth(%d) = %d, want %d (long tokens)", i, got, longDepth)
		}
	}

	// Test monotonic decrease in middle
	prev := shortDepth
	for i := shortLen + 1; i < longLen; i++ {
		got := p.Depth(i)
		if got > prev {
			t.Errorf("Depth(%d) = %d > %d, should be monotonically decreasing", i, got, prev)
		}
		prev = got
	}
}

func TestCountPolicy(t *testing.T) {
	tests := []struct {
		name      string
		maxCount  int
		segments  int
		wantDepth int
	}{
		// Edge cases
		{"zero segments", 10, 0, 0},
		{"one segment", 10, 1, 1},

		// When maxCount < segments, depth=0 (whole token only)
		{"max=1, 5 seg -> whole only", 1, 5, 0},
		{"max=4, 5 seg -> whole only", 4, 5, 0},

		// N=5 segments: counts at depth d are d=1:5, d=2:9, d=3:12, d=4:14, d=5:15
		{"5 seg, max=5", 5, 5, 1},   // d=1 fits exactly (count=5)
		{"5 seg, max=8", 8, 5, 1},   // d=1 fits (5), d=2 would be 9
		{"5 seg, max=9", 9, 5, 2},   // d=2 fits exactly (count=9)
		{"5 seg, max=11", 11, 5, 2}, // d=2 fits (9), d=3 would be 12
		{"5 seg, max=12", 12, 5, 3}, // d=3 fits exactly (count=12)
		{"5 seg, max=14", 14, 5, 4}, // d=4 fits exactly (count=14)
		{"5 seg, max=15", 15, 5, 5}, // all levels fit (count=15)
		{"5 seg, max=100", 100, 5, 5},

		// N=10 segments
		{"10 seg, max=9 -> whole only", 9, 10, 0},
		{"10 seg, max=10", 10, 10, 1},  // d=1 fits exactly
		{"10 seg, max=19", 19, 10, 2},  // d=2 = 10+9 = 19
		{"10 seg, max=55", 55, 10, 10}, // all levels: 10+9+8+...+1 = 55
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewCountPolicy(tt.maxCount)
			got := p.Depth(tt.segments)

			if got != tt.wantDepth {
				t.Errorf("CountPolicy(%d).Depth(%d) = %d, want %d",
					tt.maxCount, tt.segments, got, tt.wantDepth)
			}
		})
	}
}

func TestCountPolicy_Optimality(t *testing.T) {
	// Test that CountPolicy always finds the maximum valid depth
	for maxCount := 1; maxCount <= 100; maxCount++ {
		for segments := 1; segments <= 20; segments++ {
			p := NewCountPolicy(maxCount)
			depth := p.Depth(segments)

			// Depth should be in valid range [0, segments]
			if depth < 0 || depth > segments {
				t.Errorf("maxCount=%d, segments=%d: depth=%d out of range [0,%d]",
					maxCount, segments, depth, segments)
				continue
			}

			// Depth 0 means "whole token only" - valid when maxCount < segments
			if depth == 0 {
				if maxCount >= segments {
					t.Errorf("maxCount=%d, segments=%d: depth=0 but should be >=1",
						maxCount, segments)
				}
				continue
			}

			// Count at this depth should not exceed maxCount
			count := subtokenCount(segments, depth)
			if count > maxCount {
				t.Errorf("maxCount=%d, segments=%d: depth=%d gives count=%d (exceeds max)",
					maxCount, segments, depth, count)
			}

			// Count at depth+1 should exceed maxCount (unless we're at max depth)
			if depth < segments {
				nextCount := subtokenCount(segments, depth+1)
				if nextCount <= maxCount {
					t.Errorf("maxCount=%d, segments=%d: depth=%d not optimal, depth=%d gives count=%d",
						maxCount, segments, depth, depth+1, nextCount)
				}
			}
		}
	}
}

func TestSubtokenCount(t *testing.T) {
	// Verify the count formula: d*n - d*(d-1)/2
	tests := []struct {
		n, d, want int
	}{
		{5, 1, 5},  // 5
		{5, 2, 9},  // 5+4
		{5, 3, 12}, // 5+4+3
		{5, 4, 14}, // 5+4+3+2
		{5, 5, 15}, // 5+4+3+2+1
		{10, 1, 10},
		{10, 2, 19}, // 10+9
		{10, 10, 55}, // 10+9+8+...+1 = 55
	}

	for _, tt := range tests {
		got := subtokenCount(tt.n, tt.d)
		if got != tt.want {
			t.Errorf("subtokenCount(%d, %d) = %d, want %d", tt.n, tt.d, got, tt.want)
		}
	}
}
