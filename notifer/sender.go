/*
-------------------------------------------------
   Author :       Zhang Fan
   date：         2020/5/17
   Description :
-------------------------------------------------
*/

package notifer

import (
	"strings"

	"github.com/zlyuancn/zsignal"

	"github.com/zlyuancn/drone-build-notify/config"
	"github.com/zlyuancn/drone-build-notify/logger"
	"github.com/zlyuancn/drone-build-notify/model"
)

type NotifierType string

const (
	DingtalkNotifier NotifierType = "dingtalk"
)

var notifiers []INotifier
var notifyTask chan *model.Msg

type INotifier interface {
	Name() NotifierType
	Notify(msg *model.Msg) error
}

func Init() {
	notifyTask = make(chan *model.Msg, 100)
	zsignal.RegisterOnShutdown(func() {
		close(notifyTask)
	})

	go func() {
		for msg := range notifyTask {
			notify(msg)
		}
	}()

	if config.Config.Notifer == "" {
		return
	}

	for _, notifier := range strings.Split(config.Config.Notifer, ",") {
		switch NotifierType(notifier) {
		case DingtalkNotifier:
			notifiers = append(notifiers, NewDingtalkNotifier())
		default:
			logger.Log.Warn("未知的通告类型: ", notifier)
		}
		logger.Log.Debug("添加通告者: ", notifier)
	}
}

func notify(msg *model.Msg) {
	if config.Config.OffCreateNotify && msg.Status == "start" {
		logger.Log.Debug("跳过动作的公告: ", msg.Status)
		return
	}

	for _, notifier := range notifiers {
		if err := notifier.Notify(msg); err != nil {
			logger.Log.Warnf("通告者<%s>失败: %s", notifier.Name(), err.Error())
		}
	}
}

func Notify(msg *model.Msg) {
	notifyTask <- msg
}
