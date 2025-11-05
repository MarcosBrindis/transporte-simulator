package statemanager

import (
	"fmt"
	"time"

	"github.com/MarcosBrindi/transporte-simulator/internal/config"
	"github.com/MarcosBrindi/transporte-simulator/internal/eventbus"
)

// PassengerTracker gestiona el tracking de pasajeros
type PassengerTracker struct {
	config config.Config
	bus    *eventbus.EventBus

	// Contadores
	passengerCountCurrent int // Pasajeros a bordo actualmente
	dailyEntries          int // Total de entradas del día
	dailyExits            int // Total de salidas del día

	// Estado de tracking
	trackHistory       map[int]*TrackInfo // Historial de tracks
	pendingEntries     map[int]*PendingEntry
	pendingExits       map[int]*PendingExit
	initialPersonCount int       // Conteo al abrir puerta
	doorOpenTime       time.Time // Cuando se abrió la puerta por última vez
}

// TrackInfo mantiene información de un track
type TrackInfo struct {
	TrackID         int
	FirstSeen       time.Time
	LastSeen        time.Time
	Counted         bool
	IsOnboard       bool
	CorrelationTime float64 // Tiempo de correlación con sensor
}

// PendingEntry entrada pendiente de confirmación
type PendingEntry struct {
	TrackID        int
	Timestamp      time.Time
	Confidence     float64
	SensorDistance *int
}

// PendingExit salida pendiente de confirmación
type PendingExit struct {
	TrackID        int
	Timestamp      time.Time
	Confidence     float64
	SensorDistance *int
	FramesMissing  int
}

// NewPassengerTracker crea un nuevo tracker de pasajeros
func NewPassengerTracker(bus *eventbus.EventBus, cfg config.Config) *PassengerTracker {
	return &PassengerTracker{
		config:                cfg,
		bus:                   bus,
		passengerCountCurrent: 0,
		dailyEntries:          0,
		dailyExits:            0,
		trackHistory:          make(map[int]*TrackInfo),
		pendingEntries:        make(map[int]*PendingEntry),
		pendingExits:          make(map[int]*PendingExit),
		initialPersonCount:    0,
	}
}

// OnDoorOpened maneja cuando la puerta se abre
func (pt *PassengerTracker) OnDoorOpened() {
	pt.initialPersonCount = pt.GetCurrentDetectedCount()
	pt.doorOpenTime = time.Now()

	fmt.Printf("👥 [Passengers] Conteo inicial al abrir puerta: %d personas\n", pt.initialPersonCount)
}

// OnDoorClosed maneja cuando la puerta se cierra (confirmado)
func (pt *PassengerTracker) OnDoorClosed() {
	currentTime := time.Now()
	currentCount := pt.GetCurrentDetectedCount()
	passengerDelta := currentCount - pt.initialPersonCount
	monitoringDuration := currentTime.Sub(pt.doorOpenTime).Seconds()

	fmt.Printf("🔍 [Passengers] FINALIZANDO MONITOREO DE PASAJEROS\n")
	fmt.Printf("   Conteo inicial: %d\n", pt.initialPersonCount)
	fmt.Printf("   Conteo final: %d\n", currentCount)
	fmt.Printf("   Delta: %+d\n", passengerDelta)
	fmt.Printf("   A bordo: %d\n", pt.passengerCountCurrent)
	fmt.Printf("   Duración: %.1fs\n", monitoringDuration)

	// Caso especial: Salidas detectadas (sistema cree que hay personas pero YOLO no ve ninguna)
	if pt.passengerCountCurrent > 0 && currentCount == 0 {
		estimatedExits := pt.passengerCountCurrent

		fmt.Printf("🔍 [Passengers] DETECCIÓN ESPECIAL DE SALIDA:\n")
		fmt.Printf("   Sistema creía: %d personas a bordo\n", pt.passengerCountCurrent)
		fmt.Printf("   👁️YOLO detecta: %d personas\n", currentCount)
		fmt.Printf("   Salidas estimadas: %d\n", estimatedExits)

		pt.processBulkExits(estimatedExits)
		return
	}

	// Procesar cambios normales
	if passengerDelta > 0 {
		// Entradas
		fmt.Printf("   🟢 Detectadas %d entradas\n", passengerDelta)
		pt.processBulkEntries(passengerDelta)
	} else if passengerDelta < 0 {
		// Salidas
		exits := -passengerDelta
		fmt.Printf("   🔴 Detectadas %d salidas\n", exits)
		pt.processBulkExits(exits)
	} else {
		// Sin cambios
		if pt.passengerCountCurrent != currentCount {
			fmt.Printf("   ⚠️  INCONSISTENCIA: Sistema=%d, YOLO=%d\n",
				pt.passengerCountCurrent, currentCount)
		}
		fmt.Printf("   Sin cambios en pasajeros\n")
	}
}

