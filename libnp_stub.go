//go:build !linux && !windows

package libnp

import "context"

func getInfo(ctx context.Context) (*Info, error) {
	return nil, nil
}
