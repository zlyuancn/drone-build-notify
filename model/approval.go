package model

import (
	"github.com/zlyuancn/zutils"
)

type Approval struct {
	VerifyCode string
	done       chan struct{}
}

func newApproval() *Approval {
	return &Approval{
		VerifyCode: zutils.Rand.RandTextOfConfig(&zutils.TextConfig{
			Num:   true,
			Lower: true,
			Upper: true,
		}, 16),
		done: make(chan struct{}, 1),
	}
}

func (a *Approval) Done() chan struct{} {
	return a.done
}
