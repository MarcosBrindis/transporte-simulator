package ui

import (
	"fmt"
	"image/color"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// SpeedGraph muestra una gráfica de velocidad en tiempo real
type SpeedGraph struct {
	mu sync.RWMutex

	x      float32
	y      float32
	width  float32
	height float32

	maxPoints    int
	speedHistory []float32
	maxSpeed     float32

	// Colores
	colorBg     color.RGBA
	colorBorder color.RGBA
	colorLine   color.RGBA
	colorGrid   color.RGBA
	colorText   color.RGBA
}

// NewSpeedGraph crea una nueva gráfica de velocidad
func NewSpeedGraph(x, y, width, height float32, maxPoints int) *SpeedGraph {
	return &SpeedGraph{
		x:            x,
		y:            y,
		width:        width,
		height:       height,
		maxPoints:    maxPoints,
		speedHistory: make([]float32, 0, maxPoints),
		maxSpeed:     60.0, // km/h
		colorBg:      color.RGBA{30, 30, 40, 255},
		colorBorder:  color.RGBA{80, 80, 100, 255},
		colorLine:    color.RGBA{100, 200, 255, 255},
		colorGrid:    color.RGBA{50, 50, 60, 255},
		colorText:    color.RGBA{200, 200, 220, 255},
	}
}

// AddSpeed agrega un punto de velocidad
func (sg *SpeedGraph) AddSpeed(speed float32) {
	sg.mu.Lock()
	defer sg.mu.Unlock()

	sg.speedHistory = append(sg.speedHistory, speed)

	// Mantener solo los últimos maxPoints
	if len(sg.speedHistory) > sg.maxPoints {
		sg.speedHistory = sg.speedHistory[1:]
	}
}

// Clear limpia la gráfica
func (sg *SpeedGraph) Clear() {
	sg.mu.Lock()
	defer sg.mu.Unlock()

	sg.speedHistory = make([]float32, 0, sg.maxPoints)
}

// Draw dibuja la gráfica
func (sg *SpeedGraph) Draw(screen *ebiten.Image) {
	sg.mu.RLock()
	defer sg.mu.RUnlock()

	// Fondo
	vector.DrawFilledRect(screen, sg.x, sg.y, sg.width, sg.height, sg.colorBg, false)
	vector.StrokeRect(screen, sg.x, sg.y, sg.width, sg.height, 2, sg.colorBorder, false)

	// Título
	ebitenutil.DebugPrintAt(screen, "📈 VELOCIDAD (km/h)", int(sg.x+10), int(sg.y+5))

	// Dibujar grid horizontal (cada 10 km/h)
	graphY := sg.y + 25
	graphHeight := sg.height - 30

	for speed := 0.0; speed <= float64(sg.maxSpeed); speed += 10.0 {
		ratio := float32(speed) / sg.maxSpeed
		lineY := graphY + graphHeight*(1-ratio)

		// Línea de grid
		vector.StrokeLine(screen, sg.x, lineY, sg.x+sg.width, lineY, 1, sg.colorGrid, false)

		// Etiqueta
		label := int(speed)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%d", label), int(sg.x+5), int(lineY-5))
	}

	// Dibujar línea de velocidad
	if len(sg.speedHistory) < 2 {
		return
	}

	graphWidth := sg.width - 40
	pointSpacing := graphWidth / float32(sg.maxPoints-1)

	for i := 0; i < len(sg.speedHistory)-1; i++ {
		// Punto actual
		x1 := sg.x + 35 + float32(i)*pointSpacing
		speed1 := sg.speedHistory[i]
		ratio1 := speed1 / sg.maxSpeed
		if ratio1 > 1.0 {
			ratio1 = 1.0
		}
		y1 := graphY + graphHeight*(1-ratio1)

		// Punto siguiente
		x2 := sg.x + 35 + float32(i+1)*pointSpacing
		speed2 := sg.speedHistory[i+1]
		ratio2 := speed2 / sg.maxSpeed
		if ratio2 > 1.0 {
			ratio2 = 1.0
		}
		y2 := graphY + graphHeight*(1-ratio2)

		// Dibujar línea
		vector.StrokeLine(screen, x1, y1, x2, y2, 2, sg.colorLine, false)
	}

	// Dibujar velocidad actual
	if len(sg.speedHistory) > 0 {
		currentSpeed := sg.speedHistory[len(sg.speedHistory)-1]
		speedText := fmt.Sprintf("Actual: %.1f km/h", currentSpeed)
		ebitenutil.DebugPrintAt(screen, speedText, int(sg.x+sg.width-150), int(sg.y+5))
	}
}
