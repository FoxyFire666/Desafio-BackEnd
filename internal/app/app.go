package app

import (
	"context"
	"net/http"

	"go.uber.org/fx"

	"desafio-backend/internal/config"
	httpserver "desafio-backend/internal/http"
	"desafio-backend/internal/repository"
	"desafio-backend/internal/service"
)

func New() *fx.App {
	cfg := config.Load()
	return fx.New(
		fx.Provide(
			func() config.Config { return cfg },
			func(cfg config.Config) (*repository.Store, error) { return repository.NewStore(cfg.PostgresDSN) },
			func(store *repository.Store) *service.Service { return service.NewService(store) },
		),
		fx.Invoke(func(lc fx.Lifecycle, cfg config.Config, svc *service.Service) {
			server := httpserver.NewServer(cfg, svc)
			lc.Append(fx.Hook{
				OnStart: func(context.Context) error {
					go func() { _ = http.ListenAndServe(cfg.HTTPAddr, server) }()
					return nil
				},
				OnStop: func(context.Context) error { return nil },
			})
		}),
	)
}
