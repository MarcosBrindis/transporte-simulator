package mqtt

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/MarcosBrindi/transporte-simulator/internal/config"
	"github.com/MarcosBrindi/transporte-simulator/internal/eventbus"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Publisher publica eventos a MQTT
type Publisher struct {
	config   config.MQTTConfig
	deviceID string
	client   mqtt.Client
	bus      *eventbus.EventBus

	// Estado
	mu          sync.RWMutex
	running     bool
	connected   bool
	lastGPS     eventbus.GPSData
	lastMPU     eventbus.MPUData
	lastDoor    eventbus.DoorData
	lastVehicle eventbus.VehicleStateData
	hasGPS      bool
	hasMPU      bool
	hasDoor     bool
	hasVehicle  bool

	// Channels
	gpsEvents       chan eventbus.Event
	mpuEvents       chan eventbus.Event
	doorEvents      chan eventbus.Event
	vehicleEvents   chan eventbus.Event
	passengerEvents chan eventbus.Event
}

// NewPublisher crea un nuevo publicador MQTT
func NewPublisher(cfg config.MQTTConfig, deviceID string, bus *eventbus.EventBus) *Publisher {
	return &Publisher{
		config:          cfg,
		deviceID:        deviceID,
		bus:             bus,
		running:         false,
		connected:       false,
		gpsEvents:       make(chan eventbus.Event, 10),
		mpuEvents:       make(chan eventbus.Event, 10),
		doorEvents:      make(chan eventbus.Event, 10),
		vehicleEvents:   make(chan eventbus.Event, 10),
		passengerEvents: make(chan eventbus.Event, 10),
	}
}

// Start inicia el publicador
func (p *Publisher) Start() error {
	if !p.config.Enabled {
		fmt.Println("ℹ️  [MQTT] Deshabilitado en configuración")
		return nil
	}

	p.mu.Lock()
	p.running = true
	p.mu.Unlock()

	// Configurar cliente MQTT
	opts := mqtt.NewClientOptions()
	opts.AddBroker(p.config.Broker)
	opts.SetClientID(p.config.ClientID)

	if p.config.Username != "" {
		opts.SetUsername(p.config.Username)
		opts.SetPassword(p.config.Password)
	}

	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)
	opts.SetMaxReconnectInterval(1 * time.Minute)

	// Callbacks
	opts.SetOnConnectHandler(p.onConnect)
	opts.SetConnectionLostHandler(p.onConnectionLost)

	p.client = mqtt.NewClient(opts)

	// Conectar
	fmt.Printf("📡 [MQTT] Conectando a %s...\n", p.config.Broker)

	token := p.client.Connect()
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("error conectando a MQTT: %w", token.Error())
	}

	// Suscribirse a eventos del bus
	p.subscribeToEvents()

	// Iniciar publicación periódica
	go p.publishLoop()

	return nil
}

// Stop detiene el publicador
func (p *Publisher) Stop() {
	p.mu.Lock()
	p.running = false
	p.mu.Unlock()

	if p.client != nil && p.client.IsConnected() {
		// Publicar mensaje de desconexión
		p.publishStatus("offline")

		p.client.Disconnect(250)
		fmt.Println("🛑 [MQTT] Desconectado")
	}
}

// onConnect callback cuando se conecta
func (p *Publisher) onConnect(client mqtt.Client) {
	p.mu.Lock()
	p.connected = true
	p.mu.Unlock()

	fmt.Println("✅ [MQTT] Conectado exitosamente")

	// Publicar mensaje de conexión
	p.publishStatus("online")
}

// onConnectionLost callback cuando se pierde conexión
func (p *Publisher) onConnectionLost(client mqtt.Client, err error) {
	p.mu.Lock()
	p.connected = false
	p.mu.Unlock()

	fmt.Printf("⚠️  [MQTT] Conexión perdida: %v\n", err)
	fmt.Println("🔄 [MQTT] Intentando reconectar...")
}

