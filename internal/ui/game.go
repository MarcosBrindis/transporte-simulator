package ui

import (
	"fmt"
	"image/color"
	"sync"
	"time"

	"github.com/MarcosBrindi/transporte-simulator/internal/config"
	"github.com/MarcosBrindi/transporte-simulator/internal/eventbus"
	"github.com/MarcosBrindi/transporte-simulator/internal/scenario"
	"github.com/MarcosBrindi/transporte-simulator/internal/sensors"
	"github.com/MarcosBrindi/transporte-simulator/internal/statemanager"
	"github.com/hajimehoshi/ebiten/v2"
)

// Game es la estructura principal de Ebiten
type Game struct {
	bus      *eventbus.EventBus
	config   *config.Config
	route    *scenario.Route
	stateMgr *statemanager.StateManager
	executor *scenario.Executor

	// Referencias a sensores para reset
	gps     *sensors.GPSSimulator
	mpu     *sensors.MPU6050Simulator
	vl53l0x *sensors.VL53L0XSimulator
	camera  *sensors.CameraSimulator

	// Componentes UI
	vehicleView *VehicleView
	controls    *Controls
	eventLog    *EventLog

	// Estado actual (thread-safe)
	mu           sync.RWMutex
	gpsData      eventbus.GPSData
	mpuData      eventbus.MPUData
	vehicleState eventbus.VehicleStateData
	progress     float64
	hasData      bool

	// Control de ejecución
	running bool

	// Channels de suscripción
	gpsEvents       chan eventbus.Event
	mpuEvents       chan eventbus.Event
	vehicleEvents   chan eventbus.Event
	passengerEvents chan eventbus.Event
}

// NewGame crea una nueva instancia del juego
func NewGame(
	bus *eventbus.EventBus,
	cfg *config.Config,
	route *scenario.Route,
	stateMgr *statemanager.StateManager,
	executor *scenario.Executor,
	gps *sensors.GPSSimulator,
	mpu *sensors.MPU6050Simulator,
	vl53l0x *sensors.VL53L0XSimulator,
	camera *sensors.CameraSimulator,
) *Game {
	game := &Game{
		bus:             bus,
		config:          cfg,
		route:           route,
		stateMgr:        stateMgr,
		executor:        executor,
		gps:             gps,
		mpu:             mpu,
		vl53l0x:         vl53l0x,
		camera:          camera,
		gpsEvents:       make(chan eventbus.Event, 10),
		mpuEvents:       make(chan eventbus.Event, 10),
		vehicleEvents:   make(chan eventbus.Event, 10),
		passengerEvents: make(chan eventbus.Event, 10),
		running:         true,
		hasData:         false,
	}

	// Crear componentes UI
	game.vehicleView = NewVehicleView(cfg, route)
	game.controls = NewControls(cfg)
	game.eventLog = NewEventLog(15) // Mostrar últimos 15 eventos

	// Suscribirse a eventos
	game.subscribeToEvents()

	// Log inicial
	game.eventLog.Add("Sistema iniciado", "success")

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

	//Passenger events
	passengerChannel := g.bus.Subscribe(eventbus.EventPassenger)
	go func() {
		for event := range passengerChannel {
			if g.isRunning() {
				select {
				case g.passengerEvents <- event:
				default:
				}
			}
		}
	}()
}

// Update actualiza la lógica del juego (llamado por Ebiten a 60 FPS)
func (g *Game) Update() error {
	// Verificar si debe cerrar
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		return fmt.Errorf("salir")
	}
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

	select {
	case event := <-g.passengerEvents:
		g.handlePassengerEvent(event)
	default:
	}

	//Actualizar controles y procesar acciones
	action := g.controls.Update()
	if action != "" {
		g.handleControlAction(action)
	}
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
	progress := g.progress
	g.mu.RUnlock()

	if !hasData {
		// Mostrar mensaje de "Esperando datos..."
		g.drawWaitingMessage(screen)
		return
	}
	//Obtener estadísticas de pasajeros
	current, entries, exits := g.stateMgr.GetPassengerStats()
	// Dibujar vista del vehículo PASANDO progress
	g.vehicleView.Draw(screen, gpsData, mpuData, vehicleState, progress, current, entries, exits)

	// Dibujar controles (en la parte inferior)
	g.controls.Draw(screen)

	// Dibujar log de eventos
	/*logX := float32(450)                             // Más a la derecha
	logY := float32(g.config.UI.Window.Height - 250) // Desde abajo
	logWidth := float32(800)                         // Más ancho
	logHeight := float32(200)                        // Más bajo
	g.eventLog.Draw(screen, logX, logY, logWidth, logHeight)*/
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

// handlePassengerEvent procesa eventos de pasajeros
func (g *Game) handlePassengerEvent(event eventbus.Event) {
	data := event.Data.(eventbus.PassengerEventData)

	// Log en consola
	if data.EventType == "ENTRY" {
		g.eventLog.Add("✅ Pasajero subió", "success")
	} else if data.EventType == "EXIT" {
		g.eventLog.Add("🚪 Pasajero bajó", "info")
	}
}

