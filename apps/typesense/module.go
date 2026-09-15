package typesense

//go:generate go run ../../cmd/kaizengo gen-types typesense

import (
	"os"
	"strings"

	"kaizengo/internal/app"
	"kaizengo/internal/engine"
	"kaizengo/internal/extension"
	"kaizengo/internal/module"
	tsbackend "kaizengo/internal/platform/search/typesense"
)

const appName = "typesense"
const appVersion = "0.1.0"

func init() {
	extension.RegisterNamed("indexDocument", indexDocument)
	extension.RegisterNamed("deleteDocument", deleteDocument)
	extension.RegisterNamed("queryDocuments", queryDocuments)
	module.Register(engine.New(engine.Options{
		AppName: appName,
		Version: appVersion,
		Setup: func(host *module.Host, _ *engine.EventsSetup) error {
			tsbackend.RegisterFromEnv()
			spec := app.MustAppSpec(appName)
			if err := extension.SetupAddon(spec); err != nil {
				return err
			}
			RegisterGQL(host)
			return nil
		},
	}))
}

func typesenseConnected() bool {
	return strings.TrimSpace(os.Getenv("KaizenGo_TYPESENSE_URL")) != ""
}