// ProcessCameraData procesa datos de la cámara
func (pt *PassengerTracker) ProcessCameraData(data eventbus.CameraData) {
	// Actualizar historial de tracks
	currentTime := time.Now()

	for _, track := range data.Tracks {
		if _, exists := pt.trackHistory[track.TrackID]; !exists {
			// Nuevo track
			pt.trackHistory[track.TrackID] = &TrackInfo{
				TrackID:   track.TrackID,
				FirstSeen: currentTime,
				LastSeen:  currentTime,
				Counted:   false,
				IsOnboard: false,
			}
		} else {
			// Track existente
			pt.trackHistory[track.TrackID].LastSeen = currentTime
		}
	}

	// Limpiar tracks antiguos (no vistos en más de 10 segundos)
	pt.cleanupOldTracks(currentTime)
}

// CheckPendingConfirmations verifica entradas/salidas pendientes
func (pt *PassengerTracker) CheckPendingConfirmations(currentTime time.Time, isStopped bool) {
	// Solo confirmar si el vehículo está detenido
	if !isStopped {
		return
	}

	// Verificar entradas pendientes
	pt.checkPendingEntries(currentTime)

	// Verificar salidas pendientes
	pt.checkPendingExits(currentTime)
}

// checkPendingEntries verifica entradas pendientes
func (pt *PassengerTracker) checkPendingEntries(currentTime time.Time) {
	entriesToConfirm := []int{}

	for trackID, entry := range pt.pendingEntries {
		timePending := currentTime.Sub(entry.Timestamp).Seconds()

		if timePending >= pt.config.Timeouts.EntryMin {
			// Confirmar entrada
			pt.confirmEntry(trackID, entry)
			entriesToConfirm = append(entriesToConfirm, trackID)
		} else if timePending >= pt.config.Timeouts.EntryMax {
			// Timeout - cancelar
			fmt.Printf("⏰ [Passengers] ENTRADA CANCELADA por timeout - Track ID: %d\n", trackID)
			entriesToConfirm = append(entriesToConfirm, trackID)
		}
	}

	// Limpiar confirmadas/canceladas
	for _, trackID := range entriesToConfirm {
		delete(pt.pendingEntries, trackID)
	}
}

// checkPendingExits verifica salidas pendientes
func (pt *PassengerTracker) checkPendingExits(currentTime time.Time) {
	exitsToConfirm := []int{}

	for trackID, exit := range pt.pendingExits {
		timePending := currentTime.Sub(exit.Timestamp).Seconds()

		if timePending >= pt.config.Timeouts.ExitConfirmation {
			// Confirmar salida
			pt.confirmExit(trackID, exit)
			exitsToConfirm = append(exitsToConfirm, trackID)
		}
	}

	// Limpiar confirmadas
	for _, trackID := range exitsToConfirm {
		delete(pt.pendingExits, trackID)
		delete(pt.trackHistory, trackID)
	}
}

// confirmEntry confirma una entrada
func (pt *PassengerTracker) confirmEntry(trackID int, entry *PendingEntry) {
	event := pt.createPassengerEvent(trackID, "ENTRY", entry.Confidence, entry.SensorDistance)
	pt.bus.Publish(eventbus.Event{
		Type:      eventbus.EventPassenger,
		Timestamp: time.Now(),
		Data:      event,
	})

	pt.passengerCountCurrent++
	pt.dailyEntries++

	if track, exists := pt.trackHistory[trackID]; exists {
		track.Counted = true
		track.IsOnboard = true
	}

	fmt.Printf("[Passengers] ENTRADA CONFIRMADA - Track ID: %d\n", trackID)
	fmt.Printf("   A bordo: %d\n", pt.passengerCountCurrent)
}

