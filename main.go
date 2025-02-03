package main

import (
	"log"
	"fmt"
	"os"
	"image/png"
	"image"
	"net/url"
	"syscall"

	"golang.org/x/image/draw"
	"golang.org/x/term"

	"github.com/godbus/dbus/v5"
)

const (
	ANSI_BG_COLOR = "\x1b[48;2;%d;%d;%dm"
	ANSI_FG_COLOR = "\x1b[38;2;%d;%d;%dm▄"
	ANSI_RESET    = "\x1b[0m"
	ANSI_CLEAR    = "\x1b[2J"
	ANSI_HOME     = "\x1b[H"
)

var (
	ANSI_STATUS_SYM = map[string]string{
		"Playing": "⏵",
		"Paused":  "⏸",
		"Stopped": "⏹",
	}
	album_art_max int = 20

	// global state used for rendering
	art    string
	title  string
	artist string
	album  string
	state  string = "Playing"
)

func get(m map[string]dbus.Variant, key string) string {
	if newtitle, ok := m[key]; ok {
		v := newtitle.Value()
		if v == nil {
			return ""
		}
		return v.(string)
	}
	return ""
}

func getone(m map[string]dbus.Variant, key string) string {
	if newtitle, ok := m[key]; ok {
		v := newtitle.Value()
		if v == nil {
			return ""
		}
		return v.([]string)[0]
	}
	return ""
}

func render() {
	_, h, err := term.GetSize(syscall.Stdout)
	if err == nil {
		album_art_max = h * 2
	}

	dst := image.NewRGBA(image.Rect(0, 0, album_art_max, album_art_max))

	// Try and parse supplied media url, otherwise use our blank image
	if u, err := url.Parse(art); err == nil {
		f, err := os.Open(u.Path)
		if err == nil {
			img, err := png.Decode(f)
			if err == nil {
				draw.NearestNeighbor.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)
			}
			f.Close()
		}
	}
	fmt.Print(ANSI_CLEAR, ANSI_HOME)

	// Use half square unicode for denser image vertical rendering
	for y := dst.Bounds().Min.Y; y < (dst.Bounds().Max.Y - dst.Bounds().Max.Y%2); y += 2 {
		for x := dst.Bounds().Min.X; x < dst.Bounds().Max.X; x++ {
			// Top half
			r, g, b, _ := dst.At(x, y).RGBA()
			fmt.Printf(ANSI_BG_COLOR, r>>8, g>>8, b>>8)
			// Bottom half
			r, g, b, _ = dst.At(x, y+1).RGBA()
			fmt.Printf(ANSI_FG_COLOR, r>>8, g>>8, b>>8)
		}
		fmt.Print(ANSI_RESET)
		// Playing media information & newlines except for last
		switch y {
		case 0:
			fmt.Printf(" \x1b[1m%s\x1b[0m\n", title)
		case 2:
			fmt.Printf(" %s\n", artist)
		case 4:
			fmt.Printf(" %s\n", album)
		case 6:
			fmt.Printf(" \x1b[1m%s\x1b[0m\n", ANSI_STATUS_SYM[state])
		case album_art_max - 2:
			// no newline
		default:
			fmt.Printf("\n")
		}
	}

}

func main() {
	ch := make(chan *dbus.Signal, 100)
	sigconn, err := dbus.ConnectSessionBus()
	if err != nil {
		log.Fatal(err)
	}

	sigconn.AddMatchSignal(
		dbus.WithMatchObjectPath("/org/mpris/MediaPlayer2"),
		dbus.WithMatchInterface("org.freedesktop.DBus.Properties"),
		dbus.WithMatchMember("PropertiesChanged"),
	)
	sigconn.Signal(ch)

	for {
		signal, ok := <-ch
		if !ok {
			return
		}
		switch signal.Name {
		case "org.freedesktop.DBus.Properties.PropertiesChanged":
			interface_name := signal.Body[0].(string)
			changed_properties := signal.Body[1].(map[string]dbus.Variant)
			_ = signal.Body[2].([]string) // we dont use invalidated_properties

			if interface_name != "org.mpris.MediaPlayer2.Player" {
				log.Fatalf("Captured bad signals, has the DBUS api changed?\n%v\n", signal)
			}

			if newstate, ok := changed_properties["PlaybackStatus"]; ok {
				state = newstate.Value().(string)
			}

			if metadata, ok := changed_properties["Metadata"]; ok {
				songmeta := metadata.Value().(map[string]dbus.Variant)

				art = get(songmeta, "mpris:artUrl")
				album = get(songmeta, "xesam:album")
				artist = getone(songmeta, "xesam:artist")
				title = get(songmeta, "xesam:title")
			}

			render()
		}
	}
}

/*
map[string]dbus.Variant{
"mpris:artUrl":dbus.Variant{sig:dbus.Signature{str:"s"}, value:"file:///home/smcgregor/.mozilla/firefox/firefox-mpris/4531_241.png"},
"mpris:length":dbus.Variant{sig:dbus.Signature{str:"x"}, value:217000000},
"mpris:trackid":dbus.Variant{sig:dbus.Signature{str:"o"}, value:"/org/mpris/MediaPlayer2/firefox"},
"xesam:album":dbus.Variant{sig:dbus.Signature{str:"s"}, value:"You're Welcome"},
"xesam:artist":dbus.Variant{sig:dbus.Signature{str:"as"}, value:[]string{"A Day To Remember"}},
"xesam:title":dbus.Variant{sig:dbus.Signature{str:"s"}, value:"Viva La Mexico"},
"xesam:url":dbus.Variant{sig:dbus.Signature{str:"s"}, value:"https://music.youtube.com/watch?v=9OK6_7qLhzY&list=MLCT"}
}

*/
