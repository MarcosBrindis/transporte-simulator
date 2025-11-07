package sensors

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/MarcosBrindi/transporte-simulator/internal/config"
	"github.com/MarcosBrindi/transporte-simulator/internal/eventbus"
)

// MPU6050Simulator simula un sensor MPU6050 (acelerómetro + giroscopio)
type MPU6050Simulator struct {
	bus    *eventbus.EventBus
	config config.MPU6050Config

	// Campos protegidos por mutex
	mu             sync.RWMutex
	running        bool
	paused         bool
	currentSpeed   float64   // Velocidad actual (del GPS)
	previousSpeed  float64   // Velocidad anterior (para calcular aceleración)
	accelBuffer    []float64 // Buffer para suavizar aceleración
	lastUpdateTime time.Time

	// Campos de estado actual
	lastSpeed   float64
	accelSmooth float64
	gyroZ       float64
}

// NewMPU6050Simulator crea un nuevo simulador MPU6050
func NewMPU6050Simulator(bus *eventbus.EventBus, cfg config.MPU6050Config) *MPU6050Simulator {
	return &MPU6050Simulator{
		bus:            bus,
		config:         cfg,
		running:        false,
		paused:         false,
		currentSpeed:   0.0,
		previousSpeed:  0.0,
		accelBuffer:    make([]float64, 0, 20), // Buffer de 20 muestras (10s a 2Hz)
		lastUpdateTime: time.Now(),
	}
}

// Start inicia el simulador en su propia goroutine
func (mpu *MPU6050Simulator) Start() {
	mpu.mu.Lock()
	mpu.running = true
	mpu.lastUpdateTime = time.Now()
	mpu.mu.Unlock()

	go mpu.loop()

	fmt.Println("✅ [MPU6050] Simulador iniciado")
}

// Stop detiene el simulador
func (mpu *MPU6050Simulator) Stop() {
	mpu.mu.Lock()
	mpu.running = false
	mpu.mu.Unlock()

	fmt.Println("[MPU6050] Simulador detenido")
}

// Pause pausa el simulador
func (mpu *MPU6050Simulator) Pause() {
	mpu.mu.Lock()
	mpu.paused = true
	mpu.mu.Unlock()
}

// Resume reanuda el simulador
func (mpu *MPU6050Simulator) Resume() {
	mpu.mu.Lock()
	mpu.paused = false
	mpu.mu.Unlock()
}

// UpdateSpeed actualiza la velocidad (llamado desde GPS o State Manager)
func (mpu *MPU6050Simulator) UpdateSpeed(speed float64) {
	mpu.mu.Lock()
	mpu.currentSpeed = speed
	mpu.mu.Unlock()
}

// loop es el bucle principal del simulador
func (mpu *MPU6050Simulator) loop() {
	ticker := time.NewTicker(time.Duration(1000.0/mpu.config.Frequency) * time.Millisecond)
	defer ticker.Stop()

	for {
		// Verificar si está corriendo
		mpu.mu.RLock()
		running := mpu.running
		paused := mpu.paused
		mpu.mu.RUnlock()

		if !running {
			break
		}

		<-ticker.C

		if paused {
			continue
		}

		// Generar datos MPU
		data := mpu.generateData()

		// Publicar evento
		mpu.bus.Publish(eventbus.Event{
			Type:      eventbus.EventMPU,
			Timestamp: time.Now(),
			Data:      data,
		})
	}
}

