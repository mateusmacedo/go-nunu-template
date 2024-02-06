//go:build wireinject
// +build wireinject

package wire

import (
	"github.com/mateusmacedo/go-nunu-template/internal/server"
	"github.com/mateusmacedo/go-nunu-template/pkg/app"
	"github.com/mateusmacedo/go-nunu-template/pkg/jwt"
	"github.com/mateusmacedo/go-nunu-template/pkg/log"
	"github.com/mateusmacedo/go-nunu-template/pkg/server/http"

	"github.com/google/wire"
	"github.com/spf13/viper"
)

// var repositorySet = wire.NewSet(
// 	repository.NewDB,
// 	repository.NewRedis,
// 	repository.NewRepository,
// 	repository.NewTransaction,
// )

// var serviceSet = wire.NewSet(
// 	service.NewService,
// )

// var handlerSet = wire.NewSet(
// 	handler.NewHandler,
// )

var serverSet = wire.NewSet(
	server.NewHTTPServer,
	server.NewJob,
	server.NewTask,
)

// build App
func newApp(httpServer *http.Server, job *server.Job) *app.App {
	return app.NewApp(
		app.WithServer(httpServer, job),
		app.WithName("demo-server"),
	)
}

func NewWire(*viper.Viper, *log.Logger) (*app.App, func(), error) {

	panic(wire.Build(
		// repositorySet,
		// serviceSet,
		// handlerSet,
		serverSet,
		// sid.NewSid,
		jwt.NewJwt,
		newApp,
	))
}
