package ui

import (
	"image/color"

	"github.com/MarcosBrindi/transporte-simulator/internal/config"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// SystemState representa el estado del sistema
type SystemState string

const (
	StateRunning SystemState = "RUNNING"
	StatePaused  SystemState = "PAUSED"
	StateStopped SystemState = "STOPPED"
	StateLoading SystemState = "LOADING"
)

// Button representa un botón clickeable
type Button struct {
	X       float32
	Y       float32
	Width   float32
	Height  float32
	Label   string
	Action  string // "play", "pause", "reset", "speed"
	Enabled bool
	Hovered bool
}

// Controls gestiona los controles de la UI
type Controls struct {
	config  *config.Config
	buttons []Button

	// Estado
	isPaused         bool
	speedMultiplier  int // 1x, 2x, 3x
	selectedScenario string

	// Colores
	colorButton       color.RGBA
	colorButtonHover  color.RGBA
	colorButtonActive color.RGBA
	colorText         color.RGBA

	systemState SystemState
}

// NewControls crea nuevos controles
func NewControls(cfg *config.Config) *Controls {
	controls := &Controls{
		config:            cfg,
		isPaused:          false,
		speedMultiplier:   1,
		selectedScenario:  "parada_normal",
		colorButton:       color.RGBA{60, 60, 80, 255},
		colorButtonHover:  color.RGBA{80, 80, 100, 255},
		colorButtonActive: color.RGBA{100, 200, 100, 255},
		colorText:         color.RGBA{255, 255, 255, 255},
		systemState:       StateRunning,
	}

	// Crear botones
	controls.createButtons()

	return controls
}

// createButtons crea los botones de control
func (c *Controls) createButtons() {
	y := float32(c.config.UI.Window.Height - 50)
	x := float32(20)
	buttonWidth := float32(100)
	buttonHeight := float32(35)
	spacing := float32(10)

	c.buttons = []Button{
		{
			X:       x,
			Y:       y,
			Width:   buttonWidth,
			Height:  buttonHeight,
			Label:   "PLAY",
			Action:  "play",
			Enabled: true,
		},
		{
			X:       x + buttonWidth + spacing,
			Y:       y,
			Width:   buttonWidth,
			Height:  buttonHeight,
			Label:   "PAUSE",
			Action:  "pause",
			Enabled: true,
		},
		{
			X:       x + (buttonWidth+spacing)*2,
			Y:       y,
			Width:   buttonWidth,
			Height:  buttonHeight,
			Label:   "1x",
			Action:  "speed",
			Enabled: true,
		},
		{
			X:       x + (buttonWidth+spacing)*3,
			Y:       y,
			Width:   buttonWidth,
			Height:  buttonHeight,
			Label:   "RESET",
			Action:  "reset",
			Enabled: true,
		},
	}
}

// método para cambiar estado
func (c *Controls) SetSystemState(state SystemState) {
	c.systemState = state
}

// Método para dibujar estado del sistema
func (c *Controls) drawSystemState(screen *ebiten.Image) {
	x := float32(c.config.UI.Window.Width - 200)
	y := float32(20)

	// Determinar color e icono según estado
	var stateColor color.RGBA
	var icon string

	switch c.systemState {
	case StateRunning:
		stateColor = color.RGBA{100, 255, 100, 255}
		icon = "🟢"
	case StatePaused:
		stateColor = color.RGBA{255, 200, 100, 255}
		icon = "🟡"
	case StateStopped:
		stateColor = color.RGBA{255, 100, 100, 255}
		icon = "🔴"
	case StateLoading:
		stateColor = color.RGBA{100, 150, 255, 255}
		icon = "🔵"
	}

	// Fondo
	vector.DrawFilledRect(screen, x, y, 180, 35, c.colorButton, false)
	vector.StrokeRect(screen, x, y, 180, 35, 2, stateColor, false)

	// Texto
	text := icon + " " + string(c.systemState)
	ebitenutil.DebugPrintAt(screen, text, int(x+10), int(y+10))
}

// Update actualiza el estado de los controles
func (c *Controls) Update() (action string) {
	// Obtener posición del mouse
	mouseX, mouseY := ebiten.CursorPosition()

	// Verificar hover en botones
	for i := range c.buttons {
		c.buttons[i].Hovered = c.isMouseOver(&c.buttons[i], mouseX, mouseY)
	}

	// Verificar clicks
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		for i := range c.buttons {
			if c.buttons[i].Hovered && c.buttons[i].Enabled {
				return c.handleButtonClick(&c.buttons[i])
			}
		}
	}

	// Atajos de teclado
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		if c.isPaused {
			c.isPaused = false
			return "play"
		} else {
			c.isPaused = true
			return "pause"
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		return "reset"
	}

	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		c.speedMultiplier = 1
		c.updateSpeedButton()
		return "speed_1x"
	}

	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		c.speedMultiplier = 2
		c.updateSpeedButton()
		return "speed_2x"
	}

	if inpututil.IsKeyJustPressed(ebiten.Key3) {
		c.speedMultiplier = 3
		c.updateSpeedButton()
		return "speed_3x"
	}

	return ""
}

