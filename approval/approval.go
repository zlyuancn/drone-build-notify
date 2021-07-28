package approval

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/zlyuancn/zstr"

	"github.com/zlyuancn/drone-build-notify/config"
)

const (
	// 构建审批通过api地址
	BuildApprovalPassApiUrl = `{@endpoint}/api/repos/{@repos}/builds/{@build_id}/approve/1`
	// 构建审批不通过api地址
	BuildApprovalNoPassApiUrl = `{@endpoint}/api/repos/{@repos}/builds/{@build_id}`
)

// 初始化
func Init() {
	http.HandleFunc("/approval", func(w http.ResponseWriter, req *http.Request) {
		if config.Config.UseApprovalBranch == "" {
			_, _ = w.Write([]byte("UseApprovalBranch is empty"))
			return
		}
		query := req.URL.Query()
		repos := query.Get("repos")
		buildID := query.Get("build_id")
		allow := query.Get("allow") == "true"

		err := Approval(repos, buildID, allow)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
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