// subscribeToEvents suscribe a eventos del bus
func (p *Publisher) subscribeToEvents() {
	// GPS
	gpsChannel := p.bus.Subscribe(eventbus.EventGPS)
	go func() {
		for event := range gpsChannel {
			if p.isRunning() {
				select {
				case p.gpsEvents <- event:
				default:
				}
			}
		}
	}()

	// MPU
	mpuChannel := p.bus.Subscribe(eventbus.EventMPU)
	go func() {
		for event := range mpuChannel {
			if p.isRunning() {
				select {
				case p.mpuEvents <- event:
				default:
				}
			}
		}
	}()

	// Door
	doorChannel := p.bus.Subscribe(eventbus.EventDoor)
	go func() {
		for event := range doorChannel {
			if p.isRunning() {
				select {
				case p.doorEvents <- event:
				default:
				}
			}
		}
	}()

	// Vehicle
	vehicleChannel := p.bus.Subscribe(eventbus.EventVehicle)
	go func() {
		for event := range vehicleChannel {
			if p.isRunning() {
				select {
				case p.vehicleEvents <- event:
				default:
				}
			}
		}
	}()

	// Passenger
	passengerChannel := p.bus.Subscribe(eventbus.EventPassenger)
	go func() {
		for event := range passengerChannel {
			if p.isRunning() {
				select {
				case p.passengerEvents <- event:
				default:
				}
			}
		}
	}()
}

// publishLoop publica periódicamente
func (p *Publisher) publishLoop() {
	ticker := time.NewTicker(time.Duration(p.config.PublishInterval * float64(time.Second)))
	defer ticker.Stop()

	for p.isRunning() {
		select {
		case gpsEvent := <-p.gpsEvents:
			p.handleGPS(gpsEvent)

		case mpuEvent := <-p.mpuEvents:
			p.handleMPU(mpuEvent)

		case doorEvent := <-p.doorEvents:
			p.handleDoor(doorEvent)

		case vehicleEvent := <-p.vehicleEvents:
			p.handleVehicle(vehicleEvent)

		case passengerEvent := <-p.passengerEvents:
			p.handlePassenger(passengerEvent)

		case <-ticker.C:
			// Publicar estado híbrido cada intervalo
			if p.config.PublishHybrid {
				p.publishHybrid()
			}
		}
	}
}

// handleGPS procesa eventos GPS
func (p *Publisher) handleGPS(event eventbus.Event) {
	data := event.Data.(eventbus.GPSData)

	p.mu.Lock()
	p.lastGPS = data
	p.hasGPS = true
	p.mu.Unlock()

	if p.config.PublishGPS {
		p.publishGPS(data)
	}
}

// handleMPU procesa eventos MPU
func (p *Publisher) handleMPU(event eventbus.Event) {
	data := event.Data.(eventbus.MPUData)

	p.mu.Lock()
	p.lastMPU = data
	p.hasMPU = true
	p.mu.Unlock()
}

// handleDoor procesa eventos Door
func (p *Publisher) handleDoor(event eventbus.Event) {
	data := event.Data.(eventbus.DoorData)

	p.mu.Lock()
	p.lastDoor = data
	p.hasDoor = true
	p.mu.Unlock()

	if p.config.PublishDoor {
		p.publishDoor(data)
	}
}

// handleVehicle procesa eventos Vehicle
func (p *Publisher) handleVehicle(event eventbus.Event) {
	data := event.Data.(eventbus.VehicleStateData)

	p.mu.Lock()
	p.lastVehicle = data
	p.hasVehicle = true
	p.mu.Unlock()
}

// handlePassenger procesa eventos Passenger
func (p *Publisher) handlePassenger(event eventbus.Event) {
	data := event.Data.(eventbus.PassengerEventData)

	if p.config.PublishPassenger {
		p.publishPassenger(data)
	}
}

// publishGPS publica datos GPS
func (p *Publisher) publishGPS(data eventbus.GPSData) {
	topic := p.config.GetTopic(p.config.Topics.GPS, p.deviceID)

	payload := map[string]interface{}{
		"device_id":   p.deviceID,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"latitude":    data.Latitude,
		"longitude":   data.Longitude,
		"altitude":    data.Altitude,
		"speed":       data.Speed,
		"course":      data.Course,
		"satellites":  data.Satellites,
		"fix_quality": data.FixQuality,
		"progress":    data.Progress,
	}

	p.publish(topic, payload)
}

