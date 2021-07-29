package approval

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/zlyuancn/zsignal"
	"github.com/zlyuancn/zstr"

	"github.com/zlyuancn/drone-build-notify/config"
	"github.com/zlyuancn/drone-build-notify/logger"
	"github.com/zlyuancn/drone-build-notify/model"
)

const (
	// 构建审批通过api地址
	BuildApprovalPassApiUrl = `{@endpoint}/api/repos/{@repos}/builds/{@build_id}/approve/1`
	// 构建审批不通过api地址
	BuildApprovalNoPassApiUrl = `{@endpoint}/api/repos/{@repos}/builds/{@build_id}`
)

var defaultApproval *approvalStorage

type approvalStorage struct {
	baseCtx       context.Context
	baseCtxCancel context.CancelFunc
	approvals     map[uint32]model.IApproval
	mx            sync.Mutex
}

type ApprovalData struct {
	c          chan struct{}
	ID         uint32
	VerifyCode string
	repos      string
	buildID    string
}

func init() {
	ctx, cancel := context.WithCancel(context.Background())
	defaultApproval = &approvalStorage{
		baseCtx:       ctx,
		baseCtxCancel: cancel,
		approvals:     make(map[uint32]model.IApproval, 8),
	}
	zsignal.RegisterOnShutdown(defaultApproval.Stop)
}

// 注册审批
func (a *approvalStorage) RegistryApproval(repos string, buildID string) model.IApproval {
	approval := model.NewApproval(repos, buildID)

	a.mx.Lock()
	a.approvals[approval.ID()] = approval
	a.mx.Unlock()

	go func(approval model.IApproval) {
		t := time.NewTimer(time.Second * time.Duration(config.Config.ApprovalTimeout))
		select {
		case <-a.baseCtx.Done():
			t.Stop()
			return
		case <-t.C:
			err := Approval(approval.Repos(), approval.BuildID(), false)
			if err != nil {
				logger.Log.Errorf("超时审批发送删除任务失败: %v", err)
			}
		case <-approval.Done():
			t.Stop()
		}
		a.mx.Lock()
		delete(a.approvals, approval.ID())
		a.mx.Unlock()
	}(approval)
	return approval
}

// 审批
func (a *approvalStorage) Approval(id uint32, verifyCode string, allow bool) error {
	if id == 0 || verifyCode == "" {
		return errors.New("approval_id or verify_code is empty")
	}

	a.mx.Lock()
	approval, ok := a.approvals[id]
	a.mx.Unlock()

	if !ok {
		return fmt.Errorf("approval_id(%d) not found", id)
	}

	if verifyCode != approval.VerifyCode() {
		return fmt.Errorf("verify_code(%s) for approval_id(%d) is error", verifyCode, id)
	}

	approval.Done() <- struct{}{}
	return Approval(approval.Repos(), approval.BuildID(), allow)
}

func (a *approvalStorage) Stop() {
	a.baseCtxCancel()
}

// 初始化
func Init() {
	http.HandleFunc("/approval", func(w http.ResponseWriter, req *http.Request) {
		if config.Config.UseApprovalBranch == "" {
			_, _ = w.Write([]byte("UseApprovalBranch is empty"))
			return
		}
		query := req.URL.Query()
		id, _ := strconv.Atoi(query.Get("approval_id"))
		verifyCode := query.Get("verify_code")
		allow := query.Get("allow") == "true"

		err := defaultApproval.Approval(uint32(id), verifyCode, allow)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

// 新的审批
func NewApproval(repos string, buildID string) model.IApproval {
	return defaultApproval.RegistryApproval(repos, buildID)
}

// 审批
func Approval(repos string, buildID string, allow bool) error {
	if repos == "" || buildID == "" {
		return errors.New("repos or build_id is nil")
	}

	args := map[string]interface{}{
		"endpoint": config.Config.DroneServer,
		"repos":    repos,
		"build_id": buildID,
	}

	var request *http.Request
	var err error
	if allow {
		url := zstr.Render(BuildApprovalPassApiUrl, args)
		request, err = http.NewRequest("POST", url, nil)
	} else {
		url := zstr.Render(BuildApprovalNoPassApiUrl, args)
		request, err = http.NewRequest("DELETE", url, nil)
	}

	if err != nil {
		return fmt.Errorf("make request err: %v", err)
	}
	request.Header.Add("Authorization", "Bearer "+config.Config.DroneUserToken)
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("send build approval failure: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("got err http status code: %v", resp.StatusCode)
	}

	return nil
}
