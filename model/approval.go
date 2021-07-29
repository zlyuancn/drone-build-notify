package model

import (
	"sync/atomic"

	"github.com/zlyuancn/zutils"
)

var approvalID uint32 // 审批id
type IApproval interface {
	// 审批id
	ID() uint32
	// 校验码
	VerifyCode() string
	// 库
	Repos() string
	// 构建号
	BuildID() string
	// 完成chan
	Done() chan struct{}
}

type approval struct {
	id         uint32
	verifyCode string
	repos      string
	buildID    string
	done       chan struct{}
}

func NewApproval(repos string, buildID string) IApproval {
	return &approval{
		id: atomic.AddUint32(&approvalID, 1),
		verifyCode: zutils.Rand.RandTextOfConfig(&zutils.TextConfig{
			Num:   true,
			Lower: true,
			Upper: true,
		}, 16),
		repos:   repos,
		buildID: buildID,
		done:    make(chan struct{}, 1),
	}
}

func (a *approval) ID() uint32 {
	return a.id
}

func (a *approval) VerifyCode() string {
	return a.verifyCode
}

func (a *approval) Repos() string {
	return a.repos
}

func (a *approval) BuildID() string {
	return a.buildID
}

func (a *approval) Done() chan struct{} {
	return a.done
}