// publishDoor publica datos de puerta
func (p *Publisher) publishDoor(data eventbus.DoorData) {
	topic := p.config.GetTopic(p.config.Topics.Door, p.deviceID)

	payload := map[string]interface{}{
		"device_id":   p.deviceID,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"distance_mm": data.DistanceMM,
		"is_open":     data.IsOpen,
	}

	p.publish(topic, payload)
}

// publishPassenger publica eventos de pasajeros
func (p *Publisher) publishPassenger(data eventbus.PassengerEventData) {
	topic := p.config.GetTopic(p.config.Topics.Passenger, p.deviceID)

	payload := map[string]interface{}{
		"device_id":       p.deviceID,
		"timestamp":       data.Timestamp.UTC().Format(time.RFC3339),
		"event_type":      data.EventType,
		"track_id":        data.TrackID,
		"confidence":      data.Confidence,
		"passenger_delta": data.PassengerDelta,
		"current_count":   data.CurrentCount,
		"total_entries":   data.TotalEntries,
		"total_exits":     data.TotalExits,
	}

	if data.SensorDistanceMM != nil {
		payload["sensor_distance_mm"] = *data.SensorDistanceMM
	}

	p.publish(topic, payload)
}

// publishHybrid publica mensaje híbrido (GPS + MPU + Estado)
func (p *Publisher) publishHybrid() {
	p.mu.RLock()
	if !p.hasGPS || !p.hasMPU || !p.hasVehicle {
		p.mu.RUnlock()
		return
	}

	gps := p.lastGPS
	mpu := p.lastMPU
	vehicle := p.lastVehicle
	door := p.lastDoor
	p.mu.RUnlock()

	topic := p.config.GetTopic(p.config.Topics.Hybrid, p.deviceID)

	payload := map[string]interface{}{
		"device_id": p.deviceID,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"gps": map[string]interface{}{
			"latitude":   gps.Latitude,
			"longitude":  gps.Longitude,
			"speed":      gps.Speed,
			"satellites": gps.Satellites,
			"progress":   gps.Progress,
		},
		"motion": map[string]interface{}{
			"acceleration":    mpu.AccelSmooth,
			"turn_rate":       mpu.GyroZ,
			"is_accelerating": mpu.IsAccelerating,
			"is_braking":      mpu.IsBraking,
			"is_turning":      mpu.IsTurning,
		},
		"vehicle": map[string]interface{}{
			"state":       vehicle.State,
			"is_moving":   vehicle.IsMoving,
			"is_stopped":  vehicle.IsStopped,
			"door_open":   vehicle.DoorOpen,
			"has_gps_fix": vehicle.HasGPSFix,
		},
		"door": map[string]interface{}{
			"is_open":     door.IsOpen,
			"distance_mm": door.DistanceMM,
		},
	}

	p.publish(topic, payload)
}

// publishStatus publica estado de conexión
func (p *Publisher) publishStatus(status string) {
	topic := p.config.GetTopic(p.config.Topics.Status, p.deviceID)

	payload := map[string]interface{}{
		"device_id": p.deviceID,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"status":    status,
	}

	p.publish(topic, payload)
}

// publish publica un mensaje MQTT
func (p *Publisher) publish(topic string, payload interface{}) {
	if !p.isConnected() {
		return
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("⚠️  [MQTT] Error serializando JSON: %v\n", err)
		return
	}

	token := p.client.Publish(topic, p.config.QoS, p.config.Retain, jsonData)
	token.Wait()

	if token.Error() != nil {
		fmt.Printf("⚠️  [MQTT] Error publicando a %s: %v\n", topic, token.Error())
	}
}

// isRunning verifica si está corriendo
func (p *Publisher) isRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

// isConnected verifica si está conectado
func (p *Publisher) isConnected() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.connected
}
