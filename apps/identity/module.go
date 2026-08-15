package identity

//go:generate go run ../../cmd/godino gen-types identity

import (
	"kaizengo/internal/module"
	"kaizengo/packages/sdk-go/engine"
)

func init() {
	module.Register(engine.New(engine.Options{
		AppName: "identity",
		Version: "0.1.0",
		Setup:   seed,
	}).Hooks("user", userHooks()))
}
