package plugin

import (
	"context"

	"github.com/drone/drone-go/drone"
	"github.com/drone/drone-go/plugin/webhook"
	jsoniter "github.com/json-iterator/go"

	"github.com/zlyuancn/drone-build-notify/approval"
	"github.com/zlyuancn/drone-build-notify/config"
	"github.com/zlyuancn/drone-build-notify/logger"
	"github.com/zlyuancn/drone-build-notify/message"
	"github.com/zlyuancn/drone-build-notify/notifer"
)

type plugin struct {
}

func New() webhook.Plugin {
	return &plugin{}
}

func (p *plugin) Deliver(ctx context.Context, req *webhook.Request) error {
	reqText, _ := jsoniter.MarshalIndent(req, "", "    ")
	logger.Log.Debug("收到req:\n", string(reqText))

	if !Check(req) {
		return nil
	}

	msg, err := message.MakeMsg(req)
	if err != nil {
		logger.Log.Error(err)
	}

	// 如果是 开始 并且 对某些分支使用了审批
	if msg.Status == "start" && config.Config.UseApprovalBranch != "" {
		if !msg.MatchApprovalBranches() {
			err := approval.Approval(msg.RepoName, msg.TaskNum, true)
			if err != nil {
				logger.Log.Errorf("自动审批不匹配的分支失败: %v", err)
			}
		}
	}

	msgText, _ := jsoniter.MarshalIndent(msg, "", "    ")
	logger.Log.Debug("通告msg:\n", string(msgText))
	notifer.Notify(msg)
	return nil
}

func Check(req *webhook.Request) bool {
	if req.Event != webhook.EventBuild {
		return false
	}

	switch req.Action {
	case webhook.ActionCreated:
		switch req.Build.Status {
		case drone.StatusPending:
			return true
		}
	case webhook.ActionUpdated:
		switch req.Build.Status {
		case drone.StatusPassing:
			return true
		case drone.StatusFailing:
			return true
		case drone.StatusKilled:
			return true
		case drone.StatusError:
			return true
		}
	}
	return false
}
