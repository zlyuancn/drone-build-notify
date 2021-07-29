package model

import (
	"sync/atomic"

	"github.com/drone/drone-go/plugin/webhook"

	"github.com/zlyuancn/drone-build-notify/approval"
	"github.com/zlyuancn/drone-build-notify/config"
	"github.com/zlyuancn/drone-build-notify/logger"
)

// 构建状态
type BuildStatus string

const (
	// 等待审批, 需要发送消息告知
	BuildWaitApproval BuildStatus = "wait_approval"
	// 审批超时
	BuildApprovalTimeout BuildStatus = "approval_timeout"
	// 构建开始
	BuildStart BuildStatus = "start"
	// 结束
	BuildEnd BuildStatus = "end"
)

var globalBuildID uint32 // 构建id

type Build struct {
	ID uint32
	BuildStatus
	Approval *Approval
	droneReq *webhook.Request
	Msg      *Msg
	done     chan struct{}
}

func NewBuild(req *webhook.Request, msg *Msg) *Build {
	return &Build{
		ID:       atomic.AddUint32(&globalBuildID, 1),
		Approval: newApproval(),
		droneReq: req,
		Msg:      msg,
		done:     make(chan struct{}, 1),
	}
}

func (b *Build) CheckStatus() {
	if b.BuildStatus == BuildApprovalTimeout {
		b.Msg.Status = MsgApprovalTimeout
		b.Msg.StatusDesc = "审批超时"
		return
	}

	switch b.Msg.Status {
	case MsgStart:
		if config.Config.UseApprovalBranch == "" || !b.droneReq.Repo.Protected { // 不使用审批
			b.BuildStatus = BuildStart
			return
		}

		if b.Msg.MatchApprovalBranches() { // 匹配分支
			b.BuildStatus = BuildWaitApproval
			return
		}

		// 自动审批
		b.BuildStatus = BuildStart
		err := approval.Approval(b.Msg.RepoName, b.Msg.TaskNum, true)
		if err != nil {
			logger.Log.Errorf("自动审批不匹配的分支失败: %v", err)
		}
	case MsgSuccess, MsgFailing, MsgKilled, MsgError:
		b.BuildStatus = BuildEnd
	}
}

func (b *Build) Done() chan struct{} {
	return b.done
}

func (b *Build) Reset(req *webhook.Request, msg *Msg) {
	b.droneReq = req
	b.Msg = msg
}
