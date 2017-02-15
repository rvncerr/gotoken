package gotoken

type SmartTokenDepthPolicy struct {
	maxLength int
	maxDepth  int
	minLength int
	minDepth  int
}

func NewSmartTokenDepthPolicy(maxLength int, maxDepth int, minLength int, minDepth int) SmartTokenPolicy {
	return &SmartTokenDepthPolicy{
		maxLength: maxLength,
		maxDepth:  maxDepth,
		minLength: minLength,
		minDepth:  minDepth,
	}
}

func (p *SmartTokenDepthPolicy) GetDepth(length int) int {
	// TODO: It seems min & max conditoins are reversed???
	if length <= p.maxLength {
		return p.maxDepth
	} else if length >= p.minLength {
		return p.minDepth
	}

	// Linear interpolation
	return int(float64(p.minDepth) + float64(p.maxDepth-p.minDepth)*float64(p.minLength-length)/float64(p.minLength-p.maxLength))
}
