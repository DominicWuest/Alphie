package image_generators

import (
	"image"
	"image/color"
	"math/rand"

	"github.com/fogleman/gg"
)

type Fluid struct {
	// The fluid's color
	fluidColor color.RGBA
	// The background color
	bgColor color.RGBA
}

func (s *Fluid) Init(seed int64) (ImageGenerator, error) {
	rand.Seed(seed)

	fluid := Fluid{
		fluidColor: color.RGBA{
			R: uint8(rand.Uint32()),
			G: uint8(rand.Uint32()),
			B: uint8(rand.Uint32()),
			A: 0xFF,
		},
	}

	fluid.bgColor = fluid.backgroundColor(fluid.fluidColor)

	return &fluid, nil
}

func (s *Fluid) Update() error {
	return nil
}

func (s *Fluid) Draw(ctx *gg.Context) (image.Image, error) {
	return ctx.Image(), nil
}

func (s *Fluid) GetFramesAmount() int {
	return 5 * 24 // ~5 seconds of playtime
}

func (s *Fluid) GetContextDimensions() (int, int) {
	return 50, 50
}

func (s *Fluid) GetPostURL() string {
	return "fluid"
}

// Returns the background color given the color of the fluid
// Use formula 0.3*R+0.6*G+0.1*B to calculate brightness, above 0.5 => use dark background
func (s *Fluid) backgroundColor(col color.RGBA) color.RGBA {
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
