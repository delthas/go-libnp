# go-libnp [![GoDoc](https://godocs.io/github.com/delthas/go-libnp?status.svg)](https://godocs.io/github.com/delthas/go-libnp)

A tiny cross-platform library for extracting information about the music/image/video that is **N**ow **P**laying on the system. **No CGo required!**

Supported OS:
- Windows (using GlobalSystemMediaTransportControlsSessionManager)
- Linux (using plain Go: DBus/MPRIS)

This is essentially [libnp](https://github.com/delthas/libnp) rewritten in plain Go.

## Usage

The API is well-documented in its [![GoDoc](https://godocs.io/github.com/delthas/go-libnp?status.svg)](https://godocs.io/github.com/delthas/go-libnp)

```
info, err := libnp.GetInfo(context.Background())
fmt.Println(info.Album)
```

## Status

Used in small-scale production environments.

The API could be slightly changed in backwards-incompatible ways for now.

- [ ] Add darwin
- [ ] Add web

## License

MIT
