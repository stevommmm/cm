package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"net/url"
	"os"
	"syscall"
	"time"

	"github.com/leberKleber/go-mpris"
	"golang.org/x/image/draw"
	"golang.org/x/term"
)

const (
	ANSI_BG_COLOR = "\x1b[48;2;%d;%d;%dm"
	ANSI_FG_COLOR = "\x1b[38;2;%d;%d;%dm▄"
	ANSI_RESET    = "\x1b[0m"
	ANSI_CLEAR    = "\x1b[2J"
	ANSI_HOME     = "\x1b[H"
)

var (
	ANSI_STATUS_SYM = map[mpris.PlaybackStatus]string{
		mpris.PlaybackStatusPlaying: "⏵",
		mpris.PlaybackStatusPaused:  "⏸",
		mpris.PlaybackStatusStopped: "⏹",
	}
	mpris_player  string
	album_art_max int = 20
)

func renderLoop(p *mpris.Player) {
	m, err := p.Metadata()
	if err != nil {
		return
	}
	_, h, err := term.GetSize(syscall.Stdout)
	if err == nil {
		album_art_max = h * 2
	}

	dst := image.NewRGBA(image.Rect(0, 0, album_art_max, album_art_max))

	// Try and parse supplied media url, otherwise use our blank image
	if art, err := m.MPRISArtURL(); err == nil {
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
			title, _ := m.XESAMTitle()
			fmt.Printf(" \x1b[1m%s\x1b[0m\n", title)
		case 2:
			artists, _ := m.XESAMArtist()
			fmt.Printf(" %s\n", artists[0])
		case 4:
			album, _ := m.XESAMAlbum()
			fmt.Printf(" %s\n", album)
		case 6:
			status, err := p.PlaybackStatus()
			if err == nil {
				fmt.Printf(" \x1b[1m%s\x1b[0m\n", ANSI_STATUS_SYM[status])
			}
		case album_art_max - 2:
			// no newline
		default:
			fmt.Printf("\n")
		}
	}
}

func main() {
	flag.StringVar(&mpris_player, "player", "org.mpris.MediaPlayer2.firefox.instance_1_32", "DBUS player name")
	flag.IntVar(&album_art_max, "art", 20, "Album art pixel (re)size")
	flag.Parse()
	p, err := mpris.NewPlayer(mpris_player)
	if err != nil {
		log.Fatal(err)
	}
	defer p.Close()

	for {
		renderLoop(&p)
		time.Sleep(time.Second * 10)
	}
}
