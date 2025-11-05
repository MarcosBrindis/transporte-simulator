package sensors

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/MarcosBrindi/transporte-simulator/internal/config"
	"github.com/MarcosBrindi/transporte-simulator/internal/eventbus"
	"github.com/MarcosBrindi/transporte-simulator/internal/scenario"
)

// GPSSimulator simula un sensor GPS
type GPSSimulator struct {
	bus    *eventbus.EventBus
	config config.GPSConfig
	route  *scenario.Route

	// Campos protegidos por mutex
	mu       sync.RWMutex
	running  bool
	paused   bool
	speed    float64 // Velocidad actual en km/h
	progress float64 // Progreso en la ruta (0.0 a 1.0)
}

// NewGPSSimulator crea un nuevo simulador GPS
func NewGPSSimulator(bus *eventbus.EventBus, cfg config.GPSConfig, route *scenario.Route) *GPSSimulator {
	return &GPSSimulator{
		bus:      bus,
		config:   cfg,
		route:    route,
		running:  false,
		paused:   false,
		speed:    0.0,
		progress: 0.0,
	}
}

// Start inicia el simulador en su propia goroutine
func (gps *GPSSimulator) Start() {
	gps.mu.Lock()
	gps.running = true
	gps.mu.Unlock()

	go gps.loop()

	fmt.Println("✅ [GPS] Simulador iniciado")
	fmt.Printf("📍 [GPS] Posición inicial: %.6f°, %.6f°\n",
		gps.config.InitialPosition.Latitude,
		gps.config.InitialPosition.Longitude)
}

// Stop detiene el simulador
func (gps *GPSSimulator) Stop() {
	gps.mu.Lock()
	gps.running = false
	gps.mu.Unlock()

	fmt.Println("🛑 [GPS] Simulador detenido")
}

// Pause pausa el simulador
func (gps *GPSSimulator) Pause() {
	gps.mu.Lock()
	gps.paused = true
	gps.mu.Unlock()
}

// Resume reanuda el simulador
func (gps *GPSSimulator) Resume() {
	gps.mu.Lock()
	gps.paused = false
	gps.mu.Unlock()
}

// SetSpeed establece la velocidad del vehículo (km/h)
func (gps *GPSSimulator) SetSpeed(speed float64) {
	gps.mu.Lock()
	gps.speed = speed
	gps.mu.Unlock()
}

// loop es el bucle principal del simulador
func (gps *GPSSimulator) loop() {
	ticker := time.NewTicker(time.Duration(1000.0/gps.config.Frequency) * time.Millisecond)
	defer ticker.Stop()

	for {
		// Verificar si está corriendo
		gps.mu.RLock()
		running := gps.running
		paused := gps.paused
		gps.mu.RUnlock()

		if !running {
			break
		}

		<-ticker.C

		if paused {
			continue
		}

		// Generar datos GPS
		data := gps.generateData()

		// Publicar evento
		gps.bus.Publish(eventbus.Event{
			Type:      eventbus.EventGPS,
			Timestamp: time.Now(),
			Data:      data,
		})
	}
}

// generateData genera datos GPS sintéticos
func (gps *GPSSimulator) generateData() eventbus.GPSData {
	gps.mu.Lock()
	defer gps.mu.Unlock()

	// Actualizar progreso en la ruta según velocidad
	if gps.speed > 0 {
		// Distancia recorrida en 1 segundo = velocidad (km/h) / 3600
		distanceKm := gps.speed / 3600.0

		// Progreso = distancia / longitud total de la ruta
		progressDelta := distanceKm / gps.route.Length

		gps.progress += progressDelta

		// Loop: si llegamos al final, volver al inicio
		if gps.progress >= 1.0 {
			gps.progress = 0.0
			fmt.Println("🔄 [GPS] Completó la ruta, reiniciando...")
		}
	}

	// Calcular posición actual
	lat, lon := gps.route.GetPositionAtProgress(gps.progress)

	// Calcular rumbo (course) basado en la dirección de la ruta
	course := gps.calculateCourse()

	return eventbus.GPSData{
		Latitude:   lat,
		Longitude:  lon,
		Altitude:   2240.0, // Ciudad de México promedio
		Speed:      gps.speed,
		Course:     course,
		Satellites: 8,
		FixQuality: 1,
		Progress:   gps.progress,
	}
}

// calculateCourse calcula el rumbo en grados (0-360)
// NOTA: No necesita mutex porque solo lee campos de route (inmutables)
func (gps *GPSSimulator) calculateCourse() float64 {
	// Diferencia entre punto final e inicial
	deltaLat := gps.route.EndLat - gps.route.StartLat
	deltaLon := gps.route.EndLon - gps.route.StartLon

	// Calcular ángulo (en radianes)
	angleRad := math.Atan2(deltaLon, deltaLat)

	// Convertir a grados (0-360)
	angleDeg := angleRad * 180.0 / math.Pi

	if angleDeg < 0 {
		angleDeg += 360.0
	}

	return angleDeg
}

// GetProgress retorna el progreso actual en la ruta (0.0 a 1.0)
func (gps *GPSSimulator) GetProgress() float64 {
	gps.mu.RLock()
	defer gps.mu.RUnlock()
	return gps.progress
}

// GetSpeed retorna la velocidad actual
func (gps *GPSSimulator) GetSpeed() float64 {
	gps.mu.RLock()
	defer gps.mu.RUnlock()
	return gps.speed
}

// GetCurrentStop retorna la parada más cercana
func (gps *GPSSimulator) GetCurrentStop() *scenario.Stop {
	gps.mu.RLock()
	progress := gps.progress
	gps.mu.RUnlock()

	return gps.route.GetNearestStop(progress)
}

// GetNextStop retorna la próxima parada
func (gps *GPSSimulator) GetNextStop() *scenario.Stop {
	gps.mu.RLock()
	progress := gps.progress
	gps.mu.RUnlock()

	return gps.route.GetNextStop(progress)
}