// generateData genera datos MPU6050 sintéticos
func (mpu *MPU6050Simulator) generateData() eventbus.MPUData {
	mpu.mu.Lock()
	defer mpu.mu.Unlock()

	// Calcular delta de tiempo
	now := time.Now()
	deltaTime := now.Sub(mpu.lastUpdateTime).Seconds()
	mpu.lastUpdateTime = now

	// Calcular aceleración longitudinal (m/s²)
	// a = (v_final - v_inicial) / delta_t
	// Convertir km/h a m/s: v_ms = v_kmh / 3.6
	currentSpeedMS := mpu.currentSpeed / 3.6
	previousSpeedMS := mpu.previousSpeed / 3.6

	accelLongitudinal := 0.0
	if deltaTime > 0 {
		accelLongitudinal = (currentSpeedMS - previousSpeedMS) / deltaTime
	}

	// Agregar ruido realista
	noise := (rand.Float64() - 0.5) * 0.1 // ±0.05 m/s²
	accelLongitudinal += noise

	// Actualizar velocidad anterior
	mpu.previousSpeed = mpu.currentSpeed

	// Agregar a buffer para suavizado
	mpu.accelBuffer = append(mpu.accelBuffer, math.Abs(accelLongitudinal))
	if len(mpu.accelBuffer) > 20 {
		mpu.accelBuffer = mpu.accelBuffer[1:]
	}

	// Calcular aceleración suavizada
	accelSmooth := 0.0
	if len(mpu.accelBuffer) > 0 {
		sum := 0.0
		for _, a := range mpu.accelBuffer {
			sum += a
		}
		accelSmooth = sum / float64(len(mpu.accelBuffer))
	}

	// Simular componentes de aceleración (X, Y, Z)
	// X: longitudinal (adelante/atrás)
	// Y: lateral (izquierda/derecha)
	// Z: vertical (arriba/abajo) - gravedad + vibraciones
	accelX := accelLongitudinal
	accelY := (rand.Float64() - 0.5) * 0.2    // Pequeñas variaciones laterales
	accelZ := 9.81 + (rand.Float64()-0.5)*0.3 // Gravedad + vibraciones

	// Simular giroscopio (grados/segundo)
	// Giro solo si hay velocidad (no gira si está detenido)
	gyroZ := 0.0
	if mpu.currentSpeed > 5.0 { // Solo gira si va a más de 5 km/h
		// Giros ocasionales (20% de probabilidad)
		if rand.Float64() < 0.2 {
			gyroZ = (rand.Float64() - 0.5) * 60.0 // ±30°/s
		}
	}

	gyroX := (rand.Float64() - 0.5) * 2.0 // Pitch mínimo
	gyroY := (rand.Float64() - 0.5) * 2.0 // Roll mínimo

	// Detectar estados (umbrales del config.yaml)
	isAccelerating := accelSmooth > mpu.config.AccelThreshold
	isBraking := accelX < -mpu.config.AccelThreshold
	isTurning := math.Abs(gyroZ) > mpu.config.TurnThreshold

	// Determinar estado del vehículo según MPU
	vehicleState := "DETENIDO"
	if isAccelerating && isTurning {
		vehicleState = "ACELERANDO+GIRANDO"
	} else if isAccelerating {
		vehicleState = "ACELERANDO"
	} else if isBraking {
		vehicleState = "FRENANDO"
	} else if isTurning {
		vehicleState = "GIRANDO"
	}

	return eventbus.MPUData{
		AccelX:         accelX,
		AccelY:         accelY,
		AccelZ:         accelZ,
		AccelSmooth:    accelSmooth,
		GyroX:          gyroX,
		GyroY:          gyroY,
		GyroZ:          gyroZ,
		IsAccelerating: isAccelerating,
		IsBraking:      isBraking,
		IsTurning:      isTurning,
		VehicleState:   vehicleState,
	}
}

// Reset reinicia el MPU6050
func (mpu *MPU6050Simulator) Reset() {
	mpu.mu.Lock()
	defer mpu.mu.Unlock()

	mpu.currentSpeed = 0.0
	mpu.lastSpeed = 0.0
	mpu.accelSmooth = 0.0
	mpu.gyroZ = 0.0

	fmt.Println("🔄 [MPU6050] Reset completado")
}

// SetFrequency cambia la frecuencia de actualización
func (mpu *MPU6050Simulator) SetFrequency(freq float64) {
	mpu.mu.Lock()
	defer mpu.mu.Unlock()

	mpu.config.Frequency = freq
}
