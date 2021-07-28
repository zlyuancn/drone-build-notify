package approval

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zlyuancn/zsignal"
	"github.com/zlyuancn/zstr"
	"github.com/zlyuancn/zutils"

	"github.com/zlyuancn/drone-build-notify/config"
	"github.com/zlyuancn/drone-build-notify/logger"
)

const (
	// 构建审批通过api地址
	BuildApprovalPassApiUrl = `{@endpoint}/api/repos/{@repos}/builds/{@build_id}/approve/1`
	// 构建审批不通过api地址
	BuildApprovalNoPassApiUrl = `{@endpoint}/api/repos/{@repos}/builds/{@build_id}`
)

var defaultApproval *approval

type approval struct {
	baseCtx       context.Context
	baseCtxCancel context.CancelFunc
	approvalID    uint32 // 审批id
	approvalData  map[uint32]*ApprovalData
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
	defaultApproval := &approval{
		baseCtx:       ctx,
		baseCtxCancel: cancel,
		approvalData:  make(map[uint32]*ApprovalData, 8),
	}
	zsignal.RegisterOnShutdown(defaultApproval.Stop)
}

// 注册审批
func (a *approval) RegistryApproval(repos string, buildID string) *ApprovalData {
	id := atomic.AddUint32(&a.approvalID, 1)
	verifyCode := zutils.Rand.RandTextOfConfig(&zutils.TextConfig{
		Num:   true,
		Lower: true,
		Upper: true,
	}, 16)

	data := &ApprovalData{
		ID:         id,
		repos:      repos,
		buildID:    buildID,
		VerifyCode: verifyCode,
		c:          make(chan struct{}, 1),
	}

	a.mx.Lock()
	a.approvalData[id] = data
	a.mx.Unlock()

	go func(data *ApprovalData) {
		t := time.NewTimer(time.Second * time.Duration(config.Config.ApprovalTimeout))
		select {
		case <-a.baseCtx.Done():
			t.Stop()
			return
		case <-t.C:
			err := Approval(data.repos, data.buildID, false)
			if err != nil {
				logger.Log.Errorf("超时审批发送删除任务失败: %v", err)
			}
		case <-data.c:
			t.Stop()
		}
		a.mx.Lock()
		delete(a.approvalData, data.ID)
		a.mx.Unlock()
	}(data)
	return data
}

// 审批
func (a *approval) Approval(id uint32, verifyCode string, allow bool) error {
	if id == 0 || verifyCode == "" {
		return errors.New("approval_id or verify_code is empty")
	}

	a.mx.Lock()
	data, ok := a.approvalData[id]
	a.mx.Unlock()

	if !ok {
		return fmt.Errorf("approval_id(%d) not found", id)
	}

	if verifyCode != data.VerifyCode {
		return fmt.Errorf("verify_code(%s) for approval_id(%d) is error", verifyCode, id)
	}

	data.c <- struct{}{}
	return Approval(data.repos, data.buildID, allow)
}

func (a *approval) Stop() {
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
func NewApproval(repos string, buildID string) *ApprovalData {
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
