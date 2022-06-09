package image_generators

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/gif"
	"io/ioutil"
	"net/http"
	"os"
	"sync"

	pb "github.com/DominicWuest/Alphie/rpc/image_generation_server/image_generation_pb"
	"github.com/andybons/gogif"
	"github.com/fogleman/gg"
)

var cdnHostname string
var cdnPort string
var cdnConnString string

// Delay between each frame (results in ~24 FPS)
const delay int = 100 / 24

// Struct for the response we get after posting a GIF to the CDN server
type postResponse struct {
	Filename string `json:"filename"`
}

type ImageGenerator interface {
	// Initialise the generator
	Init(int64) (ImageGenerator, error)

	// Gets called sequentially once per frame
	Update() error
	Draw(*gg.Context) (image.Image, error)

	// Getters for constants
	GetFramesAmount() int
	GetContextDimensions() (int, int) // Width x Height
	GetPostURL() string
}

// Initialises the constants given by env variables
func Init() {
	hostname := os.Getenv("CDN_HOSTNAME")
	port := os.Getenv("CDN_REST_PORT")
	if len(hostname)*len(port) == 0 {
		panic("No CDN_HOSTNAME or CDN_REST_PORT set")
	}
	cdnHostname = hostname
	cdnPort = port
	cdnConnString = cdnHostname + ":" + cdnPort
}

// Init the delays array with the given amount of frames
func createDelayArray(frames int) []int {
	delays := make([]int, frames)
	for i := 0; i < frames; i++ {
		delays[i] = delay
	}
	return delays
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

// Sends the provided GIF to the CDN-server via a post request to the provided URL
// Returns the URL where the GIF can be accessed from
func postGIF(url string, inputGif *gif.GIF) (string, error) {
	// Convert GIF to byte buffer
	gifAsBytes := bytes.NewBuffer([]byte{})
	if err := gif.EncodeAll(gifAsBytes, inputGif); err != nil {
		return "", err
	}

	// Send created GIF
	res, err := http.Post("http://"+cdnConnString+"/"+url, "image/gif", gifAsBytes)
	if err != nil {
		return "", err
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to post created gif: %+v", res)
	}

	// Read response / where GIF was stored
	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	response := postResponse{}
	if err := json.Unmarshal(content, &response); err != nil {
		return "", err
	}

	return response.Filename, nil
}

// Generates the image from the passed generator
func GenerateImage(in *pb.ImageRequest, generator ImageGenerator, seed int64) (*pb.ImageResponse, error) {
	generator, err := generator.Init(seed)
	if err != nil {
		return nil, err
	}

	frames := generator.GetFramesAmount()
	images := make([]*image.Paletted, frames)

	wg := sync.WaitGroup{}
	wg.Add(frames)

	width, height := generator.GetContextDimensions()
	for i := 0; i < frames; i++ {
		if err = generator.Update(); err != nil {
			return nil, err
		}

		context := gg.NewContext(width, height)
		im, err := generator.Draw(context)
		if err != nil {
			return nil, err
		}

		go insertPalettedFromRGBA(im, i, images, &wg)
	}
	wg.Wait()

	delays := createDelayArray(frames)
	gif := &gif.GIF{
		Image: images,
		Delay: delays,
	}

	postUrl := generator.GetPostURL()
	path, err := postGIF(postUrl, gif)
	if err != nil {
		return nil, err
	}

	return &pb.ImageResponse{ContentPath: path}, nil
}
