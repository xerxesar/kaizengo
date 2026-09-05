package hellospec

//go:generate go run ../../cmd/godino gen-types hellospec

import (
	"kaizengo/internal/module"
	"kaizengo/internal/engine"
)

func init() {
	app := engine.New(engine.Options{
		AppName: "hellospec",
		Version: "0.4.0",
	})
	module.Register(app.Hooks("greeting", greetingHooks()))
}
