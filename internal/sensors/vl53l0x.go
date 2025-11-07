package sensors

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/MarcosBrindi/transporte-simulator/internal/config"
	"github.com/MarcosBrindi/transporte-simulator/internal/eventbus"
)

// VL53L0XSimulator simula un sensor VL53L0X (láser de distancia para puerta)
type VL53L0XSimulator struct {
	bus       *eventbus.EventBus
	config    config.VL53L0XConfig
	threshold int // Umbral en mm (>= threshold = puerta abierta)

	// Campos protegidos por mutex
	mu              sync.RWMutex
	running         bool
	paused          bool
	distanceMM      int  // Distancia actual en mm
	isOpen          bool // Estado de la puerta
	vehicleStopped  bool // Si el vehículo está detenido
	simulationCycle int  // Ciclo de simulación
	lastOpenTime    time.Time

	// Campos de estado actual
	distance         int  // Distancia actual en mm
	isVehicleStopped bool // Si el vehículo está detenido
}

// NewVL53L0XSimulator crea un nuevo simulador VL53L0X
func NewVL53L0XSimulator(bus *eventbus.EventBus, cfg config.VL53L0XConfig) *VL53L0XSimulator {
	return &VL53L0XSimulator{
		bus:             bus,
		config:          cfg,
		threshold:       cfg.Threshold,
		running:         false,
		paused:          false,
		distanceMM:      100, // Inicialmente cerrada (cerca)
		isOpen:          false,
		vehicleStopped:  false,
		simulationCycle: 0,
		lastOpenTime:    time.Now(),
	}
}

// Start inicia el simulador en su propia goroutine
func (vl *VL53L0XSimulator) Start() {
	vl.mu.Lock()
	vl.running = true
	vl.mu.Unlock()

	go vl.loop()

	fmt.Println("[VL53L0X] Simulador iniciado")
	fmt.Printf("[VL53L0X] Umbral puerta: %dmm (>= abierta, < cerrada)\n", vl.threshold)
}

// Stop detiene el simulador
func (vl *VL53L0XSimulator) Stop() {
	vl.mu.Lock()
	vl.running = false
	vl.mu.Unlock()

	fmt.Println("[VL53L0X] Simulador detenido")
}

// Pause pausa el simulador
func (vl *VL53L0XSimulator) Pause() {
	vl.mu.Lock()
	vl.paused = true
	vl.mu.Unlock()
}

// Resume reanuda el simulador
func (vl *VL53L0XSimulator) Resume() {
	vl.mu.Lock()
	vl.paused = false
	vl.mu.Unlock()
}

// UpdateVehicleState actualiza si el vehículo está detenido
func (vl *VL53L0XSimulator) UpdateVehicleState(isStopped bool) {
	vl.mu.Lock()
	vl.vehicleStopped = isStopped
	vl.mu.Unlock()
}

// loop es el bucle principal del simulador
func (vl *VL53L0XSimulator) loop() {
	ticker := time.NewTicker(time.Duration(1000.0/vl.config.Frequency) * time.Millisecond)
	defer ticker.Stop()

	for {
		// Verificar si está corriendo
		vl.mu.RLock()
		running := vl.running
		paused := vl.paused
		vl.mu.RUnlock()

		if !running {
			break
		}

		<-ticker.C

		if paused {
			continue
		}

		// Generar datos del sensor
		data := vl.generateData()

		// Publicar evento
		vl.bus.Publish(eventbus.Event{
			Type:      eventbus.EventDoor,
			Timestamp: time.Now(),
			Data:      data,
		})
	}
}

// generateData genera datos del sensor VL53L0X sintéticos
func (vl *VL53L0XSimulator) generateData() eventbus.DoorData {
	vl.mu.Lock()
	defer vl.mu.Unlock()

	// Simular comportamiento de puerta basado en si el vehículo está detenido
	if vl.vehicleStopped {
		// Vehículo detenido: simular ciclo de apertura/cierre de puerta
		vl.simulateDoorCycle()
	} else {
		// Vehículo en movimiento: puerta siempre cerrada
		vl.distanceMM = 100 + rand.Intn(50) // 100-150mm (cerrada)
		vl.isOpen = false
	}

	// Agregar ruido realista
	noise := rand.Intn(20) - 10 // ±10mm
	distanceWithNoise := vl.distanceMM + noise

	// Clamp para evitar valores negativos
	if distanceWithNoise < 50 {
		distanceWithNoise = 50
	}

	// Determinar si la puerta está abierta según el umbral
	isOpen := distanceWithNoise >= vl.threshold

	return eventbus.DoorData{
		DistanceMM: distanceWithNoise,
		IsOpen:     isOpen,
	}
}

// simulateDoorCycle simula el ciclo de apertura/cierre de puerta
func (vl *VL53L0XSimulator) simulateDoorCycle() {
	now := time.Now()
	timeSinceLastOpen := now.Sub(vl.lastOpenTime).Seconds()

	// Ciclo de simulación:
	// Cada 30 segundos: Puerta abierta por 10 segundos, luego cerrada por 20 segundos
	cycleDuration := 30.0 // segundos
	openDuration := 10.0  // segundos

	cyclePosition := timeSinceLastOpen
	if cyclePosition >= cycleDuration {
		// Reiniciar ciclo
		vl.lastOpenTime = now
		cyclePosition = 0
	}

	if cyclePosition < openDuration {
		// Puerta ABIERTA (distancia grande)
		vl.distanceMM = 350 + rand.Intn(100) // 350-450mm
		vl.isOpen = true
	} else {
		// Puerta CERRADA (distancia pequeña)
		vl.distanceMM = 100 + rand.Intn(50) // 100-150mm
		vl.isOpen = false
	}
}

// GetCurrentState retorna el estado actual (thread-safe)
func (vl *VL53L0XSimulator) GetCurrentState() (distanceMM int, isOpen bool) {
	vl.mu.RLock()
	defer vl.mu.RUnlock()
	return vl.distanceMM, vl.isOpen
}

// Reset reinicia el sensor
func (vl *VL53L0XSimulator) Reset() {
	vl.mu.Lock()
	defer vl.mu.Unlock()

	vl.distance = vl.config.Threshold - 50
	vl.isVehicleStopped = false

	fmt.Println("🔄 [VL53L0X] Reset completado")
}

// SetFrequency cambia la frecuencia de actualización
func (vl *VL53L0XSimulator) SetFrequency(freq float64) {
	vl.mu.Lock()
	defer vl.mu.Unlock()

	vl.config.Frequency = freq
}
