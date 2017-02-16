package gotoken

type SmartTokenCountPolicy int

func NewSmartTokenCountPolicy(max int) SmartTokenPolicy {
	return nil
}

func (p PolicyCount) GetDepth(length int) int {
	panic("PolicyCount is not implemented yet")
}
