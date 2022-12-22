package libnp

import (
	"context"
	"fmt"
	"github.com/godbus/dbus/v5"
	"strings"
	"time"
)

func getInfo(ctx context.Context) (*Info, error) {
	bus, err := dbus.ConnectSessionBus(dbus.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("opening session bus: %v", err)
	}
	defer bus.Close()

	var names []string
	if err := bus.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.ListNames", 0).Store(&names); err != nil {
		return nil, fmt.Errorf("listing dbus names: %v", err)
	}
	var r *Info
	for _, name := range names {
		if !strings.HasPrefix(name, "org.mpris.MediaPlayer2.") {
			continue
		}
		var playing bool
		o := bus.Object(name, "/org/mpris/MediaPlayer2")
		var status string
		if err := o.CallWithContext(ctx, "org.freedesktop.DBus.Properties.Get", 0, "org.mpris.MediaPlayer2.Player", "PlaybackStatus").Store(&status); err != nil {
			continue
		}
		switch status {
		case "Playing":
			playing = true
		case "Paused":
			playing = false
		default:
			continue
		}
		var metadata map[string]dbus.Variant
		if err := o.CallWithContext(ctx, "org.freedesktop.DBus.Properties.Get", 0, "org.mpris.MediaPlayer2.Player", "Metadata").Store(&metadata); err != nil {
			continue
		}
		var info Info
		if e, ok := metadata["mpris:length"].Value().(int64); ok {
			info.Length = time.Duration(e) * time.Microsecond
		}
		if e, ok := metadata["mpris:artUrl"].Value().(string); ok {
			info.ArtURL = e
		}
		if e, ok := metadata["xesam:album"].Value().(string); ok {
			info.Album = e
		}
		if e, ok := metadata["xesam:albumArtist"].Value().([]string); ok && len(e) > 0 {
			info.AlbumArtists = e
		}
		if e, ok := metadata["xesam:artist"].Value().([]string); ok && len(e) > 0 {
			info.Artists = e
		}
		if e, ok := metadata["xesam:audioBPM"].Value().(int); ok {
			info.BPM = e
		}
		if e, ok := metadata["xesam:composer"].Value().([]string); ok && len(e) > 0 {
			info.Composers = e
		}
		if e, ok := metadata["xesam:contentCreated"].Value().(string); ok {
			if t, err := time.Parse(time.RFC3339, e); err == nil {
				info.Created = t
			}
		}
		if e, ok := metadata["xesam:discNumber"].Value().(int); ok {
			info.DiscNumber = e
		}
		if e, ok := metadata["xesam:firstUsed"].Value().(string); ok {
			if t, err := time.Parse(time.RFC3339, e); err == nil {
				info.FirstPlayed = t
			}
		}
		if e, ok := metadata["xesam:genre"].Value().([]string); ok && len(e) > 0 {
			info.Genres = e
		}
		if e, ok := metadata["xesam:lastUsed"].Value().(string); ok {
			if t, err := time.Parse(time.RFC3339, e); err == nil {
				info.LastPlayed = t
			}
		}
		if e, ok := metadata["xesam:lyricist"].Value().([]string); ok && len(e) > 0 {
			info.Lyricists = e
		}
		if e, ok := metadata["xesam:title"].Value().(string); ok {
			info.Title = e
		}
		if e, ok := metadata["xesam:trackNumber"].Value().(int); ok {
			info.TrackNumber = e
		}
		if e, ok := metadata["xesam:url"].Value().(string); ok {
			info.URL = e
		}
		if e, ok := metadata["xesam:useCount"].Value().(int); ok {
			info.PlayCount = e
		}
		r = &info
		if playing {
			return r, nil
		}
	}
	return r, nil
}
