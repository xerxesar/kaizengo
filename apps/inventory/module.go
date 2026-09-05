package inventory

//go:generate go run ../../cmd/godino gen-types inventory

import (
	"kaizengo/internal/module"
	"kaizengo/internal/engine"
)

func init() {
	module.Register(engine.New(engine.Options{
		AppName: "inventory",
		Version: "0.2.0",
		Setup:   setup,
	}))
}
