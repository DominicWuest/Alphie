package image_generators

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"io/ioutil"
	"math/rand"
	"net/http"
	"sync"

	"github.com/fogleman/gg"
)

type Bounce struct {
	// The radius of the ball
	radius float64
	// Position
	ballPos [2]float64
	// Velocity
	ballVel [2]float64
	// The ball's color
	ballColor color.RGBA
	// The background color
	bgColor color.RGBA
}

func (s *Bounce) GenerateImage(seed int64) (string, error) {
	const baseUrl string = "/bounce"

	rand.Seed(seed)

	generatedGif, err := s.generateBounce()
	if err != nil {
		return "", err
	}
	gifAsBytes := bytes.NewBuffer([]byte{})
	gif.EncodeAll(gifAsBytes, generatedGif)
	// Send created GIF
	res, err := http.Post("http://"+cdnConnString+baseUrl, "image/gif", gifAsBytes)
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

// Returns the background color given the color of the ball
// Use formula 0.3*R+0.6*G+0.1*B to calculate brightness, above 0.5 => make background darker
func (s *Bounce) backgroundColor(col color.RGBA) color.RGBA {
	var brightBackground color.RGBA = color.RGBA{
		R: 0xCC,
		G: 0xCC,
		B: 0xCC,
		A: 0xFF,
	}

	var darkBackground color.RGBA = color.RGBA{
		R: 0x44,
		G: 0x44,
		B: 0x44,
		A: 0xFF,
	}

	var r float64 = float64(col.R)
	var g float64 = float64(col.G)
	var b float64 = float64(col.B)
	brightness := 0.3*r + 0.6*g + 0.1*b

	if brightness > 0.5*0xFF {
		return darkBackground
	}
	return brightBackground
}

func (s *Bounce) genRandomBall() Bounce {
	const maxRadius float64 = 30
	const minRadius float64 = 10

	const maxVel float64 = 150
	const minVel float64 = 65

	// Generate a radius between minRadius and maxRadius
	radius := rand.Float64()*(maxRadius-minRadius) + minRadius
	ball := Bounce{
		radius: radius,
		// Generate the balls position, making sure it is fully on screen
		ballPos: [2]float64{
			rand.Float64()*(float64(width)-2*radius) + radius,
			rand.Float64()*(float64(height)-2*radius) + radius,
		},
		// Generating the balls velocity, being in [minVel, maxVel]
		ballVel: [2]float64{
			rand.Float64()*(maxVel-minVel) + minVel,
			rand.Float64()*90 + 10,
		},
		ballColor: color.RGBA{
			R: uint8(rand.Uint32()),
			G: uint8(rand.Uint32()),
			B: uint8(rand.Uint32()),
			A: 0xFF,
		},
	}

	ball.bgColor = s.backgroundColor(ball.ballColor)

	return ball
}

// Generates the actual GIF
func (s *Bounce) generateBounce() (*gif.GIF, error) {
	const deltaT float64 = 0.1

	images := make([]*image.Paletted, frames)

	ball := s.genRandomBall()

	wg := sync.WaitGroup{}
	wg.Add(frames)

	for i := 0; i < frames; i++ {
		// Have to reset context so goroutines image.Image.At don't clash
		context := gg.NewContext(width, height)
		// Set the background
		context.SetColor(ball.bgColor)
		context.Clear()

		// Draw the ball
		context.SetColor(ball.ballColor)
		context.DrawCircle(ball.ballPos[0], ball.ballPos[1], ball.radius)
		context.Fill()

		go insertPalettedFromRGBA(context.Image(), i, images, &wg)

		// Update the ball
		ball.ballPos[0] += ball.ballVel[0] * deltaT
		ball.ballPos[1] += ball.ballVel[1] * deltaT

		// Make the ball bounce off the walls
		if ball.ballPos[0]-ball.radius <= 0 || ball.ballPos[0]+ball.radius >= float64(width) {
			ball.ballVel[0] *= -1
		}
		if ball.ballPos[1]-ball.radius <= 0 || ball.ballPos[1]+ball.radius >= float64(height) {
			ball.ballVel[1] *= -1
		}
	}
	wg.Wait()

	return &gif.GIF{
		Image: images,
		Delay: delays,
	}, nil
}
