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

// 审批
func Approval(repos string, droneBuildID string, allow bool) error {
	if repos == "" || droneBuildID == "" {
		return errors.New("repos or build_id is nil")
	}

	args := map[string]interface{}{
		"endpoint": config.Config.DroneServer,
		"repos":    repos,
		"build_id": droneBuildID,
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
