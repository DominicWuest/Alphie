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
	// The components of the field's forces
	forceX *[][]float64
	forceY *[][]float64
	// The sources of the fluid
	sources *[]fluidSource
	// deltaT
	dt float64
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
		minSources, maxSources int = 25, 50
		// How much fluid the source produces per time-step
		minSourceFlow, maxSourceFlow float64 = 1000, 2000

		minForce, maxForce float64 = -50, 50

		dt float64 = 0.01

		// How many frames to simulate before we start drawing
		preSimulationSteps int = 24
	)

	width, height := s.getGridDimensions()

	densities := s.createEmptyMatrix(width+2, height+2)

	velocityX := s.createEmptyMatrix(width+2, height+2)
	velocityY := s.createEmptyMatrix(width+2, height+2)

	forceX := s.createRandomMatrix(width+2, height+2, minForce, maxForce)
	forceY := s.createRandomMatrix(width+2, height+2, minForce, maxForce)

	// Initialise all the fluid sources
	sourcesCount := rand.Intn(maxSources-minSources) + minSources
	sources := make([]fluidSource, sourcesCount)
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
		forceX:    &forceX,
		forceY:    &forceY,
		sources:   &sources,
		dt:        dt,
		fluidColor: color.RGBA{
			R: uint8(rand.Uint32()),
			G: uint8(rand.Uint32()),
			B: uint8(rand.Uint32()),
			A: 0xFF,
		},
	}

	fluid.bgColor = fluid.backgroundColor(fluid.fluidColor)

	for i := 0; i < preSimulationSteps; i++ {
		if err := fluid.Update(); err != nil {
			return nil, err
		}
	}

	return &fluid, nil
}

func (s *Fluid) Update() error {
	s.velocityStep()
	s.densityStep()
	return nil
}

func (s *Fluid) Draw(ctx *gg.Context) (image.Image, error) {
	width, height := s.getGridDimensions()

	minDensity, maxDensity := s.getDensityInterval()
	densityInterval := maxDensity - minDensity

	ctx.SetColor(s.bgColor)
	ctx.Clear()

	for x := 1; x <= width; x++ {
		for y := 1; y <= height; y++ {
			normalisedDensity := 250 * (((*s.densities)[x][y]-minDensity)/densityInterval - (densityInterval / 2))
			sigmoid := 1 / (1 + math.Exp(-normalisedDensity))

			color := s.fluidColor
			ctx.SetRGBA255(int(color.R), int(color.G), int(color.B), int(sigmoid*255))
			ctx.DrawCircle(float64(x-1), float64(y-1), 1)
			ctx.Fill()
		}
	}
	return ctx.Image(), nil
}

func (s *Fluid) GetFramesAmount() int {
	return 5 * 24 // ~5 seconds of playtime
}

