// Package libnp offers information about the media that is currently being played.
// The main entrypoint is GetInfo, which returns an Info.
package libnp

import (
	"context"
	"time"
)

// PlaybackType represents the kind of media being played.
// PlaybackTypeUnknown is used on some platforms.
type PlaybackType int

const (
	PlaybackTypeUnknown PlaybackType = iota
	PlaybackTypeMusic
	PlaybackTypeVideo
	PlaybackTypeImage
)

func (p PlaybackType) String() string {
	switch p {
	case PlaybackTypeMusic:
		return "Music"
	case PlaybackTypeVideo:
		return "Video"
	case PlaybackTypeImage:
		return "Image"
	default:
		return "Unknown"
	}
}

// Info stores information about the media being played.
type Info struct {
	AlbumArtists    []string
	Artists         []string
	Composers       []string  // linux only
	Genres          []string  // linux only
	Lyricists       []string  // linux only
	Created         time.Time // linux only
	FirstPlayed     time.Time // linux only
	LastPlayed      time.Time // linux only
	Album           string
	AlbumTrackCount int           // windows only
	ArtURL          string        // linux only
	BPM             int           // linux only
	DiscNumber      int           // linux only
	Length          time.Duration // linux only
	PlayCount       int           // linux only
	Subtitle        string        // windows only
	Title           string
	TrackNumber     int
	URL             string       // linux only
	PlaybackType    PlaybackType // windows only
}

// GetInfo returns Info about the media currently being played.
//
// An error is returned if the library has issues getting valid information
// about the song being played.
//
// A nil Info with a nil error is returned if there is no media being played.
func GetInfo(ctx context.Context) (*Info, error) {
	return getInfo(ctx)
}
