package gotoken

type PolicyCount int

func NewPolicyCount(max int) TokenizationPolicy {
	return nil
}

func (p PolicyCount) GetDepth(length int) int {
	panic("PolicyCount is not implemented yet")
}
