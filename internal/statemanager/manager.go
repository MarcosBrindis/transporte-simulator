package statemanager

import (
	"fmt"
	"sync"
	"time"

	"github.com/MarcosBrindi/transporte-simulator/internal/config"
	"github.com/MarcosBrindi/transporte-simulator/internal/eventbus"
)

// StateManager gestiona el estado del vehículo
type StateManager struct {
	bus        *eventbus.EventBus
	calculator *VehicleStateCalculator
	doorState  *DoorStateManager

	// Channels de suscripción
	gpsEvents  chan eventbus.Event
	mpuEvents  chan eventbus.Event
	doorEvents chan eventbus.Event
	// Estado actual
	mu            sync.RWMutex
	latestGPS     eventbus.GPSData
	latestMPU     eventbus.MPUData
	latestDoor    eventbus.DoorData
	currentState  eventbus.VehicleStateData
	previousState string
	hasGPSData    bool
	hasMPUData    bool
	hasDoorData   bool

	// Control
	running bool
	paused  bool
}

// NewStateManager crea un nuevo State Manager
func NewStateManager(bus *eventbus.EventBus, cfg config.Config) *StateManager {
	return &StateManager{
		bus:         bus,
		calculator:  NewVehicleStateCalculator(cfg.Thresholds.MovementKmh),
		doorState:   NewDoorStateManager(cfg),
		gpsEvents:   make(chan eventbus.Event, 10),
		mpuEvents:   make(chan eventbus.Event, 10),
		doorEvents:  make(chan eventbus.Event, 10),
		running:     false,
		paused:      false,
		hasGPSData:  false,
		hasMPUData:  false,
		hasDoorData: false,
	}
}

// Start inicia el State Manager
func (sm *StateManager) Start() {
	sm.mu.Lock()
	sm.running = true
	sm.mu.Unlock()

	// Suscribirse a eventos
	gpsChannel := sm.bus.Subscribe(eventbus.EventGPS)
	mpuChannel := sm.bus.Subscribe(eventbus.EventMPU)
	doorChannel := sm.bus.Subscribe(eventbus.EventDoor)

	// Goroutines para reenviar eventos
	go func() {
		for event := range gpsChannel {
			if sm.isRunning() {
				select {
				case sm.gpsEvents <- event:
				default:
				}
			}
		}
	}()

	go func() {
		for event := range mpuChannel {
			if sm.isRunning() {
				select {
				case sm.mpuEvents <- event:
				default:
				}
			}
		}
	}()

	// Goroutine para eventos de puerta
	go func() {
		for event := range doorChannel {
			if sm.isRunning() {
				select {
				case sm.doorEvents <- event:
				default:
				}
			}
		}
	}()

	// Goroutine principal
	go sm.loop()

	fmt.Println("✅ [StateManager] Iniciado")
}

// Stop detiene el State Manager
func (sm *StateManager) Stop() {
	sm.mu.Lock()
	sm.running = false
	sm.mu.Unlock()

	fmt.Println("🛑 [StateManager] Detenido")
}

// Pause pausa el State Manager
func (sm *StateManager) Pause() {
	sm.mu.Lock()
	sm.paused = true
	sm.mu.Unlock()
}

// Resume reanuda el State Manager
func (sm *StateManager) Resume() {
	sm.mu.Lock()
	sm.paused = false
	sm.mu.Unlock()
}

// isRunning verifica si está corriendo (thread-safe)
func (sm *StateManager) isRunning() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.running
}

// isPaused verifica si está pausado (thread-safe)
func (sm *StateManager) isPaused() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.paused
}

// loop es el bucle principal del State Manager
func (sm *StateManager) loop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for sm.isRunning() {
		select {
		case gpsEvent := <-sm.gpsEvents:
			sm.handleGPS(gpsEvent)

		case mpuEvent := <-sm.mpuEvents:
			sm.handleMPU(mpuEvent)

		case doorEvent := <-sm.doorEvents:
			sm.handleDoor(doorEvent)

		case <-ticker.C:
			if !sm.isPaused() {
				sm.calculateAndPublishState()
			}
		}
	}
}

// handleGPS procesa eventos GPS
func (sm *StateManager) handleGPS(event eventbus.Event) {
	data := event.Data.(eventbus.GPSData)

	sm.mu.Lock()
	sm.latestGPS = data
	sm.hasGPSData = true
	sm.mu.Unlock()
}

// handleMPU procesa eventos MPU
func (sm *StateManager) handleMPU(event eventbus.Event) {
	data := event.Data.(eventbus.MPUData)

	sm.mu.Lock()
	sm.latestMPU = data
	sm.hasMPUData = true
	sm.mu.Unlock()
}

// handleDoor procesa eventos de puerta
func (sm *StateManager) handleDoor(event eventbus.Event) {
	data := event.Data.(eventbus.DoorData)

	sm.mu.Lock()
	sm.latestDoor = data
	sm.hasDoorData = true

	// Actualizar máquina de estados de puerta
	if sm.hasGPSData && sm.hasMPUData {
		sm.doorState.Update(data, sm.currentState)
	}

	sm.mu.Unlock()
}

// calculateAndPublishState calcula el estado y lo publica
func (sm *StateManager) calculateAndPublishState() {
	sm.mu.Lock()

	// Solo calcular si tenemos datos de GPS y MPU
	if !sm.hasGPSData || !sm.hasMPUData {
		sm.mu.Unlock()
		return
	}

	// Calcular estado
	state := sm.calculator.Calculate(sm.latestGPS, sm.latestMPU)

	// Agregar estado de puerta si disponible
	if sm.hasDoorData {
		state.DoorOpen = sm.latestDoor.IsOpen
	}

	// Detectar cambio de estado
	stateChanged := state.State != sm.previousState

	// Actualizar estado actual
	sm.currentState = state
	sm.previousState = state.State

	sm.mu.Unlock()

	// Publicar evento de estado
	sm.bus.Publish(eventbus.Event{
		Type:      eventbus.EventVehicle,
		Timestamp: time.Now(),
		Data:      state,
	})

	// Log solo cuando cambia el estado
	if stateChanged {
		doorStatus := "🔴"
		if state.DoorOpen {
			doorStatus = "🟢"
		}
		fmt.Printf("🚗 [StateManager] Estado: %s | Speed: %.1f km/h | Puerta: %s\n",
			state.State,
			state.Speed,
			doorStatus,
		)
	}
}

// GetCurrentState retorna el estado actual (thread-safe)
func (sm *StateManager) GetCurrentState() eventbus.VehicleStateData {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.currentState
}
