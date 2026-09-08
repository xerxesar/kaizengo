package identity

//go:generate go run ../../cmd/godino gen-types identity

import (
	"kaizengo/internal/engine"
	"kaizengo/internal/module"
)

func init() {
	module.Register(engine.New(engine.Options{
		AppName: "identity",
		Version: "0.1.0",
		Setup:   seed,
	}).Hooks("user", userHooks()))
}
