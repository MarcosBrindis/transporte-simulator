package mqtt

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/MarcosBrindi/transporte-simulator/internal/config"
	"github.com/MarcosBrindi/transporte-simulator/internal/eventbus"
	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQPublisher publica eventos a RabbitMQ
type RabbitMQPublisher struct {
	config   config.RabbitMQConfig
	deviceID string
	channel  *amqp.Channel
	bus      *eventbus.EventBus

	// Estado
	mu          sync.RWMutex
	running     bool
	connected   bool
	lastGPS     eventbus.GPSData
	lastMPU     eventbus.MPUData
	lastVehicle eventbus.VehicleStateData
	hasGPS      bool
	hasMPU      bool
	hasVehicle  bool

	// Channels
	gpsEvents       chan eventbus.Event
	mpuEvents       chan eventbus.Event
	vehicleEvents   chan eventbus.Event
	passengerEvents chan eventbus.Event
}

// NewRabbitMQPublisher crea un nuevo publicador RabbitMQ con canal compartido
func NewRabbitMQPublisher(ch *amqp.Channel, cfg config.RabbitMQConfig, deviceID string, bus *eventbus.EventBus) *RabbitMQPublisher {
	return &RabbitMQPublisher{
		config:          cfg,
		deviceID:        deviceID,
		channel:         ch,
		bus:             bus,
		running:         false,
		connected:       true,
		gpsEvents:       make(chan eventbus.Event, 10),
		mpuEvents:       make(chan eventbus.Event, 10),
		vehicleEvents:   make(chan eventbus.Event, 10),
		passengerEvents: make(chan eventbus.Event, 10),
	}
}

// Start inicia el publicador
func (p *RabbitMQPublisher) Start() error {
	if !p.config.Enabled {
		fmt.Println("ℹ️  [RabbitMQ] Deshabilitado en configuración")
		return nil
	}

	p.mu.Lock()
	p.running = true
	p.mu.Unlock()

	if p.channel == nil {
		return fmt.Errorf("canal RabbitMQ no inicializado")
	}

	fmt.Println("✅ [RabbitMQ] Publicador iniciado")
	fmt.Printf("📤 [RabbitMQ] Exchange: %s (type: %s)\n", p.config.Exchange, p.config.ExchangeType)
	fmt.Printf("🔑 [RabbitMQ] Device ID: %s\n", p.deviceID)

	// Suscribirse a eventos del bus
	p.subscribeToEvents()

	// Iniciar publicación periódica
	go p.publishLoop()

	return nil
}

// Stop detiene el publicador
func (p *RabbitMQPublisher) Stop() {
	p.mu.Lock()
	p.running = false
	p.mu.Unlock()

	fmt.Printf("🛑 [RabbitMQ] Publicador detenido (%s)\n", p.deviceID)
}

// subscribeToEvents suscribe a eventos del bus
func (p *RabbitMQPublisher) subscribeToEvents() {
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
func (p *RabbitMQPublisher) publishLoop() {
	ticker := time.NewTicker(time.Duration(p.config.PublishInterval * float64(time.Second)))
	defer ticker.Stop()

	for p.isRunning() {
		select {
		case gpsEvent := <-p.gpsEvents:
			p.handleGPS(gpsEvent)

		case mpuEvent := <-p.mpuEvents:
			p.handleMPU(mpuEvent)

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
func (p *RabbitMQPublisher) handleGPS(event eventbus.Event) {
	data := event.Data.(eventbus.GPSData)

	p.mu.Lock()
	p.lastGPS = data
	p.hasGPS = true
	p.mu.Unlock()
}

// handleMPU procesa eventos MPU
func (p *RabbitMQPublisher) handleMPU(event eventbus.Event) {
	data := event.Data.(eventbus.MPUData)

	p.mu.Lock()
	p.lastMPU = data
	p.hasMPU = true
	p.mu.Unlock()
}

// handleVehicle procesa eventos Vehicle
func (p *RabbitMQPublisher) handleVehicle(event eventbus.Event) {
	data := event.Data.(eventbus.VehicleStateData)

	p.mu.Lock()
	p.lastVehicle = data
	p.hasVehicle = true
	p.mu.Unlock()
}

// handlePassenger procesa eventos Passenger
func (p *RabbitMQPublisher) handlePassenger(event eventbus.Event) {
	data := event.Data.(eventbus.PassengerEventData)

	if p.config.PublishPassenger {
		p.publishPassenger(data)
	}
}

// publishPassenger publica eventos de pasajeros
func (p *RabbitMQPublisher) publishPassenger(data eventbus.PassengerEventData) {
	routingKey := p.config.RoutingKeys.Passenger

	payload := map[string]interface{}{
		"timestamp":       data.Timestamp.Unix(),
		"device_id":       p.deviceID,
		"sensor_type":     "PASSENGER_COUNTER",
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

	p.publish(routingKey, payload)
}

// publishHybrid publica mensaje híbrido (GPS + MPU + Estado)
func (p *RabbitMQPublisher) publishHybrid() {
	p.mu.RLock()
	if !p.hasGPS || !p.hasMPU || !p.hasVehicle {
		p.mu.RUnlock()
		return
	}

	gps := p.lastGPS
	mpu := p.lastMPU
	vehicle := p.lastVehicle
	p.mu.RUnlock()

	routingKey := p.config.RoutingKeys.Hybrid

	// Formato exacto como el Python
	payload := map[string]interface{}{
		"timestamp":   time.Now().Unix(),
		"device_id":   p.deviceID,
		"sensor_type": "HYBRID_GPS_MPU",
		"data": map[string]interface{}{
			"latitude":         gps.Latitude,
			"longitude":        gps.Longitude,
			"speed_kmh":        gps.Speed,
			"acceleration_ms2": mpu.AccelSmooth,
			"turn_rate_dps":    mpu.GyroZ,
			"vehicle_state":    vehicle.State,
		},
	}

	p.publish(routingKey, payload)
}

// publish publica un mensaje a RabbitMQ
func (p *RabbitMQPublisher) publish(routingKey string, payload interface{}) {
	if !p.isConnected() {
		return
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("⚠️  [RabbitMQ] Error serializando JSON: %v\n", err)
		return
	}

	p.mu.RLock()
	channel := p.channel
	exchange := p.config.Exchange
	p.mu.RUnlock()

	if channel == nil {
		return
	}

	err = channel.Publish(
		exchange,   // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        jsonData,
			Timestamp:   time.Now(),
		},
	)

	if err != nil {
		fmt.Printf("⚠️  [RabbitMQ] Error publicando a %s: %v\n", routingKey, err)
		p.mu.Lock()
		p.connected = false
		p.mu.Unlock()
	} else {
		// Log exitoso (solo para depuración, puedes comentar después)
		// fmt.Printf("📤 [RabbitMQ] Publicado → %s\n", routingKey)
	}
}

// isRunning verifica si está corriendo
func (p *RabbitMQPublisher) isRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

// isConnected verifica si está conectado
func (p *RabbitMQPublisher) isConnected() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.connected
}

// ConnectRabbitMQ establece conexión a RabbitMQ y retorna la conexión
func ConnectRabbitMQ(cfg config.RabbitMQConfig) (*amqp.Connection, error) {
	url := fmt.Sprintf(
		"amqp://%s:%s@%s:%d/%s",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.VHost,
	)

	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("error conectando a RabbitMQ: %w", err)
	}

	return conn, nil
}
