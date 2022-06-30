package image_generators

import (
	"image"
	"image/color"
	"math"
	"math/rand"

	"github.com/fogleman/gg"
)

type Fluid struct {
	// The density of the fluid
	densities *[][]float64
	// The components of the fluid's velocity
	velocityX *[][]float64
	velocityY *[][]float64
	// The sources of the fluid
	sources *[]fluidSource
	// deltaT
	dt float64
	// The diffusion constant as defined in the paper
	diff float64
	// The fluid's color
	fluidColor color.RGBA
	// The background color
	bgColor color.RGBA
}

type fluidSource struct {
	x    int
	y    int
	rate float64
}

func (s *Fluid) Init(seed int64) (ImageGenerator, error) {
	rand.Seed(seed)

	const (
		// How many fluid sources should be distributed over the grid
		minSources, maxSources int = 3, 20
		// How much fluid the source produces per time-step
		minSourceFlow, maxSourceFlow float64 = 0.01, 0.05

		minVelocity, maxVelocity float64 = -5, 5

		dt float64 = 0.1

		diff float64 = 0.5
	)

	width, height := s.getGridDimensions()
	minDensity, maxDensity := s.getDensityInterval()

	densities := s.createRandomMatrix(width+2, height+2, minDensity, maxDensity)

	velocityX := s.createRandomMatrix(width+2, height+2, minVelocity, maxVelocity)
	velocityY := s.createRandomMatrix(width+2, height+2, minVelocity, maxVelocity)

	// Initialise all the fluid sources
	sourcesCount := rand.Intn(maxSources-minSources) + minSources
	sources := make([]fluidSource, sourcesCount*2)
	for i := 0; i < sourcesCount; i++ {
		x, y := rand.Intn(width)+1, rand.Intn(height)+1
		rate := rand.Float64()*(maxSourceFlow-minSourceFlow) + minSourceFlow
		sources = append(sources, fluidSource{
			x:    x,
			y:    y,
			rate: rate,
		})
	}

	fluid := Fluid{
		densities: &densities,
		velocityX: &velocityX,
		velocityY: &velocityY,
		sources:   &sources,
		dt:        dt,
		diff:      diff,
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
	s.velocityStep()
	s.densityStep()
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

func (s Fluid) getGridDimensions() (int, int) {
	return 150, 100
}

func (s Fluid) getDensityInterval() (float64, float64) {
	return 0, 1
}

func (s Fluid) createEmptyMatrix(width, height int) [][]float64 {
	arr := make([]float64, width*height)

	matrix := make([][]float64, width)

	for i := 0; i < height; i++ {
		matrix[i] = arr[i*width : (i+1)*width]
	}

	return matrix
}

func (s Fluid) createRandomMatrix(width, height int, minVal, maxVal float64) [][]float64 {
	matrix := s.createEmptyMatrix(width, height)

	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			matrix[x][y] = rand.Float64()*(maxVal-minVal) + minVal
		}
	}

	return matrix
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

func (s *Fluid) addSource() {
	for _, source := range *s.sources {
		(*s.densities)[source.x][source.y] += s.dt * source.rate
	}
}

func (s *Fluid) diffuse() {
	const (
		gaussSeidelIterations int = 20
	)

	width, height := s.getGridDimensions()

	a := s.dt * s.diff * float64(width) * float64(height)

	nextDensities := *s.densities
	for i := 0; i < gaussSeidelIterations; i++ {
		for x := 1; x <= width; x++ {
			for y := 1; y <= height; y++ {
				nextDensities[x][y] = nextDensities[x+1][y] + nextDensities[x-1][y] + nextDensities[x][y+1] + nextDensities[x][y-1]
				nextDensities[x][y] *= a
				nextDensities[x][y] += (*s.densities)[x][y]
				nextDensities[x][y] /= (1 + 4*a)
			}
		}
	}
}

func (s *Fluid) advect() {
	width, height := s.getGridDimensions()

	dt0 := s.dt * float64(width+height) / 2

	for x := 1; x <= width; x++ {
		for y := 1; y <= width; y++ {
			prevX := float64(x) - dt0*(*s.velocityX)[x][y]
			prevY := float64(y) - dt0*(*s.velocityY)[x][y]

			oldDensities := (*s.densities)

			if prevX < 0.5 {
				prevX = 0.5
			} else if prevX > float64(width)+0.5 {
				prevX = float64(width) + 0.5
			}

			if prevY < 0.5 {
				prevY = 0.5
			} else if prevY > float64(height)+0.5 {
				prevY = float64(width) + 0.5
			}

			// Split prevX into integer and fractional part
			tmp, fractX := math.Modf(prevX)
			floorX := int(tmp)
			tmp, fractY := math.Modf(prevY)
			floorY := int(tmp)

			(*s.densities)[x][y] = fractX * (fractY*oldDensities[floorX][floorY] + (1-fractY)*oldDensities[floorX][floorY+1])
			(*s.densities)[x][y] += (1 - fractX) * (fractY*oldDensities[floorX+1][floorY] + (1-fractY)*oldDensities[floorX+1][floorY+1])
			(*s.densities)[x][y] *= dt0
		}
	}

}

func (s *Fluid) densityStep() {

}

func (s *Fluid) velocityStep() {

}

func (s *Fluid) project() {

}
