//go:build !linux && !(windows && cgo)

package libnp

import "context"

func getInfo(ctx context.Context) (*Info, error) {
	return nil, nil
}
