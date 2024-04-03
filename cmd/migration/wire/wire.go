//go:build wireinject
// +build wireinject

package wire

import (
	"github.com/google/wire"
	"github.com/mateusmacedo/go-nunu-template/internal/repository"
	"github.com/mateusmacedo/go-nunu-template/internal/server"
	"github.com/mateusmacedo/go-nunu-template/pkg/app"
	"github.com/mateusmacedo/go-nunu-template/pkg/log"
	"github.com/spf13/viper"
)

var repositorySet = wire.NewSet(
	repository.NewDB,
	//repository.NewRedis,
	repository.NewRepository,
	repository.NewUserRepository,
)
var serverSet = wire.NewSet(
	server.NewMigrate,
)

// build App
func newApp(migrate *server.Migrate) *app.App {
	return app.NewApp(
		app.WithServer(migrate),
		app.WithName("demo-migrate"),
	)
}

func NewWire(*viper.Viper, *log.Logger) (*app.App, func(), error) {
	panic(wire.Build(
		repositorySet,
		serverSet,
		newApp,
	))
}
