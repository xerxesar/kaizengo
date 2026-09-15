package appman

//go:generate go run ../../cmd/kaizengo gen-types appman

import (
	"kaizengo/internal/engine"
	"kaizengo/internal/module"
	"kaizengo/packages/sdk-go/appspec"
)

func init() {
	app := engine.New(engine.Options{
		AppName: "appman",
		Version: "0.2.0",
		Setup: func(host *module.Host, events *engine.EventsSetup) error {
			mgr, err := engine.ManagerFromHost(host)
			if err != nil {
				return err
			}
			spec, err := appspec.LoadApp("appman")
			if err != nil {
				return err
			}
			return registerAppModel(host, spec, mgr)
		},
	})
	registerCommands(app)
	module.Register(app)
}
