package image_generators

import (
	"image"
	"image/draw"
	"os"
	"sync"

	"github.com/andybons/gogif"
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

/*
  Converts an RGBA image to a paletted image
  This is needed, as the drawing library returns an image.Image but the gif library requires an image.Paletted
  It is to be executed as a goroutine, as the Draw function is rather slow
  The function inserts the image at the provided index in the images array and calls Done on the provided WaitGroup
*/
func insertPalettedFromRGBA(img image.Image, index int, images []*image.Paletted, wg *sync.WaitGroup) {
	defer wg.Done()

	bounds := img.Bounds()
	dst := image.NewPaletted(bounds, nil)
	quantizer := gogif.MedianCutQuantizer{NumColor: 64}
	quantizer.Quantize(dst, bounds, img, image.Point{})
	draw.Draw(dst, bounds, img, bounds.Min, draw.Src)

	images[index] = dst
}
