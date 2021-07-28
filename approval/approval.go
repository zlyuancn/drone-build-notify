package approval

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/zlyuancn/zstr"

	"github.com/zlyuancn/drone-build-notify/config"
)

const (
	// 构建审批通过api地址
	BuildApprovalPassApiUrl = `{@endpoint}/api/repos/{@repos}/builds/{@build_id}/approve/1`
	// 构建审批不通过api地址
	BuildApprovalNoPassApiUrl = `{@endpoint}/api/repos/{@repos}/builds/{@build_id}`
)

func Init() {
	http.HandleFunc("/approval", func(w http.ResponseWriter, req *http.Request) {
		if !config.Config.UseApproval {
			_, _ = w.Write([]byte("UseApproval is false"))
			return
		}
		w.WriteHeader(http.StatusBadRequest)

		query := req.URL.Query()
		repos := query.Get("repos")
		buildID, _ := strconv.Atoi(query.Get("build_id"))
		allow := query.Get("allow") == "true"
		if repos == "" || buildID == 0 {
			_, _ = w.Write([]byte("repos or build_id is nil"))
			return
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
			_, _ = w.Write([]byte(fmt.Sprintf("make request err: %v", err)))
			return
		}
		request.Header.Add("Authorization", "Bearer "+config.Config.DroneUserToken)
		resp, err := http.DefaultClient.Do(request)
		if err != nil {
			_, _ = w.Write([]byte(fmt.Sprintf("send build approval failure: %v", err)))
			return
		}
		_ = resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			_, _ = w.Write([]byte(fmt.Sprintf("got err http status code: %v", resp.StatusCode)))
			return
		}

		_, _ = w.Write([]byte("ok"))
		w.WriteHeader(http.StatusOK)
	})
}
