/*
-------------------------------------------------
   Author :       Zhang Fan
   date：         2020/5/16
   Description :
-------------------------------------------------
*/

package model

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/drone/drone-go/drone"
	"github.com/drone/drone-go/plugin/webhook"

	"github.com/zlyuancn/drone-build-notify/config"
)

const TimeLayout = "2006-01-02 15:04:05"

type MsgStatus string

const (
	MsgStart           MsgStatus = "start"
	MsgSuccess         MsgStatus = "success"
	MsgApprovalTimeout MsgStatus = "approval_timeout"
	MsgFailing         MsgStatus = "failure"
	MsgKilled          MsgStatus = "killed"
	MsgError           MsgStatus = "error"
)

type Msg struct {
	TaskNum  string `json:"task_num"`  // 任务号
	TaskUrl  string `json:"task_url"`  // 任务跳转url
	RepoName string `json:"repo_name"` // 仓库名
	Branch   string `json:"branch"`    // 分支名
	RepoUrl  string `json:"repo_url"`  // 仓库地址, 转到该分支

	Auther       string `json:"auther"`        // 操作人员
	AutherEmail  string `json:"auther_email"`  // 操作人员邮箱
	AutherAvatar string `json:"auther_avatar"` // 操作人员头像

	Status       MsgStatus `json:"status"`         // 执行结果
	StatusDesc   string    `json:"status_desc"`    // 执行结果描述
	StatusPicUrl string    `json:"status_pic_url"` // 执行结果图片url

	StartTime   string `json:"start_time"`   // 开始时间
	EndTime     string `json:"end_time"`     // 结束时间
	ProcessTime string `json:"process_time"` // 处理时间

	CommitMsg string `json:"commit_msg"` // 提交信息
	CommitId  string `json:"commit_id"`  // 提交id
	CommitUrl string `json:"commit_url"` // 提交信息的跳转url

	templateValues map[string]string
}

func (m *Msg) Get(key string) string {
	if len(m.templateValues) == 0 {
		msgType := reflect.TypeOf(m).Elem()
		msgVal := reflect.ValueOf(m).Elem()

		fieldCount := msgType.NumField()
		m.templateValues = make(map[string]string, fieldCount)
		for i := 0; i < fieldCount; i++ {
			field := msgType.Field(i)
			if field.PkgPath != "" {
				continue
			}

			k := field.Tag.Get("json")
			v := msgVal.Field(i).String()
			m.templateValues[k] = v
		}
	}

	if v, ok := m.templateValues[key]; ok {
		return v
	}
	return fmt.Sprintf("{ %s: undefined }", key)
}

func MakeMsg(req *webhook.Request) (*Msg, error) {
	repo := req.Repo
	build := req.Build
	if repo == nil {
		return nil, errors.New("没有 repo 信息")
	}
	if build == nil {
		return nil, errors.New("没有 build 信息")
	}

	repoUrl := repo.HTTPURL
	if build.Source != repo.Branch {
		repoUrl = makeBranchUrl(repo.HTTPURL, build.Source)
	}

	startTime := time.Unix(build.Created, 0).Format(TimeLayout)
	endTime := ""
	processTime := "0s"
	if build.Finished > 0 {
		endTime = time.Unix(build.Finished, 0).Format(TimeLayout)
		processTime = (time.Duration(build.Finished-build.Created) * time.Second).String()
	}

	msgStatus := MsgStart
	statusDesc := "开始"
	if req.Action != webhook.ActionCreated {
		switch req.Build.Status {
		case drone.StatusPassing:
			msgStatus = MsgSuccess
			statusDesc = "完成"
		case drone.StatusFailing:
			msgStatus = MsgFailing
			statusDesc = "失败"
		case drone.StatusKilled:
			msgStatus = MsgKilled
			statusDesc = "删除"
		case drone.StatusError:
			msgStatus = MsgError
			statusDesc = "错误"
		}
	}

	msg := &Msg{
		TaskNum:      strconv.FormatInt(build.Number, 10),
		RepoName:     repo.Slug,
		Branch:       build.Source,
		RepoUrl:      repoUrl,
		Auther:       build.AuthorName,
		AutherEmail:  build.AuthorEmail,
		AutherAvatar: build.AuthorAvatar,
		Status:       msgStatus,
		StatusDesc:   statusDesc,
		StatusPicUrl: makeStatusPicUrl(msgStatus),
		StartTime:    startTime,
		EndTime:      endTime,
		ProcessTime:  processTime,
		CommitMsg:    strings.TrimSpace(build.Message),
		CommitId:     build.After,
		CommitUrl:    makeCommitUrl(repo.HTTPURL, build.After),
		TaskUrl:      makeTaskUrl(repo.Slug, build.Number),
	}
	return msg, nil
}

// 是否匹配审批分支
func (m *Msg) MatchApprovalBranches() bool {
	if config.Config.UseApprovalBranch == "" {
		return false
	}

	branches := strings.Split(config.Config.UseApprovalBranch, ",")
	for _, branch := range branches {
		if m.Branch == branch {
			return true
		}
	}
	return false
}

// 构建储存库基础url(储存库的url地址)
func makeRepoBaseUrl(repoUrl string) string {
	return strings.TrimSuffix(repoUrl, ".git")
}

// 构建分支url
func makeBranchUrl(repoUrl, branch string) string {
	repoBaseUrl := makeRepoBaseUrl(repoUrl)
	if strings.Contains(repoBaseUrl, "//github") {
		return fmt.Sprintf("%s/tree/%s", repoBaseUrl, branch)
	}
	if strings.Contains(repoBaseUrl, "//gitee") {
		return fmt.Sprintf("%s/tree/%s", repoBaseUrl, branch)
	}
	if strings.Contains(repoBaseUrl, "//gitea") {
		return fmt.Sprintf("%s/src/branch/%s", repoBaseUrl, branch)
	}

	return fmt.Sprintf("%s/src/%s", repoBaseUrl, branch)
}

// 构建资源url
func makeResUrl(repoUrl, branch, res string) string {
	repoBaseUrl := makeRepoBaseUrl(repoUrl)
	if strings.Contains(repoBaseUrl, "//github") {
		return fmt.Sprintf("%s/%s/%s", strings.Replace(repoBaseUrl, "github.com", "raw.githubusercontent.com", 1), branch, res)
	}
	if strings.Contains(repoBaseUrl, "//gitee") {
		return fmt.Sprintf("%s/raw/%s/%s", repoBaseUrl, branch, res)
	}
	if strings.Contains(repoBaseUrl, "//gitea") {
		return fmt.Sprintf("%s/raw/branch/%s/%s", repoBaseUrl, branch, res)
	}
	return fmt.Sprintf("%s/raw/%s/%s", repoBaseUrl, branch, res)
}

// 构建commit跳转url
func makeCommitUrl(repoUrl, commitId string) string {
	repoBaseUrl := makeRepoBaseUrl(repoUrl)
	return fmt.Sprintf("%s/commit/%s", repoBaseUrl, commitId)
}

// 构建任务跳转url
func makeTaskUrl(slug string, taskId int64) string {
	return fmt.Sprintf("%s/%s/%d", config.Config.DroneServer, slug, taskId)
}

// 构建状态图片url
func makeStatusPicUrl(status MsgStatus) string {
	return makeResUrl("https://github.com/zlyuancn/drone-build-notify.git", "master", fmt.Sprintf("assets/%s.png", status))
}
