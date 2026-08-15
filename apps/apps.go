// Package apps blank-imports every bundled app so they register via init().
package apps

import (
	_ "kaizengo/apps/appman"
	_ "kaizengo/apps/audit"
	_ "kaizengo/apps/auth"
	_ "kaizengo/apps/core"
	_ "kaizengo/apps/hellospec"
	_ "kaizengo/apps/identity"
	_ "kaizengo/apps/inventory"
	_ "kaizengo/apps/permissions"
	_ "kaizengo/apps/settings"
	_ "kaizengo/apps/typesense"
)
