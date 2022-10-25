package main

import (
	"net/http"

	"goweb/internal/cache"
	"goweb/internal/dao"
	"goweb/internal/di"
	"goweb/internal/handler"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

func main() {

	fx.New(
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log}
		}),
		di.ProvideConfig(),
		di.ProvideLogger(),
		dao.ProvideOrderDao(),
		cache.ProvideCache(),
		handler.ProvideRouter(),
		di.ProvideServer(),
		fx.Invoke(func(*http.Server) {}),
	).Run()

}
