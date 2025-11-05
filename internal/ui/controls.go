package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Controls maneja los controles de la UI
type Controls struct {
	// Estado
	paused bool
	speed  float64

	// Colores
	colorBg     color.Color
	colorBorder color.Color
}

// NewControls crea nuevos controles
func NewControls() *Controls {
	return &Controls{
		paused:      false,
		speed:       1.0,
		colorBg:     color.RGBA{30, 30, 40, 200},
		colorBorder: color.RGBA{100, 100, 120, 255},
	}
}

// Update actualiza los controles
func (c *Controls) Update() {
	// TODO: Implementar lógica de botones
	// Por ahora solo placeholder
}

// Draw dibuja los controles
func (c *Controls) Draw(screen *ebiten.Image) {
	width := float32(screen.Bounds().Dx())
	height := float32(screen.Bounds().Dy())

	// Panel de controles (parte inferior)
	panelY := height - 60
	vector.DrawFilledRect(screen, 0, panelY, width, 60, c.colorBg, false)
	vector.StrokeLine(screen, 0, panelY, width, panelY, 2, c.colorBorder, false)

	// Placeholder de controles
	ebitenutil.DebugPrintAt(screen, "[▶ PLAY]  [⏸ PAUSE]  [⏩ 1x]  [🔄 RESET]  [🎬 Escenario: Normal]",
		20, int(panelY+20))
}
