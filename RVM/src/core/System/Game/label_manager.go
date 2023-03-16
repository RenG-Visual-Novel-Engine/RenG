package game

import (
	"RenG/RVM/src/core/obj"
)

type LabelManager struct {
	labels map[string]*obj.Label

	nowlabelName  string
	nowlabelIndex int
	// TODO struct {string; int;} 로 변경해야함
	labelCallStack []struct {
		Name  string // label name
		Index int
	}
}

func (lm *LabelManager) GetNowLabelObject() obj.LabelObject {
	return lm.labels[lm.nowlabelName].Obj[lm.nowlabelIndex]
}

func (lm *LabelManager) GetNowLabelName() string {
	return lm.nowlabelName
}

func (lm *LabelManager) GetNowLabelIndex() int {
	return lm.nowlabelIndex
}

func (lm *LabelManager) GetCallStack() []struct {
	Name  string
	Index int
} {
	return lm.labelCallStack
}

func (lm *LabelManager) SetNowLabelName(n string) {
	lm.nowlabelName = n
}

func (lm *LabelManager) SetNowLabelIndex(i int) {
	lm.nowlabelIndex = i
}

func (lm *LabelManager) SetCallStack(s []struct {
	Name  string
	Index int
}) {
	lm.labelCallStack = s
}

func (lm *LabelManager) AddCallStack(n string, i int) {
	lm.labelCallStack = append(lm.labelCallStack, struct {
		Name  string
		Index int
	}{n, i})
}

func (lm *LabelManager) JumpLabel(name string) bool {
	_, ok := lm.labels[name]
	if !ok {
		return ok
	}

	lm.nowlabelName = name
	lm.nowlabelIndex = 0
	lm.labelCallStack = nil
	lm.labelCallStack = append(lm.labelCallStack, struct {
		Name  string
		Index int
	}{name, 0})

	return ok
}

func (lm *LabelManager) CallLabel(name string) bool {
	_, ok := lm.labels[name]
	if !ok {
		return ok
	}

	lm.nowlabelName = name
	lm.nowlabelIndex = 0
	lm.labelCallStack = append(lm.labelCallStack, struct {
		Name  string
		Index int
	}{name, 0})

	return ok
}

func (lm *LabelManager) NextLabelObject() bool {
	// label의 마지막에 도착했을때,
	if len(lm.labels[lm.nowlabelName].Obj)-1 <= lm.nowlabelIndex {
		// Call Stack이 현재 라벨만이 아니라면
		if len(lm.labelCallStack) > 1 {
			lm.nowlabelName = lm.labelCallStack[len(lm.labelCallStack)-2].Name
			lm.nowlabelIndex = lm.labelCallStack[len(lm.labelCallStack)-2].Index + 1
			lm.labelCallStack = lm.labelCallStack[:len(lm.labelCallStack)-1]

			return true
		}

		lm.nowlabelName = ""
		lm.nowlabelIndex = 0

		return false
	}

	lm.labelCallStack[len(lm.labelCallStack)-1].Index++
	lm.nowlabelIndex++

	return true
}
