package ui

import (
	"image/color"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// LogEvent representa un evento en el log
type LogEvent struct {
	Timestamp time.Time
	Message   string
	Type      string // "info", "success", "warning", "error"
}

// EventLog gestiona el log de eventos en pantalla
type EventLog struct {
	mu        sync.RWMutex
	events    []LogEvent
	maxEvents int

	// Colores
	colorBg      color.RGBA
	colorBorder  color.RGBA
	colorInfo    color.RGBA
	colorSuccess color.RGBA
	colorWarning color.RGBA
	colorError   color.RGBA
}

// NewEventLog crea un nuevo log de eventos
func NewEventLog(maxEvents int) *EventLog {
	return &EventLog{
		events:       make([]LogEvent, 0, maxEvents),
		maxEvents:    maxEvents,
		colorBg:      color.RGBA{30, 30, 40, 255},
		colorBorder:  color.RGBA{80, 80, 100, 255},
		colorInfo:    color.RGBA{200, 200, 220, 255},
		colorSuccess: color.RGBA{100, 255, 100, 255},
		colorWarning: color.RGBA{255, 200, 100, 255},
		colorError:   color.RGBA{255, 100, 100, 255},
	}
}

// Add agrega un evento al log
func (el *EventLog) Add(message string, eventType string) {
	el.mu.Lock()
	defer el.mu.Unlock()

	event := LogEvent{
		Timestamp: time.Now(),
		Message:   message,
		Type:      eventType,
	}

	// Agregar al inicio
	el.events = append([]LogEvent{event}, el.events...)

	// Mantener solo los últimos maxEvents
	if len(el.events) > el.maxEvents {
		el.events = el.events[:el.maxEvents]
	}
}

// Draw dibuja el log en pantalla
func (el *EventLog) Draw(screen *ebiten.Image, x, y, width, height float32) {
	// Fondo del panel
	vector.DrawFilledRect(screen, x, y, width, height, el.colorBg, false)
	vector.StrokeRect(screen, x, y, width, height, 2, el.colorBorder, false)

	// Título
	ebitenutil.DebugPrintAt(screen, "📋 LOG DE EVENTOS", int(x+10), int(y+10))

	// Dibujar eventos
	el.mu.RLock()
	defer el.mu.RUnlock()

	yOffset := int(y + 35)
	lineHeight := 18

	for i, event := range el.events {
		if i >= 10 { // Mostrar solo 10
			break
		}

		// Formatear timestamp
		timestamp := event.Timestamp.Format("15:04:05")

		// Dibujar línea
		line := timestamp + " " + event.Message
		if len(line) > 45 {
			line = line[:42] + "..."
		}

		ebitenutil.DebugPrintAt(screen, line, int(x+10), yOffset)

		yOffset += lineHeight
	}

	// Si no hay eventos
	if len(el.events) == 0 {
		ebitenutil.DebugPrintAt(screen, "Sin eventos recientes", int(x+10), int(y+35))
	}
}

// Clear limpia todos los eventos
func (el *EventLog) Clear() {
	el.mu.Lock()
	defer el.mu.Unlock()
	el.events = make([]LogEvent, 0, el.maxEvents)
}
