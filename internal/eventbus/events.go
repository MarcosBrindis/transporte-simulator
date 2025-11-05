package eventbus

import "time"

// ========================================
// TIPOS DE EVENTOS
// ========================================

type EventType string

const (
	EventGPS       EventType = "gps"
	EventMPU       EventType = "mpu"
	EventDoor      EventType = "door"
	EventCamera    EventType = "camera"
	EventVehicle   EventType = "vehicle_state"
	EventPassenger EventType = "passenger"
)

// ========================================
// EVENTO GENÉRICO
// ========================================

type Event struct {
	Type      EventType
	Timestamp time.Time
	Data      interface{}
}

// ========================================
// DATOS GPS
// ========================================

type GPSData struct {
	Latitude   float64 // Grados
	Longitude  float64 // Grados
	Altitude   float64 // Metros
	Speed      float64 // km/h
	Course     float64 // Grados (0-360)
	Satellites int     // Número de satélites
	FixQuality int     // Calidad del fix (0=sin fix, 1=GPS, 2=DGPS)
	Progress   float64 // ← NUEVO: Progreso en la ruta (0.0 a 1.0)

}

// ========================================
// DATOS MPU6050 (Acelerómetro + Giroscopio)
// ========================================

type MPUData struct {
	// Aceleración (m/s²)
	AccelX float64
	AccelY float64
	AccelZ float64

	// Aceleración suavizada (buffer de 10s)
	AccelSmooth float64

	// Giroscopio (grados/segundo)
	GyroX float64
	GyroY float64
	GyroZ float64

	// Estados detectados
	IsAccelerating bool
	IsBraking      bool
	IsTurning      bool

	// Estado del vehículo según MPU
	VehicleState string // "DETENIDO", "ACELERANDO", "GIRANDO", "ACELERANDO+GIRANDO"
}

// ========================================
// DATOS VL53L0X (Sensor Láser/Puerta)
// ========================================

type DoorData struct {
	DistanceMM int  // Distancia en milímetros
	IsOpen     bool // true si distancia >= THRESHOLD (300mm)
}

// ========================================
// DATOS CÁMARA/YOLO
// ========================================

type CameraData struct {
	DetectedPersons int           // Número de personas detectadas
	Tracks          []PersonTrack // Tracks individuales
	FrameNumber     int           // Número de frame
	Confidence      float64       // Confianza promedio
}

type PersonTrack struct {
	TrackID     int     // ID único del track
	Confidence  float64 // Confianza de la detección
	BoundingBox Box     // Coordenadas del bounding box
	FirstSeen   int     // Frame en que se vio por primera vez
	LastSeen    int     // Frame en que se vio por última vez
}

type Box struct {
	X1 float64
	Y1 float64
	X2 float64
	Y2 float64
}

// ========================================
// ESTADO DEL VEHÍCULO (Agregado)
// ========================================

type VehicleStateData struct {
	State string // "DETENIDO", "MOVIMIENTO_CONFIRMADO", "GPS_MOVIMIENTO", etc.

	// Datos agregados
	Speed        float64 // km/h (del GPS)
	Acceleration float64 // m/s² (del MPU)
	TurnRate     float64 // °/s (del MPU)

	// Estados booleanos
	IsMoving  bool
	IsStopped bool
	DoorOpen  bool

	// GPS
	HasGPSFix  bool
	GPSQuality int

	// Timestamp
	Timestamp time.Time
}

// Estados posibles del vehículo (de tu Python)
const (
	VehicleDetenido             = "DETENIDO"
	VehicleDetenidoSinGPS       = "DETENIDO_SIN_GPS"
	VehicleMovimientoConfirmado = "MOVIMIENTO_CONFIRMADO"
	VehicleGPSMovimiento        = "GPS_MOVIMIENTO"
	VehicleMPUMovimiento        = "MPU_MOVIMIENTO"
	VehicleMPUMovimientoSinGPS  = "MPU_MOVIMIENTO_SIN_GPS"
)

// ========================================
// EVENTOS DE PASAJEROS
// ========================================

type PassengerEventData struct {
	EventType  string  // "ENTRY", "EXIT"
	TrackID    int     // ID del track YOLO
	Confidence float64 // Confianza de la detección

	// Sensor láser
	SensorDistanceMM *int // Distancia cuando se detectó (puede ser nil)

	// Contadores
	PassengerDelta int // +1 para entrada, -1 para salida
	CurrentCount   int // Pasajeros actuales a bordo
	TotalEntries   int // Total de entradas del día
	TotalExits     int // Total de salidas del día

	// Metadata
	DeviceID  string
	Timestamp time.Time
}

// ========================================
// ESTADOS DE LA MÁQUINA DE ESTADOS DE PUERTA
// (PASSENGER_STATES)
// ========================================

type DoorState int

const (
	DoorIdle DoorState = iota
	DoorOpened
	DoorMonitoringActive
	DoorClosing
	DoorAnalyzingChanges
	DoorConfirmingEvents
)

func (ds DoorState) String() string {
	return [...]string{
		"IDLE",
		"DOOR_OPENED",
		"MONITORING_ACTIVE",
		"DOOR_CLOSING",
		"ANALYZING_CHANGES",
		"CONFIRMING_EVENTS",
	}[ds]
}

func (ds DoorState) Description() string {
	return [...]string{
		"Sin actividad",
		"Puerta abierta - monitoreando",
		"Monitoreo activo - esperando cierre de puerta",
		"Puerta cerrada - confirmando cierre",
		"Analizando cambios de pasajeros",
		"Confirmando entradas/salidas",
	}[ds]
}

// ========================================
// CONFIGURACIÓN DE TIMEOUTS
// ========================================

const (
	// Timeouts en segundos (convertir a time.Duration en uso)
	SensorCorrelationWindow   = 10.0 // Ventana para correlacionar sensor + YOLO
	EntryMinTime              = 3.0  // Mínimo para confirmar entrada
	EntryMaxTime              = 8.0  // Máximo antes de timeout
	ExitConfirmationTime      = 3.0  // Para confirmar salidas
	DoorCloseConfirmationTime = 5.0  // Tiempo para confirmar que puerta está cerrada
	MaxMonitoringTime         = 60.0 // Timeout máximo de seguridad
	YOLODetectionDelay        = 2.0  // Tiempo típico que tarda YOLO
	MovementTimeout           = 10.0 // Máximo antes de considerar movimiento temporal

	// Umbrales
	DistanceThresholdMM  = 300  // mm - umbral sensor VL53L0X (30cm)
	MovementThresholdKmh = 3.0  // km/h - umbral de velocidad para considerar movimiento
	AccelThresholdMS2    = 0.8  // m/s² - umbral de aceleración
	TurnThresholdDPS     = 30.0 // °/s - umbral de giro
)