func (s *Fluid) GetContextDimensions() (int, int) {
	return 150, 100
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

	for i := 0; i < width; i++ {
		matrix[i] = arr[i*height : (i+1)*height]
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

func (s *Fluid) densityStep() {
	const (
		diff float64 = 50
	)
	s.addSource()
	s.diffuse(s.densities, diff, 0)
	s.advect(s.densities, *s.velocityX, *s.velocityY, 0)
}

func (s *Fluid) velocityStep() {
	const (
		viscosity float64 = 50
	)

	s.addForce(*s.forceX, s.velocityX)
	s.addForce(*s.forceY, s.velocityY)

	s.diffuse(s.forceX, viscosity, 1)
	s.diffuse(s.forceY, viscosity, 2)

	s.project()

	s.advect(s.velocityX, *s.forceX, *s.forceY, 1)
	s.advect(s.velocityY, *s.forceX, *s.forceY, 2)

	s.project()
}

func (s *Fluid) addSource() {
	for _, source := range *s.sources {
		(*s.densities)[source.x][source.y] += s.dt * source.rate
	}
}

func (s Fluid) diffuse(field *[][]float64, diff float64, situation int) {
	const (
		gaussSeidelIterations int = 20
	)

	width, height := s.getGridDimensions()

	a := s.dt * diff

	nextDensities := s.createEmptyMatrix(width+2, height+2)
	for i := 0; i < gaussSeidelIterations; i++ {
		for x := 1; x <= width; x++ {
			for y := 1; y <= height; y++ {
				nextDensities[x][y] = nextDensities[x+1][y] + nextDensities[x-1][y] + nextDensities[x][y+1] + nextDensities[x][y-1]
				nextDensities[x][y] *= a
				nextDensities[x][y] += (*field)[x][y]
				nextDensities[x][y] /= 1 + 4*a
			}
		}
		s.setBoundary(situation, &nextDensities)
	}
	*field = nextDensities
}

func (s *Fluid) advect(dest *[][]float64, xChange, yChange [][]float64, situation int) {
	width, height := s.getGridDimensions()

	dt0 := s.dt * float64(width+height) / 2
	oldDensities := (*s.densities)

	for x := 1; x <= width; x++ {
		for y := 1; y <= height; y++ {
			prevX := float64(x) - dt0*(*s.velocityX)[x][y]
			prevY := float64(y) - dt0*(*s.velocityY)[x][y]

			if prevX < 0.5 {
				prevX = 0.5
			} else if prevX > float64(width)+0.5 {
				prevX = float64(width) + 0.5
			}

			if prevY < 0.5 {
				prevY = 0.5
			} else if prevY > float64(height)+0.5 {
				prevY = float64(height) + 0.5
			}

			// Split prevX into integer and fractional part
			tmp, fractX := math.Modf(prevX)
			floorX := int(tmp)
			tmp, fractY := math.Modf(prevY)
			floorY := int(tmp)

			(*s.densities)[x][y] = (1 - fractX) * ((1-fractY)*oldDensities[floorX][floorY] + fractY*oldDensities[floorX][floorY+1])
			(*s.densities)[x][y] += fractX * ((1-fractY)*oldDensities[floorX+1][floorY] + fractY*oldDensities[floorX+1][floorY+1])
		}
	}
	s.setBoundary(situation, dest)
}

func (s *Fluid) addForce(source [][]float64, dest *[][]float64) {
	for i := range source {
		for j := range source[i] {
			(*dest)[i][j] += s.dt * source[i][j]
		}
	}
}

func (s *Fluid) project() {
	const (
		gaussSeidelIterations int = 20
	)

	width, height := s.getGridDimensions()

	h := 1 / float64(width)

	// Calculate the divergence of the points
	divergence := s.createEmptyMatrix(width+2, height+2)
	for x := 1; x <= width; x++ {
		for y := 1; y <= height; y++ {
			divergence[x][y] = -0.5 * h * ((*s.velocityX)[x+1][y] - (*s.velocityX)[x-1][y] + (*s.velocityY)[x][y+1] - (*s.velocityY)[x][y-1])
		}
	}
	s.setBoundary(0, &divergence)

	// Calculate the p-values using GaussSeidel relaxation
	pValues := s.createEmptyMatrix(width+2, height+2)
	for i := 0; i < gaussSeidelIterations; i++ {
		for x := 1; x <= width; x++ {
			for y := 1; y <= height; y++ {
				pValues[x][y] = divergence[x][y] + pValues[x-1][y] + pValues[x+1][y] + pValues[x][y+1] + pValues[x][y-1]
				pValues[x][y] /= 4
			}
		}
		s.setBoundary(0, &pValues)
	}

	// Subtract the gradient from the velocities
	for x := 1; x <= width; x++ {
		for y := 1; y <= height; y++ {
			(*s.velocityX)[x][y] -= 0.5 * (pValues[x+1][y] - pValues[x-1][y]) / h
			(*s.velocityY)[x][y] -= 0.5 * (pValues[x][y+1] - pValues[x][y-1]) / h
		}
	}
	s.setBoundary(1, s.velocityX)
	s.setBoundary(2, s.velocityY)
}

func (s Fluid) setBoundary(situation int, target *[][]float64) {
	width, height := s.getGridDimensions()

	for x := 1; x <= width; x++ {
		(*target)[x][0] = (*target)[x][1]
		(*target)[x][height+1] = (*target)[x][height]
		if situation == 2 {
			(*target)[x][0] *= -1
			(*target)[x][height+1] *= -1
		}
	}
	for y := 1; y <= height; y++ {
		(*target)[0][y] = (*target)[1][y]
		(*target)[width+1][y] = (*target)[width][y]
		if situation == 1 {
			(*target)[0][y] *= -1
			(*target)[width+1][y] *= -1
		}
	}
	(*target)[0][0] = 0.5 * ((*target)[1][0] + (*target)[0][1])
	(*target)[0][height+1] = 0.5 * ((*target)[1][height+1] + (*target)[0][height])
	(*target)[width+1][0] = 0.5 * ((*target)[width][0] + (*target)[width+1][1])
	(*target)[width+1][height+1] = 0.5 * ((*target)[width][height+1] + (*target)[width+1][height])
}
