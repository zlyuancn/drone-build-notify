package main

import (
	"net/http"

	"github.com/drone/drone-go/plugin/webhook"
	jsoniter "github.com/json-iterator/go"
	"github.com/zlyuancn/zsignal"

	"github.com/zlyuancn/drone-build-notify/build"
	"github.com/zlyuancn/drone-build-notify/config"
	"github.com/zlyuancn/drone-build-notify/logger"
	"github.com/zlyuancn/drone-build-notify/notifer"
	"github.com/zlyuancn/drone-build-notify/plugin"
)

func main() {
	defer zsignal.Shutdown()

	config.Init()
	logger.Init(config.Config.Debug, config.Config.LogPath)

	// 打印config
	{
		configText, _ := jsoniter.MarshalIndent(&config.Config, "", "    ")
		logger.Log.Debug("config:\n", string(configText))
	}

	notifer.Init()

	// 注册handler
	handler := webhook.Handler(
		plugin.New(),
		config.Config.Secret,
		logger.MakeLogger(),
	)
	http.Handle("/", handler)
	build.RegistryRouter()

	// 启动
	logger.Log.Info("服务启动: ", config.Config.Bind)
	server := &http.Server{Addr: config.Config.Bind}
	zsignal.RegisterOnShutdown(func() {
		_ = server.Close()
	})
	if err := server.ListenAndServe(); err != nil && err == http.ErrServerClosed {
		logger.Log.Fatal(err)
	}
}
