package audit

//go:generate go run ../../cmd/godino gen-types audit

import (
	"log/slog"

	"kaizengo/internal/module"
	"kaizengo/internal/app"
	"kaizengo/internal/engine"
	"kaizengo/internal/extension"
)

const appName = "audit"
const appVersion = "0.1.0"

func init() {
	extension.Register("model.*.*.afterCreate", 100, logMutation("create"))
	extension.Register("model.*.*.afterUpdate", 100, logMutation("update"))
	extension.Register("model.*.*.afterDelete", 100, logMutation("delete"))
	module.Register(&App{})
}

type App struct{}

func (a *App) Manifest() module.Manifest {
	return app.ManifestFromSpec(app.MustAppSpec(appName), appVersion)
}

func (a *App) Setup(host *module.Host) error {
	spec := app.MustAppSpec(appName)
	if _, err := engine.SetupEvents(host, appName, spec, nil); err != nil {
		return err
	}
	return nil
}

func (a *App) Mount(host *module.Host) error {
	_ = host
	return nil
}

func logMutation(action string) func(extension.Context) error {
	return func(ctx extension.Context) error {
		slog.Info("audit model mutation",
			"action", action,
			"point", ctx.Point,
			"app", ctx.App.Name,
			"model", ctx.Model.Name,
			"recordId", ctx.RecordID,
			"orgId", ctx.OrgID,
			"userId", ctx.UserID,
		)
		return nil
	}
}
