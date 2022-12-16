package main

import (
	"context"
	"fmt"
	"github.com/delthas/go-libnp"
	"log"
	"time"
)

func main() {
	info, err := libnp.GetInfo(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	for _, v := range info.AlbumArtists {
		fmt.Printf("Album Artist: %s\n", v)
	}
	for _, v := range info.Artists {
		fmt.Printf("Artist: %s\n", v)
	}
	for _, v := range info.Composers {
		fmt.Printf("Composer: %s\n", v)
	}
	for _, v := range info.Genres {
		fmt.Printf("Genre: %s\n", v)
	}
	for _, v := range info.Lyricists {
		fmt.Printf("Lyricist: %s\n", v)
	}
	if !info.Created.IsZero() {
		fmt.Printf("Created: %s\n", info.Created.Format(time.RFC3339))
	}
	if !info.FirstPlayed.IsZero() {
		fmt.Printf("First Played: %s\n", info.FirstPlayed.Format(time.RFC3339))
	}
	if !info.LastPlayed.IsZero() {
		fmt.Printf("Last Played: %s\n", info.LastPlayed.Format(time.RFC3339))
	}
	if info.Album != "" {
		fmt.Printf("Album Title: %s\n", info.Album)
	}
	if info.AlbumTrackCount != 0 {
		fmt.Printf("Album Track Count: %d\n", info.AlbumTrackCount)
	}
	if info.ArtURL != "" {
		fmt.Printf("Art URL: %s\n", info.ArtURL)
	}
	if info.BPM != 0 {
		fmt.Printf("BPM: %d\n", info.BPM)
	}
	if info.DiscNumber != 0 {
		fmt.Printf("Disc Number: %d\n", info.DiscNumber)
	}
	if info.Length != 0 {
		fmt.Printf("Length: %d\n", info.Length/time.Second)
	}
	if info.PlayCount != 0 {
		fmt.Printf("Play Count: %d\n", info.PlayCount)
	}
	if info.Subtitle != "" {
		fmt.Printf("Subtitle: %s\n", info.Subtitle)
	}
	if info.Title != "" {
		fmt.Printf("Title: %s\n", info.Title)
	}
	if info.TrackNumber != 0 {
		fmt.Printf("Track Number: %d\n", info.TrackNumber)
	}
	if info.URL != "" {
		fmt.Printf("URL: %s\n", info.URL)
	}
	if info.PlaybackType != libnp.PlaybackTypeUnknown {
		fmt.Printf("Type: %v\n", info.PlaybackType)
	}
}
