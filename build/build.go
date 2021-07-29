package build

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/drone/drone-go/plugin/webhook"
	"github.com/zlyuancn/zsignal"

	"github.com/zlyuancn/drone-build-notify/approval"
	"github.com/zlyuancn/drone-build-notify/config"
	"github.com/zlyuancn/drone-build-notify/logger"
	"github.com/zlyuancn/drone-build-notify/model"
)

var defaultBuildStorage *buildStorage

type buildStorage struct {
	baseCtx       context.Context
	baseCtxCancel context.CancelFunc
	builds        map[string]*model.Build // key=repos/droneBuildID
	buildsIndex   map[uint32]*model.Build
	mx            sync.Mutex
}

func init() {
	ctx, cancel := context.WithCancel(context.Background())
	defaultBuildStorage = &buildStorage{
		baseCtx:       ctx,
		baseCtxCancel: cancel,
		builds:        make(map[string]*model.Build, 8),
		buildsIndex:   make(map[uint32]*model.Build, 8),
	}
	zsignal.RegisterOnShutdown(defaultBuildStorage.Stop)
}

// 构建Build
func (a *buildStorage) MakeBuild(req *webhook.Request) (*model.Build, error) {
	msg, err := model.MakeMsg(req)
	if err != nil {
		return nil, err
	}

	key := msg.RepoName + "/" + msg.TaskNum
	a.mx.Lock()
	build, ok := a.builds[key]
	a.mx.Unlock()

	if ok {
		build.Reset(req, msg)
		build.CheckStatus()
		return build, nil
	}

	return a.newBuild(key, req, msg), nil
}

func (a *buildStorage) newBuild(key string, req *webhook.Request, msg *model.Msg) *model.Build {
	build := model.NewBuild(req, msg)
	build.CheckStatus()

	a.mx.Lock()
	a.builds[key] = build
	a.buildsIndex[build.ID] = build
	a.mx.Unlock()

	// 如果等待审批设置审批超时
	if build.BuildStatus == model.BuildWaitApproval {
		go func(build *model.Build) {
			t := time.NewTimer(time.Second * time.Duration(config.Config.ApprovalTimeout))
			select {
			case <-a.baseCtx.Done():
				build.BuildStatus = model.BuildEnd
				t.Stop()
			case <-t.C:
				build.BuildStatus = model.BuildApprovalTimeout
				build.CheckStatus()
				err := approval.Approval(build.Msg.RepoName, build.Msg.TaskNum, false)
				if err != nil {
					logger.Log.Errorf("超时审批发送删除任务失败: %v", err)
				}
			case <-build.Approval.Done():
				build.BuildStatus = model.BuildStart
				t.Stop()
			}
		}(build)
	}

	// 清除
	go func(key string, build *model.Build) {
		// 这里默认总会收到结束信息, 所以不做超时自动清除
		select {
		case <-a.baseCtx.Done():
			return
		case <-build.Done():
		}
		a.mx.Lock()
		delete(a.builds, key)
		delete(a.buildsIndex, build.ID)
		a.mx.Unlock()
	}(key, build)
	return build
}

// 检查验证码并审批
func (a *buildStorage) ApprovalAndCheck(id uint32, verifyCode string, allow bool) error {
	if id == 0 || verifyCode == "" {
		return errors.New("build_id or verify_code is empty")
	}

	a.mx.Lock()
	build, ok := a.buildsIndex[id]
	a.mx.Unlock()

	if !ok {
		return fmt.Errorf("build_id(%d) not found", id)
	}

	if verifyCode != build.Approval.VerifyCode {
		return fmt.Errorf("verify_code(%s) for build_id(%d) is error", verifyCode, id)
	}

	if build.BuildStatus != model.BuildWaitApproval {
		return fmt.Errorf("build_id(%d) do not require approval", id)
	}

	err := approval.Approval(build.Msg.RepoName, build.Msg.TaskNum, allow)
	if err != nil {
		return err
	}
	build.Approval.Done() <- struct{}{}
	return nil
}

func (a *buildStorage) Stop() {
	a.baseCtxCancel()
}

// 注册路由
func RegistryRouter() {
	http.HandleFunc("/approval", func(w http.ResponseWriter, req *http.Request) {
		if config.Config.UseApprovalBranch == "" {
			_, _ = w.Write([]byte("UseApprovalBranch is empty"))
			return
		}
		query := req.URL.Query()
		id, _ := strconv.Atoi(query.Get("build_id"))
		verifyCode := query.Get("verify_code")
		allow := query.Get("allow") == "true"

		err := defaultBuildStorage.ApprovalAndCheck(uint32(id), verifyCode, allow)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

// 构建build
func MakeBuild(req *webhook.Request) (*model.Build, error) {
	return defaultBuildStorage.MakeBuild(req)
}
