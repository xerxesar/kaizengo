package hellospec

//go:generate go run ../../cmd/kaizengo gen-types hellospec

import (
	"context"
	"fmt"

	"kaizengo/internal/engine"
	"kaizengo/internal/module"
	"kaizengo/packages/sdk-go/appspec"

	"github.com/jackc/pgx/v5/pgxpool"
)

func init() {
	app := engine.New(engine.Options{
		AppName: "hellospec",
		Version: "0.5.1",
		Setup:   registerDomainMigrations,
	})
	module.Register(app.Hooks("greeting", greetingHooks()))
}

// registerDomainMigrations wires named Go data migrations for tip applies:
//
//	domain add-field hellospec.greeting ok string --no-migrate
//	domain … --migrate-go okok_to_ok
func registerDomainMigrations(host *module.Host, _ *engine.EventsSetup) error {
	doms, err := engine.DomainsFromHost(host)
	if err != nil {
		return nil // platform without domain manager
	}
	doms.RegisterDomainMigration("hellospec", "okok_to_ok", func(ctx context.Context, pool *pgxpool.Pool, schema string, prev, next appspec.AppSpec) error {
		_, err := pool.Exec(ctx, fmt.Sprintf(`
			UPDATE %s.greetings
			SET ok = CASE WHEN okok = '' OR okok IS NULL THEN '$' ELSE okok || '$' END
			WHERE true
		`, quoteIdent(schema)))
		return err
	})
	return nil
}

func quoteIdent(s string) string {
	return `"` + s + `"`
}
