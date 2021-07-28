/*
-------------------------------------------------
   Author :       Zhang Fan
   date：         2020/5/16
   Description :
-------------------------------------------------
*/

package config

import (
	"github.com/zlyuancn/zenvconf"

	"github.com/zlyuancn/drone-build-notify/logger"
)

var Config struct {
	Bind  string `env:"DRONE_BIND"`  // bind端口
	Debug bool   `env:"DRONE_DEBUG"` // debug模式

	DroneServer string `env:"DRONE_SERVER"`         // drone服务地址
	Secret      string `env:"DRONE_WEBHOOK_SECRET"` // webhook秘钥

	LogPath string `env:"LOG_PATH"` // 日志路径

	Notifer         string `env:"NOTIFER"`           // 通告者,多个通告者用半角逗号隔开
	NotifyRetry     int    `env:"NOTIFY_RETRY"`      // 通告失败重试次数
	OffCreateNotify bool   `env:"OFF_CREATE_NOTIFY"` // 关闭创建动作的通告

	UseApprovalBranch string `env:"USE_APPROVAL_BRANCH"` // 使用审批的分支, 多个分支用英文逗号隔开, AdvertiseAddress和DroneUserToken不能为空
	AdvertiseAddress  string `env:"ADVERTISE_ADDRESS"`   // 公告地址, 如: http://notify.drone.example.com
	DroneUserToken    string `env:"DRONE_USER_TOKEN"`    // drone用户token

	DingtalkAccessToken     string `env:"DINGTALK_ACCESSTOKEN"`    // 钉钉access_token
	DingtalkSecret          string `env:"DINGTALK_SECRET"`         // 钉钉secret
	DingtalkStartTemplate   string `env:"DINGTALK_START_TEMPLATE"` // 钉钉消息任务开始模板文件
	DingtalkEndTemplateFile string `env:"DINGTALK_END_TEMPLATE"`   // 钉钉消息任务结束模板文件
}

func Init() {
	Config.NotifyRetry = 2

	err := zenvconf.NewEnvConf().Parse(&Config)
	if err != nil {
		logger.Log.Fatal("初始化失败 ", err)
	}

	if Config.Bind == "" {
		Config.Bind = ":80"
	}
	if Config.Secret == "" {
		logger.Log.Fatal("未设置 Secret")
	}
	if Config.DroneServer == "" {
		logger.Log.Fatal("未设置 DroneServer")
	}
	if Config.DingtalkStartTemplate == "" {
		Config.DingtalkStartTemplate = "./conf/dingtask_start_template.md"
	}
	if Config.DingtalkEndTemplateFile == "" {
		Config.DingtalkEndTemplateFile = "./conf/dingtask_end_template.md"
	}
	if Config.UseApprovalBranch != "" {
		if Config.AdvertiseAddress == "" || Config.DroneUserToken == "" {
			logger.Log.Fatal("如果使用审批, AdvertiseAddress和DroneUserToken不能为空")
		}
	}
}
