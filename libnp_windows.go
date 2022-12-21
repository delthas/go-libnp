//go:build cgo

package libnp

// #cgo CFLAGS: -Ilibnp/include
// #cgo LDFLAGS: -lruntimeobject
// #include <np.h>
// #include <libnp/np_windows.c>
// #include <libnp/np_util.c>
import "C"

import (
	"context"
	"runtime"
	"time"
	"unsafe"
)

// dummy object to represent the global np state to destroy
var global []interface{}

func init() {
	C.np_init()
	global = []interface{}{}
	runtime.SetFinalizer(&global, func(_ interface{}) {
		C.np_destroy()
	})
}

func parseArray(p **C.char, size C.size_t) []string {
	if p == nil || int(size) == 0 {
		return nil
	}
	buf := (*[1 << 30]*C.char)(unsafe.Pointer(p)) // see: https://groups.google.com/g/golang-nuts/c/sV_f0VkjZTA
	b := make([]string, int(size))
	for i := range b {
		b[i] = C.GoString(buf[i])
	}
	return b
}

func parseTime(p *C.char) time.Time {
	if p == nil {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, C.GoString(p))
	if err != nil {
		return time.Time{}
	}
	return t
}

func getInfo(ctx context.Context) (*Info, error) {
	ci := C.np_info_get()
	if ci == nil {
		return nil, nil
	}
	info := &Info{
		AlbumArtists:    parseArray(ci.album_artists, ci.album_artists_size),
		Artists:         parseArray(ci.artists, ci.artists_size),
		Composers:       parseArray(ci.composers, ci.composers_size),
		Genres:          parseArray(ci.genres, ci.genres_size),
		Lyricists:       parseArray(ci.lyricists, ci.lyricists_size),
		Created:         parseTime(ci.created),
		FirstPlayed:     parseTime(ci.first_played),
		LastPlayed:      parseTime(ci.last_played),
		Album:           C.GoString(ci.album),
		AlbumTrackCount: int(ci.album_track_count),
		ArtURL:          C.GoString(ci.art_url),
		BPM:             int(ci.bpm),
		DiscNumber:      int(ci.disc_number),
		Length:          time.Duration(ci.length) * time.Microsecond,
		PlayCount:       int(ci.play_count),
		Subtitle:        C.GoString(ci.subtitle),
		Title:           C.GoString(ci.title),
		TrackNumber:     int(ci.track_number),
		URL:             C.GoString(ci.url),
		PlaybackType:    PlaybackType(int(ci.playback_type)),
	}
	C.np_info_destroy(ci)
	return info, nil
}
