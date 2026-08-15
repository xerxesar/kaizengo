package module

import "context"

func Shutdown(ctx context.Context, host *Host) error {
	if host == nil {
		return nil
	}
	for i := len(host.shutdown) - 1; i >= 0; i-- {
		if err := host.shutdown[i](ctx); err != nil {
			return err
		}
	}
	return nil
}
