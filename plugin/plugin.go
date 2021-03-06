package plugin

import (
	"context"

	"github.com/drone/drone-go/drone"
	"github.com/drone/drone-go/plugin/webhook"
	jsoniter "github.com/json-iterator/go"

	"github.com/zlyuancn/drone-build-notify/build"
	"github.com/zlyuancn/drone-build-notify/logger"
	"github.com/zlyuancn/drone-build-notify/model"
	"github.com/zlyuancn/drone-build-notify/notifer"
)

type plugin struct{}

func New() webhook.Plugin {
	return &plugin{}
}

func (p *plugin) Deliver(ctx context.Context, req *webhook.Request) error {
	reqText, _ := jsoniter.MarshalIndent(req, "", "    ")
	logger.Log.Debug("收到req:\n", string(reqText))

	if !Check(req) {
		return nil
	}

	b, err := build.MakeBuild(req)
	if err != nil {
		logger.Log.Error(err)
		return err
	}

	buildText, _ := jsoniter.MarshalIndent(b, "", "    ")
	logger.Log.Debug("build:\n", string(buildText))

	notifer.Notify(b)

	if b.BuildStatus == model.BuildEnd || b.BuildStatus == model.BuildApprovalTimeout {
		b.Done() <- struct{}{}
	}
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
