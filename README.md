# go-libnp [![GoDoc](https://godocs.io/github.com/delthas/go-libnp?status.svg)](https://godocs.io/github.com/delthas/go-libnp) [![stability-experimental](https://img.shields.io/badge/stability-experimental-orange.svg)](https://github.com/emersion/stability-badges#experimental)

A tiny cross-platform library for extracting information about the music/image/video that is **N**ow **P**laying on the system.

Supported OS:
- Windows (using CGo: GlobalSystemMediaTransportControlsSessionManager)
- Linux (using plain Go: DBus/MPRIS)

This is essentially Go bindings for [libnp](https://github.com/delthas/libnp), but with the Linux implementation rewritten in plain Go.

## Usage

The API is well-documented in its [![GoDoc](https://godocs.io/github.com/delthas/go-libnp?status.svg)](https://godocs.io/github.com/delthas/go-libnp)

```
info, err := libnp.GetInfo(context.Background())
fmt.Println(info.Album)
```

## Building

On Windows, building requires a MinGW toolchain, and the Windows SDK.

Make sure that GCC is in your path, and set `CPATH` to include the Windows SDK include folder:
```cmd
set "CPATH=C:\Program Files (x86)\Windows Kits\10\Include\10.0.22621\winrt"
```
Replace the version above with your SDK version.

Otherwise, if you just want to build the library with the stub backend (always returning an empty song),
just compile with `CGO_ENABLED=0`.

## Status

Used in small-scale production environments.

The API could be slightly changed in backwards-incompatible ways for now.

- [ ] Add darwin
- [ ] Add web

## License

MIT
