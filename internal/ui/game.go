package ui

import (
	"fmt"
	"image/color"
	"sync"

	"github.com/MarcosBrindi/transporte-simulator/internal/config"
	"github.com/MarcosBrindi/transporte-simulator/internal/eventbus"
	"github.com/MarcosBrindi/transporte-simulator/internal/scenario"
	"github.com/hajimehoshi/ebiten/v2"
)

// Game es la estructura principal de Ebiten
type Game struct {
	bus    *eventbus.EventBus
	config *config.Config
	route  *scenario.Route

	// Componentes UI
	vehicleView *VehicleView
	controls    *Controls

	// Estado actual (thread-safe)
	mu           sync.RWMutex
	gpsData      eventbus.GPSData
	mpuData      eventbus.MPUData
	vehicleState eventbus.VehicleStateData
	progress     float64
	hasData      bool
	running      bool // ← Movido dentro de la protección del mutex

	// Channels de suscripción
	gpsEvents     chan eventbus.Event
	mpuEvents     chan eventbus.Event
	vehicleEvents chan eventbus.Event
}

// NewGame crea una nueva instancia del juego
func NewGame(bus *eventbus.EventBus, cfg *config.Config, route *scenario.Route) *Game {
	game := &Game{
		bus:           bus,
		config:        cfg,
		route:         route,
		gpsEvents:     make(chan eventbus.Event, 10),
		mpuEvents:     make(chan eventbus.Event, 10),
		vehicleEvents: make(chan eventbus.Event, 10),
		running:       true,
		hasData:       false,
	}

	// Crear componentes UI
	game.vehicleView = NewVehicleView(cfg, route)
	game.controls = NewControls()

	// Suscribirse a eventos
	game.subscribeToEvents()

	return game
}

// isRunning verifica si el juego está corriendo (thread-safe)
func (g *Game) isRunning() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.running
}

// subscribeToEvents suscribe a eventos del bus
func (g *Game) subscribeToEvents() {
	// GPS
	gpsChannel := g.bus.Subscribe(eventbus.EventGPS)
	go func() {
		for event := range gpsChannel {
			if g.isRunning() { // ← Usar método thread-safe
				select {
				case g.gpsEvents <- event:
				default:
				}
			}
		}
	}()

	// MPU
	mpuChannel := g.bus.Subscribe(eventbus.EventMPU)
	go func() {
		for event := range mpuChannel {
			if g.isRunning() { // ← Usar método thread-safe
				select {
				case g.mpuEvents <- event:
				default:
				}
			}
		}
	}()

	// Vehicle State
	vehicleChannel := g.bus.Subscribe(eventbus.EventVehicle)
	go func() {
		for event := range vehicleChannel {
			if g.isRunning() { // ← Usar método thread-safe
				select {
				case g.vehicleEvents <- event:
				default:
				}
			}
		}
	}()
}

// Update actualiza la lógica del juego (llamado por Ebiten a 60 FPS)
func (g *Game) Update() error {
	// Procesar eventos del Event Bus (non-blocking)
	select {
	case event := <-g.gpsEvents:
		g.handleGPSEvent(event)
	default:
	}

	select {
	case event := <-g.mpuEvents:
		g.handleMPUEvent(event)
	default:
	}

	select {
	case event := <-g.vehicleEvents:
		g.handleVehicleEvent(event)
	default:
	}

	// Actualizar componentes UI
	g.controls.Update()

	return nil
}

// Draw dibuja el juego (llamado por Ebiten a 60 FPS)
func (g *Game) Draw(screen *ebiten.Image) {
	// Fondo
	screen.Fill(color.RGBA{20, 20, 30, 255}) // Fondo oscuro

	g.mu.RLock()
	hasData := g.hasData
	gpsData := g.gpsData
	mpuData := g.mpuData
	vehicleState := g.vehicleState
	progress := g.progress // ← Leer progress (línea 148)
	g.mu.RUnlock()

	if !hasData {
		// Mostrar mensaje de "Esperando datos..."
		g.drawWaitingMessage(screen)
		return
	}

	// Dibujar vista del vehículo PASANDO progress
	g.vehicleView.Draw(screen, gpsData, mpuData, vehicleState, progress)

	// Dibujar controles (en la parte inferior)
	g.controls.Draw(screen)
}

// Layout define el tamaño de la ventana
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.config.UI.Window.Width, g.config.UI.Window.Height
}

// handleGPSEvent procesa eventos GPS
func (g *Game) handleGPSEvent(event eventbus.Event) {
	data := event.Data.(eventbus.GPSData)

	g.mu.Lock()
	g.gpsData = data
	g.progress = data.Progress
	g.hasData = true
	g.mu.Unlock()
}

// handleMPUEvent procesa eventos MPU
func (g *Game) handleMPUEvent(event eventbus.Event) {
	data := event.Data.(eventbus.MPUData)

	g.mu.Lock()
	g.mpuData = data
	g.mu.Unlock()
}

// handleVehicleEvent procesa eventos de estado del vehículo
func (g *Game) handleVehicleEvent(event eventbus.Event) {
	data := event.Data.(eventbus.VehicleStateData)

	g.mu.Lock()
	g.vehicleState = data
	g.mu.Unlock()
}

// drawWaitingMessage dibuja mensaje de espera
func (g *Game) drawWaitingMessage(screen *ebiten.Image) {
	// TODO: Dibujar texto "Esperando datos de sensores..."
	// Por ahora solo un rectángulo de placeholder
	_ = screen
}

// Stop detiene el juego
func (g *Game) Stop() {
	g.mu.Lock()
	g.running = false
	g.mu.Unlock()

	fmt.Println("🛑 [UI] Juego detenido")
}
