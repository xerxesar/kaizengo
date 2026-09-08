package status

//go:generate go run ../../cmd/kaizengo gen-types status

import (
	"kaizengo/internal/module"
	"kaizengo/internal/engine"
)

func init() {
	module.Register(engine.New(engine.Options{
		AppName: "status",
		Version: "0.1.0",
	}))
}
