package appman

//go:generate go run ../../cmd/godino gen-types appman

import (
	"kaizengo/internal/module"
	"kaizengo/internal/engine"
)

func init() {
	module.Register(engine.New(engine.Options{
		AppName: "appman",
		Version: "0.1.0",
		Setup: func(host *module.Host, events *engine.EventsSetup) error {
			_ = events
			mgr, err := engine.ManagerFromHost(host)
			if err != nil {
				return err
			}
			registerGQL(host, mgr)
			return nil
		},
	}))
}
