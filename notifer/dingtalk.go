/*
-------------------------------------------------
   Author :       Zhang Fan
   date：         2020/5/17
   Description :
-------------------------------------------------
*/

package notifer

import (
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/zlyuancn/zdingtalk/robot"
	"github.com/zlyuancn/zstr"

	"github.com/zlyuancn/drone-build-notify/approval"
	"github.com/zlyuancn/drone-build-notify/config"
	"github.com/zlyuancn/drone-build-notify/logger"
	"github.com/zlyuancn/drone-build-notify/model"
	"github.com/zlyuancn/drone-build-notify/template"
)

var _ INotifier = (*DingtalkNotifer)(nil)

var dingtalkStartTemplate string
var dingtalkEndTemplate string

type DingtalkNotifer struct {
	dt *robot.DingTalk
}

func NewDingtalkNotifier() INotifier {
	m := &DingtalkNotifer{}

	if dingtalkStartTemplate == "" {
		dingtalkStartTemplate = m.loadTemplate(config.Config.DingtalkStartTemplate)
	}
	if dingtalkEndTemplate == "" {
		dingtalkEndTemplate = m.loadTemplate(config.Config.DingtalkEndTemplateFile)
	}

	if config.Config.DingtalkAccessToken == "" {
		logger.Log.Warn("未设置DingtalkAccessToken")
		return m
	}

	m.dt = robot.NewDingTalk(config.Config.DingtalkAccessToken).SetSecret(config.Config.DingtalkSecret)
	return m
}

func (m *DingtalkNotifer) loadTemplate(file string) string {
	body, err := ioutil.ReadFile(file)
	if err != nil {
		logger.Log.Fatalf("无法加载模板文件: %s: %s", file, err.Error())
	}
	return string(body)
}

func (m *DingtalkNotifer) Name() NotifierType {
	return DingtalkNotifier
}

func (m *DingtalkNotifer) Notify(msg *model.Msg) error {
	if m.dt == nil {
		return errors.New("未创建DingTalk实例")
	}
	return m.dt.Send(m.makeDingtalkMsg(msg), config.Config.NotifyRetry)
}

func (m *DingtalkNotifer) makeDingtalkMsg(msg *model.Msg) *robot.Msg {
	title := fmt.Sprintf("[%s] #%d %s", msg.StatusDesc, msg.TaskNum, msg.RepoName)
	text := m.makeContext(msg)
	buttons := []robot.Button{
		{
			Title:     "更改记录",
			ActionURL: msg.CommitUrl,
		},
		{
			Title:     "任务构建信息",
			ActionURL: msg.TaskUrl,
		},
	}

	if msg.Status == "start" && msg.MatchApprovalBranches() {
		approvalData := approval.NewApproval(msg.RepoName, msg.TaskNum)
		const ApprovalUrl = `{@endpoint}/approval?approval_id={@approval_id}&verify_code={@verify_code}&allow={@allow}`
		buttons = append(buttons, robot.Button{
			Title: "允许构建",
			ActionURL: zstr.Render(ApprovalUrl, map[string]interface{}{
				"endpoint":    config.Config.AdvertiseAddress,
				"approval_id": approvalData.ID,
				"verify_code": approvalData.VerifyCode,
				"allow":       "true",
			}),
		}, robot.Button{
			Title: "取消构建",
			ActionURL: zstr.Render(ApprovalUrl, map[string]interface{}{
				"endpoint":    config.Config.AdvertiseAddress,
				"approval_id": approvalData.ID,
				"verify_code": approvalData.VerifyCode,
				"allow":       "false",
			}),
		})
	}
	return robot.NewCustomCard(title, text, buttons...).HorizontalButton()
}

func (m *DingtalkNotifer) makeContext(msg *model.Msg) string {
	if msg.Status == "start" {
		return template.Render(dingtalkStartTemplate, msg)
	}
	return template.Render(dingtalkEndTemplate, msg)
}
