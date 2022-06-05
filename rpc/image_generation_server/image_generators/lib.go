package image_generators

import (
	"image"
	"image/color/palette"
	"image/draw"
	"os"
)

var cdnHostname string
var cdnPort string
var cdnConnString string

// Dimensions of the GIF
const width int = 250
const height int = 200

// Delay between each frame (results in ~24 FPS)
const delay int = 100 / 24

// Amount of frames to be displayed (results in ~10 seconds playtime)
const frames int = 10 * 24

var delays []int

// Struct for the response we get after posting a GIF to the CDN server
type postResponse struct {
	Filename string `json:"filename"`
}

func Init() {
	// Init the constants given by env variables
	hostname := os.Getenv("CDN_HOSTNAME")
	port := os.Getenv("CDN_REST_PORT")
	if len(hostname)*len(port) == 0 {
		panic("No CDN_HOSTNAME or CDN_REST_PORT set")
	}
	cdnHostname = hostname
	cdnPort = port
	cdnConnString = cdnHostname + ":" + cdnPort

	// Init the delays
	for i := 0; i < frames; i++ {
		delays = append(delays, delay)
	}
}

// Converts an RGBA image to a paletted image
// This is needed, as the drawing library returns an image.Image but the gif library requires an image.Paletted
func rgbaToPaletted(img image.Image) *image.Paletted {
	bounds := img.Bounds()
	dst := image.NewPaletted(bounds, palette.WebSafe)
	draw.Draw(dst, bounds, img, bounds.Min, draw.Src)
	return dst
}
