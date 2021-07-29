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
var notifyTask chan *model.Build

type INotifier interface {
	Name() NotifierType
	Notify(msg *model.Build) error
}

func Init() {
	notifyTask = make(chan *model.Build, 100)
	zsignal.RegisterOnShutdown(func() {
		close(notifyTask)
	})

	go func() {
		for b := range notifyTask {
			notify(b)
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

func notify(b *model.Build) {
	if config.Config.OffCreateNotify && b.BuildStatus == model.BuildStart {
		logger.Log.Debug("跳过创建build通知")
		return
	}

	for _, notifier := range notifiers {
		if err := notifier.Notify(b); err != nil {
			logger.Log.Warnf("通告者<%s>失败: %s", notifier.Name(), err.Error())
		}
	}
}

func Notify(b *model.Build) {
	notifyTask <- b
}
