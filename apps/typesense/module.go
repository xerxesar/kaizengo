package typesense

//go:generate go run ../../cmd/godino gen-types typesense

import (
	"os"
	"strings"

	"kaizengo/internal/module"
	tsbackend "kaizengo/internal/platform/search/typesense"
	"kaizengo/packages/sdk-go/app"
	"kaizengo/packages/sdk-go/engine"
	"kaizengo/packages/sdk-go/extension"
)

const appName = "typesense"
const appVersion = "0.1.0"

func init() {
	extension.RegisterNamed("indexDocument", indexDocument)
	extension.RegisterNamed("deleteDocument", deleteDocument)
	extension.RegisterNamed("queryDocuments", queryDocuments)
	module.Register(&App{})
}

type App struct{}

func (a *App) Manifest() module.Manifest {
	return app.ManifestFromSpec(app.MustAppSpec(appName), appVersion)
}

func (a *App) Setup(host *module.Host) error {
	tsbackend.RegisterFromEnv()
	spec := app.MustAppSpec(appName)
	if spec.EnableI18n {
		app.MustLoadLocales(appName)
	}
	if _, err := engine.SetupEvents(host, appName, spec, nil); err != nil {
		return err
	}
	if err := extension.SetupAddon(spec); err != nil {
		return err
	}
	engine.RegisterPing(host, spec)
	RegisterGQL(host)
	return nil
}

func (a *App) Mount(host *module.Host) error {
	return nil
}

func typesenseConnected() bool {
	return strings.TrimSpace(os.Getenv("KaizenGo_TYPESENSE_URL")) != ""
}
