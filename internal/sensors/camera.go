package sensors

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/MarcosBrindi/transporte-simulator/internal/config"
	"github.com/MarcosBrindi/transporte-simulator/internal/eventbus"
)

// CameraSimulator simula una cámara con detector YOLO
type CameraSimulator struct {
	bus    *eventbus.EventBus
	config config.CameraConfig

	// Campos protegidos por mutex
	mu             sync.RWMutex
	running        bool
	paused         bool
	frameNumber    int
	doorOpen       bool
	vehicleStopped bool
	activeTracks   map[int]*PersonTrackState // Tracks activos
	nextTrackID    int
	personCount    int // Número de personas simuladas en la puerta
}

// PersonTrackState mantiene el estado de un track
type PersonTrackState struct {
	TrackID    int
	FirstSeen  int
	LastSeen   int
	Confidence float64
}

// NewCameraSimulator crea un nuevo simulador de cámara
func NewCameraSimulator(bus *eventbus.EventBus, cfg config.CameraConfig) *CameraSimulator {
	return &CameraSimulator{
		bus:            bus,
		config:         cfg,
		running:        false,
		paused:         false,
		frameNumber:    0,
		doorOpen:       false,
		vehicleStopped: false,
		activeTracks:   make(map[int]*PersonTrackState),
		nextTrackID:    1,
		personCount:    0,
	}
}

// Start inicia el simulador en su propia goroutine
func (cam *CameraSimulator) Start() {
	cam.mu.Lock()
	cam.running = true
	cam.mu.Unlock()

	go cam.loop()

	fmt.Println("✅ [Camera] Simulador iniciado")
	fmt.Printf("📷 [Camera] Frecuencia: %.1f Hz (%.0fms/frame)\n",
		cam.config.Frequency, 1000.0/cam.config.Frequency)
}

// Stop detiene el simulador
func (cam *CameraSimulator) Stop() {
	cam.mu.Lock()
	cam.running = false
	cam.mu.Unlock()

	fmt.Println("[Camera] Simulador detenido")
}

// Pause pausa el simulador
func (cam *CameraSimulator) Pause() {
	cam.mu.Lock()
	cam.paused = true
	cam.mu.Unlock()
}

// Resume reanuda el simulador
func (cam *CameraSimulator) Resume() {
	cam.mu.Lock()
	cam.paused = false
	cam.mu.Unlock()
}

// UpdateDoorState actualiza el estado de la puerta
func (cam *CameraSimulator) UpdateDoorState(doorOpen bool) {
	cam.mu.Lock()
	cam.doorOpen = doorOpen
	cam.mu.Unlock()
}

// UpdateVehicleState actualiza si el vehículo está detenido
func (cam *CameraSimulator) UpdateVehicleState(isStopped bool) {
	cam.mu.Lock()
	cam.vehicleStopped = isStopped
	cam.mu.Unlock()
}

// loop es el bucle principal del simulador
func (cam *CameraSimulator) loop() {
	ticker := time.NewTicker(time.Duration(1000.0/cam.config.Frequency) * time.Millisecond)
	defer ticker.Stop()

	for {
		cam.mu.RLock()
		running := cam.running
		paused := cam.paused
		cam.mu.RUnlock()

		if !running {
			break
		}

		<-ticker.C

		if paused {
			continue
		}

		// Generar frame
		data := cam.generateFrame()

		// Publicar evento
		cam.bus.Publish(eventbus.Event{
			Type:      eventbus.EventCamera,
			Timestamp: time.Now(),
			Data:      data,
		})

		cam.mu.Lock()
		cam.frameNumber++
		cam.mu.Unlock()
	}
}

