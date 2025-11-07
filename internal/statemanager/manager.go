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
	bus              *eventbus.EventBus
	cfg              config.Config
	calculator       *VehicleStateCalculator
	doorState        *DoorStateManager
	passengerTracker *PassengerTracker

	// Channels de suscripción
	gpsEvents    chan eventbus.Event
	mpuEvents    chan eventbus.Event
	doorEvents   chan eventbus.Event
	cameraEvents chan eventbus.Event

	// Estado actual
	mu            sync.RWMutex
	latestGPS     eventbus.GPSData
	latestMPU     eventbus.MPUData
	latestDoor    eventbus.DoorData
	latestCamera  eventbus.CameraData
	currentState  eventbus.VehicleStateData
	previousState string
	hasGPSData    bool
	hasMPUData    bool
	hasDoorData   bool
	hasCameraData bool

	// Control
	running bool
	paused  bool

	// Estado de puerta para tracking de pasajeros
	previousDoorOpen bool
}

// NewStateManager crea un nuevo State Manager
func NewStateManager(bus *eventbus.EventBus, cfg config.Config) *StateManager {
	return &StateManager{
		bus:              bus,
		cfg:              cfg,
		calculator:       NewVehicleStateCalculator(cfg.Thresholds.MovementKmh),
		doorState:        NewDoorStateManager(cfg),
		passengerTracker: NewPassengerTracker(bus, cfg),
		gpsEvents:        make(chan eventbus.Event, 10),
		mpuEvents:        make(chan eventbus.Event, 10),
		doorEvents:       make(chan eventbus.Event, 10),
		cameraEvents:     make(chan eventbus.Event, 10),
		running:          false,
		paused:           false,
		hasGPSData:       false,
		hasMPUData:       false,
		hasDoorData:      false,
		hasCameraData:    false,
		previousDoorOpen: false,
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
	cameraChannel := sm.bus.Subscribe(eventbus.EventCamera)

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

	//  Goroutine para eventos de cámara
	go func() {
		for event := range cameraChannel {
			if sm.isRunning() {
				select {
				case sm.cameraEvents <- event:
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

		case cameraEvent := <-sm.cameraEvents:
			sm.handleCamera(cameraEvent)

		case <-ticker.C:
			if !sm.isPaused() {
				sm.calculateAndPublishState()
				sm.checkPassengerConfirmations()
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
	previousDoorOpen := sm.previousDoorOpen
	sm.latestDoor = data
	sm.hasDoorData = true
	sm.previousDoorOpen = data.IsOpen

	// Actualizar máquina de estados de puerta
	if sm.hasGPSData && sm.hasMPUData {
		sm.doorState.Update(data, sm.currentState)
	}

	sm.mu.Unlock()

	// Notificar a PassengerTracker sobre cambios de puerta
	if !previousDoorOpen && data.IsOpen {
		// Puerta se abrió
		sm.passengerTracker.OnDoorOpened()
	} else if previousDoorOpen && !data.IsOpen {
		// Puerta EMPIEZA a cerrarse (antes de confirmación)
		sm.passengerTracker.OnDoorClosing() // ← NUEVO
	}
}

// handleCamera procesa eventos de cámara
func (sm *StateManager) handleCamera(event eventbus.Event) {
	data := event.Data.(eventbus.CameraData)

	sm.mu.Lock()
	sm.latestCamera = data
	sm.hasCameraData = true
	sm.mu.Unlock()

	// Procesar datos de cámara en PassengerTracker
	sm.passengerTracker.ProcessCameraData(data)
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

	// Verificar transiciones de estado de puerta
	sm.checkDoorStateTransitions()
}

// checkDoorStateTransitions verifica cambios en el estado de la puerta
func (sm *StateManager) checkDoorStateTransitions() {
	doorState := sm.doorState.GetCurrentState()

	// Cuando la puerta confirma el cierre (IDLE después de monitoreo)
	if doorState == eventbus.DoorIdle && sm.doorState.wasMonitoring {
		sm.passengerTracker.OnDoorClosed()
		sm.doorState.wasMonitoring = false
	}

	// Actualizar flag de monitoreo
	if sm.doorState.IsMonitoring() {
		sm.doorState.wasMonitoring = true
	}
}

// checkPassengerConfirmations verifica confirmaciones pendientes de pasajeros
func (sm *StateManager) checkPassengerConfirmations() {
	sm.mu.RLock()
	isStopped := sm.currentState.IsStopped
	sm.mu.RUnlock()

	sm.passengerTracker.CheckPendingConfirmations(time.Now(), isStopped)
}

// GetCurrentState retorna el estado actual (thread-safe)
func (sm *StateManager) GetCurrentState() eventbus.VehicleStateData {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.currentState
}

// GetPassengerStats retorna estadísticas de pasajeros
func (sm *StateManager) GetPassengerStats() (current, entries, exits int) {
	return sm.passengerTracker.GetStats()
}

// Reset reinicia el state manager
func (sm *StateManager) Reset() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Reset flags de datos
	sm.hasGPSData = false
	sm.hasMPUData = false
	sm.hasDoorData = false
	sm.hasCameraData = false
	sm.previousDoorOpen = false
	sm.previousState = ""

	// Reset currentState a valores por defecto (struct vacío)
	sm.currentState = eventbus.VehicleStateData{
		State:     "",
		IsMoving:  false,
		IsStopped: false,
		DoorOpen:  false,
		HasGPSFix: false,
		Speed:     0.0,
	}

	// Reset datos de sensores
	sm.latestGPS = eventbus.GPSData{}
	sm.latestMPU = eventbus.MPUData{}
	sm.latestDoor = eventbus.DoorData{}
	sm.latestCamera = eventbus.CameraData{}

	// Recrear DoorStateManager
	sm.doorState = NewDoorStateManager(sm.cfg)

	// Recrear PassengerTracker
	sm.passengerTracker = NewPassengerTracker(sm.bus, sm.cfg)

	fmt.Println("🔄 [StateManager] Reset completado")
}
