package status

//go:generate go run ../../cmd/godino gen-types status

import (
	"kaizengo/internal/module"
	"kaizengo/packages/sdk-go/engine"
)

func init() {
	module.Register(engine.New(engine.Options{
		AppName: "status",
		Version: "0.1.0",
	}))
}