// generateFrame genera un frame sintético con detecciones YOLO
func (cam *CameraSimulator) generateFrame() eventbus.CameraData {
	cam.mu.Lock()
	defer cam.mu.Unlock()

	// Solo detectar personas si:
	// 1. Vehículo está detenido
	// 2. Puerta está abierta
	if !cam.vehicleStopped || !cam.doorOpen {
		// Sin detecciones
		cam.activeTracks = make(map[int]*PersonTrackState)
		cam.personCount = 0

		return eventbus.CameraData{
			DetectedPersons: 0,
			Tracks:          []eventbus.PersonTrack{},
			FrameNumber:     cam.frameNumber,
			Confidence:      0.0,
		}
	}

	// Simular detección de personas cuando la puerta está abierta
	cam.simulatePersonDetections()

	// Convertir tracks activos a slice
	tracks := make([]eventbus.PersonTrack, 0, len(cam.activeTracks))
	totalConfidence := 0.0

	for _, track := range cam.activeTracks {
		tracks = append(tracks, eventbus.PersonTrack{
			TrackID:    track.TrackID,
			Confidence: track.Confidence,
			BoundingBox: eventbus.Box{
				X1: 100 + float64(rand.Intn(200)),
				Y1: 100 + float64(rand.Intn(200)),
				X2: 300 + float64(rand.Intn(200)),
				Y2: 400 + float64(rand.Intn(100)),
			},
			FirstSeen: track.FirstSeen,
			LastSeen:  cam.frameNumber,
		})

		totalConfidence += track.Confidence
		track.LastSeen = cam.frameNumber
	}

	avgConfidence := 0.0
	if len(tracks) > 0 {
		avgConfidence = totalConfidence / float64(len(tracks))
	}

	return eventbus.CameraData{
		DetectedPersons: len(tracks),
		Tracks:          tracks,
		FrameNumber:     cam.frameNumber,
		Confidence:      avgConfidence,
	}
}

// simulatePersonDetections simula detecciones de personas
func (cam *CameraSimulator) simulatePersonDetections() {
	// Generar cambios en el número de personas cada ~3 segundos
	// (asumiendo 5 FPS, cada 15 frames)
	if cam.frameNumber%15 == 0 {
		// Decidir si agregar/quitar personas
		change := rand.Intn(5) - 1 // -1, 0, 1, 2, 3 (bias hacia agregar)

		cam.personCount += change
		if cam.personCount < 0 {
			cam.personCount = 0
		}
		if cam.personCount > 5 {
			cam.personCount = 5
		}

		// Ajustar tracks según nuevo conteo
		cam.adjustTracks()
	}

	// Mantener tracks existentes (actualizar LastSeen se hace en generateFrame)
}

// adjustTracks ajusta los tracks según el conteo deseado
func (cam *CameraSimulator) adjustTracks() {
	currentCount := len(cam.activeTracks)
	targetCount := cam.personCount

	if targetCount > currentCount {
		// Agregar nuevos tracks
		toAdd := targetCount - currentCount
		for i := 0; i < toAdd; i++ {
			trackID := cam.nextTrackID
			cam.nextTrackID++

			cam.activeTracks[trackID] = &PersonTrackState{
				TrackID:    trackID,
				FirstSeen:  cam.frameNumber,
				LastSeen:   cam.frameNumber,
				Confidence: 0.7 + rand.Float64()*0.25, // 0.7-0.95
			}

			fmt.Printf("👤 [Camera] Nuevo track detectado: ID=%d (frame %d)\n", trackID, cam.frameNumber)
		}
	} else if targetCount < currentCount {
		// Remover tracks (simular que personas salieron del campo de visión)
		toRemove := currentCount - targetCount
		removed := 0

		for trackID := range cam.activeTracks {
			if removed >= toRemove {
				break
			}
			fmt.Printf("👋 [Camera] Track perdido: ID=%d (frame %d)\n", trackID, cam.frameNumber)
			delete(cam.activeTracks, trackID)
			removed++
		}
	}
}

// GetActiveTracksCount retorna el número de tracks activos
func (cam *CameraSimulator) GetActiveTracksCount() int {
	cam.mu.RLock()
	defer cam.mu.RUnlock()
	return len(cam.activeTracks)
}
