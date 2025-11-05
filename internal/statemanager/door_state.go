package statemanager

import (
	"fmt"
	"time"

	"github.com/MarcosBrindi/transporte-simulator/internal/config"
	"github.com/MarcosBrindi/transporte-simulator/internal/eventbus"
)

// DoorStateManager gestiona la máquina de estados de la puerta
type DoorStateManager struct {
	config config.Config

	// Estado actual
	currentState         eventbus.DoorState
	previousDoorOpen     bool
	doorMonitoringActive bool
	doorMonitoringStart  time.Time
	doorCloseStart       time.Time
	doorCloseConfirmed   bool
	initialPersonCount   int // Conteo inicial al abrir puerta (Fase 6)
}

// NewDoorStateManager crea un nuevo gestor de estado de puerta
func NewDoorStateManager(cfg config.Config) *DoorStateManager {
	return &DoorStateManager{
		config:               cfg,
		currentState:         eventbus.DoorIdle,
		previousDoorOpen:     false,
		doorMonitoringActive: false,
		doorCloseConfirmed:   false,
		initialPersonCount:   0,
	}
}

// Update actualiza la máquina de estados según datos de puerta y vehículo
func (dsm *DoorStateManager) Update(doorData eventbus.DoorData, vehicleState eventbus.VehicleStateData) {
	currentTime := time.Now()

	// Detectar cambio de estado de la puerta
	if doorData.IsOpen != dsm.previousDoorOpen {
		if doorData.IsOpen {
			// PUERTA SE ABRIÓ
			dsm.handleDoorOpened(doorData, vehicleState, currentTime)
		} else {
			// PUERTA SE CERRÓ
			dsm.handleDoorClosed(doorData, currentTime)
		}

		dsm.previousDoorOpen = doorData.IsOpen
	}

	// Verificar confirmación de cierre
	if dsm.doorMonitoringActive && !doorData.IsOpen && !dsm.doorCloseConfirmed {
		dsm.checkCloseConfirmation(currentTime)
	}

	// Verificar timeout de seguridad
	if dsm.doorMonitoringActive {
		dsm.checkMonitoringTimeout(currentTime)
	}
}

// handleDoorOpened maneja cuando la puerta se abre
func (dsm *DoorStateManager) handleDoorOpened(doorData eventbus.DoorData, vehicleState eventbus.VehicleStateData, currentTime time.Time) {
	// Solo iniciar monitoreo si el vehículo está detenido
	if vehicleState.IsStopped {
		dsm.doorMonitoringActive = true
		dsm.doorMonitoringStart = currentTime
		dsm.doorCloseConfirmed = false
		dsm.doorCloseStart = time.Time{} // Reset
		dsm.currentState = eventbus.DoorOpened

		fmt.Printf("🚪 [DoorState] PUERTA ABIERTA (distancia: %dmm)\n", doorData.DistanceMM)
		fmt.Printf("⏱️  Iniciando monitoreo (hasta cierre confirmado)\n")
		fmt.Printf("🔄 Estado: %s - %s\n", dsm.currentState, dsm.currentState.Description())
	} else {
		fmt.Printf("🚫 [DoorState] Puerta abierta pero vehículo en movimiento (%s) - ignorando\n", vehicleState.State)
	}
}

// handleDoorClosed maneja cuando la puerta se cierra
func (dsm *DoorStateManager) handleDoorClosed(doorData eventbus.DoorData, currentTime time.Time) {
	if dsm.doorMonitoringActive {
		dsm.doorCloseStart = currentTime
		dsm.currentState = eventbus.DoorClosing

		fmt.Printf("🚪 [DoorState] PUERTA CERRADA (distancia: %dmm)\n", doorData.DistanceMM)
		fmt.Printf("   Iniciando confirmación de cierre (%.0fs)\n", dsm.config.Timeouts.DoorCloseConfirm)
		fmt.Printf("   🔄 Estado: %s - %s\n", dsm.currentState, dsm.currentState.Description())
	}
}

// checkCloseConfirmation verifica si el cierre está confirmado
func (dsm *DoorStateManager) checkCloseConfirmation(currentTime time.Time) {
	if dsm.doorCloseStart.IsZero() {
		return
	}

	closeDuration := currentTime.Sub(dsm.doorCloseStart).Seconds()

	if closeDuration >= dsm.config.Timeouts.DoorCloseConfirm {
		// Cierre confirmado
		dsm.doorCloseConfirmed = true
		dsm.currentState = eventbus.DoorAnalyzingChanges

		fmt.Printf("[DoorState] Cierre CONFIRMADO después de %.1fs\n", closeDuration)
		fmt.Printf("   🔄 Estado: %s - %s\n", dsm.currentState, dsm.currentState.Description())

		// Finalizar monitoreo
		dsm.finalizeDoorMonitoring()
	}
}

// checkMonitoringTimeout verifica timeout de seguridad
func (dsm *DoorStateManager) checkMonitoringTimeout(currentTime time.Time) {
	monitoringDuration := currentTime.Sub(dsm.doorMonitoringStart).Seconds()

	if monitoringDuration >= dsm.config.Timeouts.MaxMonitoring {
		fmt.Printf("⏰ [DoorState] TIMEOUT DE SEGURIDAD - Monitoreo excedió %.0fs\n", dsm.config.Timeouts.MaxMonitoring)
		fmt.Printf("   ⚠️  Posible puerta bloqueada o persona en puerta por tiempo prolongado\n")
		fmt.Printf("   🔄 Finalizando monitoreo por seguridad\n")

		dsm.finalizeDoorMonitoring()
	}
}

// finalizeDoorMonitoring finaliza el monitoreo de puerta
func (dsm *DoorStateManager) finalizeDoorMonitoring() {
	monitoringDuration := time.Since(dsm.doorMonitoringStart).Seconds()

	fmt.Printf("🔍 [DoorState] FINALIZANDO MONITOREO DE PUERTA\n")
	fmt.Printf("   ⏱️  Duración total: %.1fs\n", monitoringDuration)
	fmt.Printf("   🔄 Estado: IDLE\n")

	// TODO (Fase 6): Aquí se procesarán cambios de pasajeros
	// Por ahora solo reseteamos el estado

	dsm.doorMonitoringActive = false
	dsm.doorCloseStart = time.Time{}
	dsm.doorCloseConfirmed = false
	dsm.currentState = eventbus.DoorIdle
}

// GetCurrentState retorna el estado actual
func (dsm *DoorStateManager) GetCurrentState() eventbus.DoorState {
	return dsm.currentState
}

// IsMonitoring retorna si está monitoreando
func (dsm *DoorStateManager) IsMonitoring() bool {
	return dsm.doorMonitoringActive
}
