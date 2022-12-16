//go:build !linux && !(windows && cgo)

package libnp

func getInfo(ctx context.Context) (*Info, error) {
	return nil, nil
}