// confirmExit confirma una salida
func (pt *PassengerTracker) confirmExit(trackID int, exit *PendingExit) {
	event := pt.createPassengerEvent(trackID, "EXIT", exit.Confidence, exit.SensorDistance)
	pt.bus.Publish(eventbus.Event{
		Type:      eventbus.EventPassenger,
		Timestamp: time.Now(),
		Data:      event,
	})

	pt.passengerCountCurrent--
	if pt.passengerCountCurrent < 0 {
		pt.passengerCountCurrent = 0
	}
	pt.dailyExits++

	fmt.Printf("[Passengers] SALIDA CONFIRMADA - Track ID: %d\n", trackID)
	fmt.Printf("   A bordo: %d\n", pt.passengerCountCurrent)
}

// processBulkEntries procesa múltiples entradas
func (pt *PassengerTracker) processBulkEntries(count int) {
	currentTime := time.Now()

	for i := 0; i < count; i++ {
		trackID := int(currentTime.UnixNano()) + i
		event := pt.createPassengerEvent(trackID, "ENTRY", 0.85, nil)

		pt.bus.Publish(eventbus.Event{
			Type:      eventbus.EventPassenger,
			Timestamp: currentTime,
			Data:      event,
		})

		pt.passengerCountCurrent++
		pt.dailyEntries++

		fmt.Printf("[Passengers] ENTRADA #%d confirmada (ID: %d)\n", i+1, trackID)
	}

	fmt.Printf("   A bordo: %d\n", pt.passengerCountCurrent)
}

// processBulkExits procesa múltiples salidas
func (pt *PassengerTracker) processBulkExits(count int) {
	currentTime := time.Now()

	for i := 0; i < count; i++ {
		trackID := int(currentTime.UnixNano()) + i
		event := pt.createPassengerEvent(trackID, "EXIT", 0.85, nil)

		pt.bus.Publish(eventbus.Event{
			Type:      eventbus.EventPassenger,
			Timestamp: currentTime,
			Data:      event,
		})

		pt.passengerCountCurrent--
		if pt.passengerCountCurrent < 0 {
			pt.passengerCountCurrent = 0
		}
		pt.dailyExits++

		fmt.Printf("[Passengers] SALIDA #%d confirmada (ID: %d)\n", i+1, trackID)
	}

	fmt.Printf("   A bordo: %d\n", pt.passengerCountCurrent)
}

// createPassengerEvent crea un evento de pasajero
func (pt *PassengerTracker) createPassengerEvent(trackID int, eventType string, confidence float64, sensorDistance *int) eventbus.PassengerEventData {
	passengerDelta := 0
	if eventType == "ENTRY" {
		passengerDelta = 1
	} else if eventType == "EXIT" {
		passengerDelta = -1
	}

	return eventbus.PassengerEventData{
		EventType:        eventType,
		TrackID:          trackID,
		Confidence:       confidence,
		SensorDistanceMM: sensorDistance,
		PassengerDelta:   passengerDelta,
		CurrentCount:     pt.passengerCountCurrent,
		TotalEntries:     pt.dailyEntries,
		TotalExits:       pt.dailyExits,
		DeviceID:         pt.config.DeviceID,
		Timestamp:        time.Now(),
	}
}

// cleanupOldTracks limpia tracks antiguos
func (pt *PassengerTracker) cleanupOldTracks(currentTime time.Time) {
	toRemove := []int{}

	for trackID, track := range pt.trackHistory {
		if currentTime.Sub(track.LastSeen).Seconds() > 30 {
			toRemove = append(toRemove, trackID)
		}
	}

	for _, trackID := range toRemove {
		delete(pt.trackHistory, trackID)
	}
}

// GetCurrentDetectedCount retorna el conteo actual detectado por YOLO
func (pt *PassengerTracker) GetCurrentDetectedCount() int {
	count := 0
	currentTime := time.Now()

	for _, track := range pt.trackHistory {
		// Contar tracks vistos recientemente (últimos 2 segundos)
		if currentTime.Sub(track.LastSeen).Seconds() <= 2.0 {
			count++
		}
	}

	return count
}

// GetStats retorna estadísticas
func (pt *PassengerTracker) GetStats() (current, entries, exits int) {
	return pt.passengerCountCurrent, pt.dailyEntries, pt.dailyExits
}
