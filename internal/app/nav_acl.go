package app

import (
	"kaizengo/packages/sdk-go/appspec"
)

// RegisterShellNavResource is a no-op: shell nav is not an ACL resource.
// App visibility in the Apps dropdown is implied from view access.
func RegisterShellNavResource(appName string, spec appspec.AppSpec) {}
