package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// ScenarioOption representa una opción de escenario
type ScenarioOption struct {
	ID   string
	Name string
}

// ScenarioSelector es un dropdown para seleccionar escenarios
type ScenarioSelector struct {
	x      float32
	y      float32
	width  float32
	height float32

	options       []ScenarioOption
	selectedIndex int
	isOpen        bool
	hoveredIndex  int

	// Colores
	colorBg         color.RGBA
	colorBgHover    color.RGBA
	colorBorder     color.RGBA
	colorText       color.RGBA
	colorDropdownBg color.RGBA
}

// NewScenarioSelector crea un nuevo selector de escenarios
func NewScenarioSelector(x, y, width, height float32) *ScenarioSelector {
	return &ScenarioSelector{
		x:               x,
		y:               y,
		width:           width,
		height:          height,
		selectedIndex:   0,
		isOpen:          false,
		hoveredIndex:    -1,
		colorBg:         color.RGBA{60, 60, 80, 255},
		colorBgHover:    color.RGBA{80, 80, 100, 255},
		colorBorder:     color.RGBA{100, 100, 120, 255},
		colorText:       color.RGBA{255, 255, 255, 255},
		colorDropdownBg: color.RGBA{40, 40, 60, 255},
		options: []ScenarioOption{
			{ID: "parada_normal", Name: "Parada Normal"},
			{ID: "parada_con_salidas", Name: "Parada con Salidas"},
			{ID: "circuito_completo", Name: "Circuito Completo"},
		},
	}
}

// Update actualiza el selector
// Update actualiza el selector
func (ss *ScenarioSelector) Update() (changed bool, selectedID string) {
	mouseX, mouseY := ebiten.CursorPosition()
	mx := float32(mouseX)
	my := float32(mouseY)

	// Verificar click en el botón principal
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		// Click en el selector principal
		if mx >= ss.x && mx <= ss.x+ss.width &&
			my >= ss.y && my <= ss.y+ss.height {
			ss.isOpen = !ss.isOpen
			return false, ""
		}

		// Click en una opción del dropdown
		if ss.isOpen {
			numOptions := len(ss.options)

			for i := range ss.options {
				// ← CAMBIO: Calcular Y hacia arriba
				reverseIndex := numOptions - 1 - i
				optY := ss.y - float32(reverseIndex+1)*ss.height

				if mx >= ss.x && mx <= ss.x+ss.width &&
					my >= optY && my <= optY+ss.height {

					if i != ss.selectedIndex {
						ss.selectedIndex = i
						ss.isOpen = false
						return true, ss.options[i].ID
					}
					ss.isOpen = false
					return false, ""
				}
			}

			// Click fuera del dropdown - cerrar
			ss.isOpen = false
		}
	}

	// Actualizar hover
	if ss.isOpen {
		ss.hoveredIndex = -1
		numOptions := len(ss.options)

		for i := range ss.options {
			// ← CAMBIO: Calcular Y hacia arriba
			reverseIndex := numOptions - 1 - i
			optY := ss.y - float32(reverseIndex+1)*ss.height

			if mx >= ss.x && mx <= ss.x+ss.width &&
				my >= optY && my <= optY+ss.height {
				ss.hoveredIndex = i
				break
			}
		}
	}

	return false, ""
}

// Draw dibuja el selector
// Draw dibuja el selector
func (ss *ScenarioSelector) Draw(screen *ebiten.Image) {
	// Dibujar botón principal
	btnColor := ss.colorBg
	if ss.isOpen {
		btnColor = ss.colorBgHover
	}

	vector.DrawFilledRect(screen, ss.x, ss.y, ss.width, ss.height, btnColor, false)
	vector.StrokeRect(screen, ss.x, ss.y, ss.width, ss.height, 2, ss.colorBorder, false)

	// Texto del botón principal
	selectedText := ss.options[ss.selectedIndex].Name
	arrow := "▼"
	if ss.isOpen {
		arrow = "▲"
	}
	text := selectedText + " " + arrow

	ebitenutil.DebugPrintAt(screen, text, int(ss.x+10), int(ss.y+10))

	// ← CAMBIO: Dibujar dropdown HACIA ARRIBA si está abierto
	if ss.isOpen {
		numOptions := len(ss.options)

		for i, opt := range ss.options {
			// ← NUEVO: Calcular Y hacia ARRIBA
			// i=0 (última opción) → justo arriba del botón
			// i=1 → una posición más arriba
			// i=2 → dos posiciones más arriba
			reverseIndex := numOptions - 1 - i
			optY := ss.y - float32(reverseIndex+1)*ss.height

			// Color según hover
			optColor := ss.colorDropdownBg
			if i == ss.hoveredIndex {
				optColor = ss.colorBgHover
			}

			// Fondo de la opción
			vector.DrawFilledRect(screen, ss.x, optY, ss.width, ss.height, optColor, false)
			vector.StrokeRect(screen, ss.x, optY, ss.width, ss.height, 1, ss.colorBorder, false)

			// Texto de la opción
			prefix := "  "
			if i == ss.selectedIndex {
				prefix = "✓ "
			}
			ebitenutil.DebugPrintAt(screen, prefix+opt.Name, int(ss.x+10), int(optY+10))
		}
	}
}

// GetSelectedID retorna el ID del escenario seleccionado
func (ss *ScenarioSelector) GetSelectedID() string {
	return ss.options[ss.selectedIndex].ID
}

// SetSelected establece la selección por ID
func (ss *ScenarioSelector) SetSelected(id string) {
	for i, opt := range ss.options {
		if opt.ID == id {
			ss.selectedIndex = i
			return
		}
	}
}
