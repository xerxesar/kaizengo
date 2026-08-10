// Package apps blank-imports every bundled app so they register via init().
package apps

import (
	_ "kaizengo/apps/clock"
	_ "kaizengo/apps/core"
	_ "kaizengo/apps/counter"
	_ "kaizengo/apps/notes"
	_ "kaizengo/apps/oracle"
	_ "kaizengo/apps/settings"
	_ "kaizengo/apps/status"
)