// handleControlAction maneja las acciones de los controles
func (g *Game) handleControlAction(action string) {
	switch action {
	case "play":
		fmt.Println("▶️  [UI] PLAY - Reanudando simulación")
		g.stateMgr.Resume()
		if g.executor != nil {
			g.executor.Resume()
		}
		g.controls.SetSystemState(StateRunning)
		// g.eventLog.Add("▶️ Simulación reanudada", "success")

	case "pause":
		fmt.Println("⏸️  [UI] PAUSE - Pausando simulación")
		g.stateMgr.Pause()
		if g.executor != nil {
			g.executor.Pause()
		}
		g.controls.SetSystemState(StatePaused)
		// g.eventLog.Add("⏸️ Simulación pausada", "warning")

	case "reset":
		fmt.Println("🔄 [UI] RESET - Reiniciando simulación")
		g.controls.SetSystemState(StateLoading)
		// g.eventLog.Add("🔄 Reiniciando...", "info")
		g.resetSimulation()

	case "speed_1x":
		fmt.Println("🏃 [UI] Velocidad: 1x")
		g.applySpeedMultiplier(1.0)
		// g.eventLog.Add("🏃 Velocidad: 1x", "info")

	case "speed_2x":
		fmt.Println("🏃‍♂️ [UI] Velocidad: 2x")
		g.applySpeedMultiplier(2.0)
		// g.eventLog.Add("🏃‍♂️ Velocidad: 2x", "info")

	case "speed_3x":
		fmt.Println("🏃‍♀️💨 [UI] Velocidad: 3x")
		g.applySpeedMultiplier(3.0)
		// g.eventLog.Add("🏃‍♀️ Velocidad: 3x", "info")
	}
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

// resetSimulation reinicia toda la simulación
func (g *Game) resetSimulation() {
	fmt.Println("🔄 [UI] Reiniciando simulación completa...")

	// 1. Detener executor actual
	if g.executor != nil {
		g.executor.Stop()
	}

	// 2. Pausar sensores
	g.gps.Pause()
	g.mpu.Pause()
	g.vl53l0x.Pause()
	g.camera.Pause()

	// 3. Resetear GPS a posición inicial
	g.gps.Reset()

	// 4. Resetear StateManager
	g.stateMgr.Reset()

	// 5. Limpiar datos locales
	g.mu.Lock()
	g.hasData = false
	g.progress = 0
	g.mu.Unlock()

	// 6. Esperar un momento para que se estabilice
	time.Sleep(100 * time.Millisecond)

	// 7. Reanudar sensores
	g.gps.Resume()
	g.mpu.Resume()
	g.vl53l0x.Resume()
	g.camera.Resume()

	// 8. Reiniciar executor con escenario
	scenarioName := g.controls.GetSelectedScenario()
	newScenario := g.loadScenario(scenarioName)
	g.executor = scenario.NewExecutor(newScenario, g.gps, g.bus)
	g.executor.Start()

	// 9. Cambiar estado a running
	g.controls.SetSystemState(StateRunning)

	fmt.Println("✅ [UI] Simulación reiniciada exitosamente")
	// g.eventLog.Add("✅ Simulación reiniciada", "success")
}

// loadScenario carga un escenario por nombre
func (g *Game) loadScenario(name string) *scenario.Scenario {
	switch name {
	case "parada_normal":
		return scenario.GetParadaNormal()
	case "parada_con_salidas":
		return scenario.GetParadaConSalidas()
	case "circuito_completo":
		return scenario.GetCircuitoCompleto()
	default:
		return scenario.GetParadaNormal()
	}
}

// applySpeedMultiplier aplica el multiplicador de velocidad a los sensores
func (g *Game) applySpeedMultiplier(multiplier float64) {
	fmt.Printf("⚡ [UI] Aplicando multiplicador de velocidad: %.1fx\n", multiplier)

	// Calcular nuevas frecuencias
	baseGPSFreq := g.config.Sensors.GPS.Frequency
	baseMPUFreq := g.config.Sensors.MPU6050.Frequency
	baseVL53Freq := g.config.Sensors.VL53L0X.Frequency
	baseCameraFreq := g.config.Sensors.Camera.Frequency

	newGPSFreq := baseGPSFreq * multiplier
	newMPUFreq := baseMPUFreq * multiplier
	newVL53Freq := baseVL53Freq * multiplier
	newCameraFreq := baseCameraFreq * multiplier

	// Aplicar nuevas frecuencias
	g.gps.SetFrequency(newGPSFreq)
	g.mpu.SetFrequency(newMPUFreq)
	g.vl53l0x.SetFrequency(newVL53Freq)
	g.camera.SetFrequency(newCameraFreq)

	fmt.Printf("   GPS: %.1f Hz → %.1f Hz\n", baseGPSFreq, newGPSFreq)
	fmt.Printf("   MPU: %.1f Hz → %.1f Hz\n", baseMPUFreq, newMPUFreq)
	fmt.Printf("   VL53L0X: %.1f Hz → %.1f Hz\n", baseVL53Freq, newVL53Freq)
	fmt.Printf("   Camera: %.1f Hz → %.1f Hz\n", baseCameraFreq, newCameraFreq)
}
