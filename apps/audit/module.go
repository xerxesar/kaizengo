package audit

//go:generate go run ../../cmd/kaizengo gen-types audit

import (
	"log/slog"

	"kaizengo/internal/engine"
	"kaizengo/internal/extension"
	"kaizengo/internal/module"
)

const appName = "audit"
const appVersion = "0.1.0"

func init() {
	extension.Register("model.*.*.afterCreate", 100, logMutation("create"))
	extension.Register("model.*.*.afterUpdate", 100, logMutation("update"))
	extension.Register("model.*.*.afterDelete", 100, logMutation("delete"))
	module.Register(engine.New(engine.Options{
		AppName: appName,
		Version: appVersion,
	}))
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
