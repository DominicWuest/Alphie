package image_generators

import (
	"image"
	"image/color"
	"math/rand"
	"os"
	"strconv"

	"github.com/fogleman/gg"
)

type Bounce struct {
	// The radius of the ball
	radius float64
	// Position
	ballPos [2]float64
	// Velocity
	ballVel [2]float64
	// deltaT for euler integration
	deltaT float64
	// The ball's color
	ballColor color.RGBA
	// The background color
	bgColor color.RGBA
}

func (s *Bounce) Init(seed int64) (ImageGenerator, error) {
	rand.Seed(seed)

	const maxRadius float64 = 30
	const minRadius float64 = 10

	const maxVel float64 = 150
	const minVel float64 = 65

	const deltaT float64 = 0.1

	width, height := s.GetContextDimensions()

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
		deltaT: deltaT,
		ballColor: color.RGBA{
			R: uint8(rand.Uint32()),
			G: uint8(rand.Uint32()),
			B: uint8(rand.Uint32()),
			A: 0xFF,
		},
	}

	ball.bgColor = s.backgroundColor(ball.ballColor)

	return &ball, nil
}

func (s *Bounce) Update() error {
	s.ballPos[0] += s.ballVel[0] * s.deltaT
	s.ballPos[1] += s.ballVel[1] * s.deltaT

	// Make the ball bounce off the walls
	width, height := s.GetContextDimensions()
	if s.ballPos[0]-s.radius <= 0 || s.ballPos[0]+s.radius >= float64(width) {
		s.ballVel[0] *= -1
	}
	if s.ballPos[1]-s.radius <= 0 || s.ballPos[1]+s.radius >= float64(height) {
		s.ballVel[1] *= -1
	}

	return nil
}

func (s *Bounce) Draw(ctx *gg.Context) (image.Image, error) {
	// Set the background
	ctx.SetColor(s.bgColor)
	ctx.Clear()

	// Draw the ball
	ctx.SetColor(s.ballColor)
	ctx.DrawCircle(s.ballPos[0], s.ballPos[1], s.radius)
	ctx.Fill()

	return ctx.Image(), nil
}

func (s *Bounce) GetFramesAmount() int {
	return 10 * 24 // ~10 seconds of playtime
}

func (s *Bounce) GetContextDimensions() (int, int) {
	return 250, 200
}

func (s *Bounce) GetPostURL() string {
	return "bounce"
}

func (s Bounce) GetQueueCapacity() int {
	str := os.Getenv("BOUNCE_CAP")
	if str == "" {
		return -1
	}
	cap, err := strconv.Atoi(str)
	if err != nil {
		panic("Invalid value set for BOUNCE_CAP")
	}
	return cap
}

// Returns the background color given the color of the ball
// Use formula 0.3*R+0.6*G+0.1*B to calculate brightness, above 0.5 => use dark background
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