// handleButtonClick maneja el click en un botón
func (c *Controls) handleButtonClick(btn *Button) string {
	switch btn.Action {
	case "play":
		c.isPaused = false
		return "play"

	case "pause":
		c.isPaused = true
		return "pause"

	case "reset":
		c.isPaused = false
		c.speedMultiplier = 1
		c.updateSpeedButton()
		return "reset"

	case "speed":
		// Ciclar velocidades: 1x → 2x → 3x → 1x
		c.speedMultiplier++
		if c.speedMultiplier > 3 {
			c.speedMultiplier = 1
		}
		c.updateSpeedButton()
		return "speed_" + btn.Label
	}

	return ""
}

// updateSpeedButton actualiza el label del botón de velocidad
func (c *Controls) updateSpeedButton() {
	for i := range c.buttons {
		if c.buttons[i].Action == "speed" {
			c.buttons[i].Label = c.getSpeedLabel()
		}
	}
}

// getSpeedLabel retorna el label de velocidad según el multiplicador
func (c *Controls) getSpeedLabel() string {
	switch c.speedMultiplier {
	case 1:
		return "1x"
	case 2:
		return "2x"
	case 3:
		return "3x"
	default:
		return "1x"
	}
}

// isMouseOver verifica si el mouse está sobre un botón
func (c *Controls) isMouseOver(btn *Button, mouseX, mouseY int) bool {
	mx := float32(mouseX)
	my := float32(mouseY)
	return mx >= btn.X && mx <= btn.X+btn.Width &&
		my >= btn.Y && my <= btn.Y+btn.Height
}

// Draw dibuja los controles
func (c *Controls) Draw(screen *ebiten.Image) {
	// Dibujar cada botón
	for i := range c.buttons {
		c.drawButton(screen, &c.buttons[i])
	}

	// Dibujar información de escenario
	c.drawScenarioInfo(screen)

	// Dibujar atajos de teclado
	c.drawKeyboardShortcuts(screen)

	// Dibujar estado del sistema
	c.drawSystemState(screen)
}

// drawButton dibuja un botón individual
func (c *Controls) drawButton(screen *ebiten.Image, btn *Button) {
	// Determinar color según estado
	btnColor := c.colorButton
	if btn.Hovered {
		btnColor = c.colorButtonHover
	}

	// Si es el botón de pause y está pausado, resaltar
	if btn.Action == "pause" && c.isPaused {
		btnColor = c.colorButtonActive
	}

	// Si es el botón de play y NO está pausado, resaltar
	if btn.Action == "play" && !c.isPaused {
		btnColor = c.colorButtonActive
	}

	// Dibujar fondo del botón
	vector.DrawFilledRect(screen, btn.X, btn.Y, btn.Width, btn.Height, btnColor, false)

	// Dibujar borde
	borderColor := color.RGBA{100, 100, 120, 255}
	if btn.Hovered {
		borderColor = color.RGBA{150, 150, 180, 255}
	}
	vector.StrokeRect(screen, btn.X, btn.Y, btn.Width, btn.Height, 2, borderColor, false)

	// Dibujar texto centrado
	textX := int(btn.X + btn.Width/2 - float32(len(btn.Label)*6)/2)
	textY := int(btn.Y + btn.Height/2 - 4)
	ebitenutil.DebugPrintAt(screen, btn.Label, textX, textY)
}

// drawScenarioInfo dibuja información del escenario actual
func (c *Controls) drawScenarioInfo(screen *ebiten.Image) {
	x := float32(c.config.UI.Window.Width - 300)
	y := float32(c.config.UI.Window.Height - 50)

	// Fondo
	vector.DrawFilledRect(screen, x, y, 280, 35, c.colorButton, false)
	vector.StrokeRect(screen, x, y, 280, 35, 2, color.RGBA{100, 100, 120, 255}, false)

	// Texto
	scenarioName := c.getScenarioDisplayName()
	ebitenutil.DebugPrintAt(screen, "Escenario: "+scenarioName, int(x+10), int(y+10))
}

// drawKeyboardShortcuts dibuja ayuda de atajos
func (c *Controls) drawKeyboardShortcuts(screen *ebiten.Image) {
	y := c.config.UI.Window.Height - 25
	shortcuts := "[SPACE] Play/Pause  [R] Reset  [1/2/3] Velocidad  [ESC] Salir"
	ebitenutil.DebugPrintAt(screen, shortcuts, 20, y)
}

// getScenarioDisplayName retorna el nombre legible del escenario
func (c *Controls) getScenarioDisplayName() string {
	switch c.selectedScenario {
	case "parada_normal":
		return "Parada Normal"
	case "parada_con_salidas":
		return "Parada con Salidas"
	case "circuito_completo":
		return "Circuito Completo"
	default:
		return "Custom"
	}
}

// IsPaused retorna si está pausado
func (c *Controls) IsPaused() bool {
	return c.isPaused
}

// GetSpeedMultiplier retorna el multiplicador de velocidad
func (c *Controls) GetSpeedMultiplier() int {
	return c.speedMultiplier
}

// SetScenario cambia el escenario seleccionado
func (c *Controls) SetScenario(scenario string) {
	c.selectedScenario = scenario
}

// GetSelectedScenario retorna el escenario seleccionado
func (c *Controls) GetSelectedScenario() string {
	return c.selectedScenario
}
