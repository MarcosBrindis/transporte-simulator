package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config es la estructura principal de configuración
type Config struct {
	DeviceID   string           `yaml:"device_id"`
	Simulation SimulationConfig `yaml:"simulation"`
	Sensors    SensorsConfig    `yaml:"sensors"`
	Timeouts   TimeoutsConfig   `yaml:"timeouts"`
	Thresholds ThresholdsConfig `yaml:"thresholds"`
	MQTT       MQTTConfig       `yaml:"mqtt"`
	RabbitMQ   RabbitMQConfig   `yaml:"rabbitmq"`
	UI         UIConfig         `yaml:"ui"`
}

type SimulationConfig struct {
	InitialScenario string  `yaml:"initial_scenario"`
	Speed           float64 `yaml:"speed"`
	AutoLoop        bool    `yaml:"auto_loop"`
}

type SensorsConfig struct {
	GPS     GPSConfig     `yaml:"gps"`
	MPU6050 MPU6050Config `yaml:"mpu6050"`
	VL53L0X VL53L0XConfig `yaml:"vl53l0x"`
	Camera  CameraConfig  `yaml:"camera"`
}

type GPSConfig struct {
	Frequency       float64  `yaml:"frequency"`
	InitialPosition Position `yaml:"initial_position"`
}

type Position struct {
	Latitude  float64 `yaml:"latitude"`
	Longitude float64 `yaml:"longitude"`
}

type MPU6050Config struct {
	Frequency      float64 `yaml:"frequency"`
	AccelThreshold float64 `yaml:"accel_threshold"`
	TurnThreshold  float64 `yaml:"turn_threshold"`
}

type VL53L0XConfig struct {
	Frequency float64 `yaml:"frequency"`
	Threshold int     `yaml:"threshold"`
}

type CameraConfig struct {
	Frequency  float64 `yaml:"frequency"`
	Confidence float64 `yaml:"confidence"`
}

type TimeoutsConfig struct {
	DoorCloseConfirm float64 `yaml:"door_close_confirm"`
	MaxMonitoring    float64 `yaml:"max_monitoring"`
	ExitConfirmation float64 `yaml:"exit_confirmation"`
	EntryMin         float64 `yaml:"entry_min"`
	EntryMax         float64 `yaml:"entry_max"`
}

type ThresholdsConfig struct {
	MovementKmh float64 `yaml:"movement_kmh"`
	DistanceMM  int     `yaml:"distance_mm"`
}

// MQTTConfig configuración MQTT
type MQTTConfig struct {
	Enabled          bool             `yaml:"enabled"`
	Broker           string           `yaml:"broker"`
	ClientID         string           `yaml:"client_id"`
	Username         string           `yaml:"username"`
	Password         string           `yaml:"password"`
	QoS              byte             `yaml:"qos"`
	Retain           bool             `yaml:"retain"`
	Topics           MQTTTopicsConfig `yaml:"topics"`
	PublishInterval  float64          `yaml:"publish_interval"`
	PublishGPS       bool             `yaml:"publish_gps"`
	PublishHybrid    bool             `yaml:"publish_hybrid"`
	PublishPassenger bool             `yaml:"publish_passenger"`
	PublishDoor      bool             `yaml:"publish_door"`
}

// MQTTTopicsConfig topics MQTT
type MQTTTopicsConfig struct {
	Hybrid    string `yaml:"hybrid"`
	Passenger string `yaml:"passenger"`
	GPS       string `yaml:"gps"`
	Door      string `yaml:"door"`
	Status    string `yaml:"status"`
}

// RabbitMQConfig configuración de RabbitMQ
type RabbitMQConfig struct {
	Enabled           bool                `yaml:"enabled"`
	Host              string              `yaml:"host"`
	Port              int                 `yaml:"port"`
	Username          string              `yaml:"username"`
	Password          string              `yaml:"password"`
	VHost             string              `yaml:"vhost"`
	Exchange          string              `yaml:"exchange"`
	ExchangeType      string              `yaml:"exchange_type"`
	RoutingKeys       RabbitMQRoutingKeys `yaml:"routing_keys"`
	PublishInterval   float64             `yaml:"publish_interval"`
	PublishHybrid     bool                `yaml:"publish_hybrid"`
	PublishPassenger  bool                `yaml:"publish_passenger"`
	Heartbeat         int                 `yaml:"heartbeat"`
	ConnectionTimeout int                 `yaml:"connection_timeout"`
	PrefetchCount     int                 `yaml:"prefetch_count"`
}

// RabbitMQRoutingKeys routing keys (topics) para RabbitMQ
type RabbitMQRoutingKeys struct {
	Hybrid    string `yaml:"hybrid"`
	Passenger string `yaml:"passenger"`
}

type UIConfig struct {
	Window WindowConfig `yaml:"window"`
	Theme  string       `yaml:"theme"`
	FPS    int          `yaml:"fps"`
}

type WindowConfig struct {
	Width  int    `yaml:"width"`
	Height int    `yaml:"height"`
	Title  string `yaml:"title"`
}

// LoadConfig carga la configuración desde un archivo YAML
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error leyendo config: %w", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("error parseando YAML: %w", err)
	}

	// Reemplazar {{device_id}} y {device_id} en strings
	config = replaceDeviceIDPlaceholders(config)

	return &config, nil
}

// replaceDeviceIDPlaceholders reemplaza {{device_id}} y {device_id} en strings
func replaceDeviceIDPlaceholders(config Config) Config {
	deviceID := config.DeviceID

	// Reemplazar en título de UI
	config.UI.Window.Title = strings.ReplaceAll(
		config.UI.Window.Title,
		"{{device_id}}",
		deviceID,
	)

	// Reemplazar en topics MQTT (cada campo individualmente)
	config.MQTT.Topics.Hybrid = strings.ReplaceAll(
		config.MQTT.Topics.Hybrid,
		"{device_id}",
		deviceID,
	)

	config.MQTT.Topics.Passenger = strings.ReplaceAll(
		config.MQTT.Topics.Passenger,
		"{device_id}",
		deviceID,
	)

	config.MQTT.Topics.GPS = strings.ReplaceAll(
		config.MQTT.Topics.GPS,
		"{device_id}",
		deviceID,
	)

	config.MQTT.Topics.Door = strings.ReplaceAll(
		config.MQTT.Topics.Door,
		"{device_id}",
		deviceID,
	)

	config.MQTT.Topics.Status = strings.ReplaceAll(
		config.MQTT.Topics.Status,
		"{device_id}",
		deviceID,
	)

	// Reemplazar en routing keys de RabbitMQ
	config.RabbitMQ.RoutingKeys.Hybrid = strings.ReplaceAll(
		config.RabbitMQ.RoutingKeys.Hybrid,
		"{device_id}",
		deviceID,
	)

	config.RabbitMQ.RoutingKeys.Passenger = strings.ReplaceAll(
		config.RabbitMQ.RoutingKeys.Passenger,
		"{device_id}",
		deviceID,
	)

	return config
}

// GetTopic retorna un topic reemplazando {device_id} (método auxiliar)
func (m *MQTTConfig) GetTopic(topicTemplate string, deviceID string) string {
	return strings.ReplaceAll(topicTemplate, "{device_id}", deviceID)
}

// Default devuelve una configuración por defecto si no se puede cargar el archivo
func Default() *Config {
	return &Config{
		DeviceID: "COMBI-DEFAULT",
		Simulation: SimulationConfig{
			InitialScenario: "parada_normal",
			Speed:           1.0,
			AutoLoop:        true,
		},
		Sensors: SensorsConfig{
			GPS: GPSConfig{
				Frequency: 1.0,
				InitialPosition: Position{
					Latitude:  0.0, // Se debe configurar en config.yaml
					Longitude: 0.0, // Se debe configurar en config.yaml
				},
			},
			MPU6050: MPU6050Config{
				Frequency:      2.0,
				AccelThreshold: 0.8,
				TurnThreshold:  30.0,
			},
			VL53L0X: VL53L0XConfig{
				Frequency: 10.0,
				Threshold: 300,
			},
			Camera: CameraConfig{
				Frequency:  5.0,
				Confidence: 0.6,
			},
		},
		Timeouts: TimeoutsConfig{
			DoorCloseConfirm: 5.0,
			MaxMonitoring:    60.0,
			ExitConfirmation: 3.0,
			EntryMin:         3.0,
			EntryMax:         8.0,
		},
		Thresholds: ThresholdsConfig{
			MovementKmh: 3.0,
			DistanceMM:  300,
		},
		MQTT: MQTTConfig{
			Enabled:          false,
			Broker:           "tcp://localhost:1883",
			ClientID:         "combi-default-simulator",
			QoS:              1,
			Retain:           false,
			PublishInterval:  1.0,
			PublishGPS:       true,
			PublishHybrid:    true,
			PublishPassenger: true,
			PublishDoor:      true,
			Topics: MQTTTopicsConfig{
				Hybrid:    "vehicle/COMBI-DEFAULT/hybrid",
				Passenger: "vehicle/COMBI-DEFAULT/passenger",
				GPS:       "vehicle/COMBI-DEFAULT/gps",
				Door:      "vehicle/COMBI-DEFAULT/door",
				Status:    "vehicle/COMBI-DEFAULT/status",
			},
		},
		RabbitMQ: RabbitMQConfig{
			Enabled:           false,
			Host:              "localhost",
			Port:              5672,
			Username:          "guest",
			Password:          "guest",
			VHost:             "/",
			Exchange:          "amq.topic",
			ExchangeType:      "topic",
			PublishInterval:   1.0,
			PublishHybrid:     true,
			PublishPassenger:  true,
			Heartbeat:         60,
			ConnectionTimeout: 30,
			PrefetchCount:     1,
			RoutingKeys: RabbitMQRoutingKeys{
				Hybrid:    "vehicle.COMBI-DEFAULT.hybrid",
				Passenger: "vehicle.COMBI-DEFAULT.passenger",
			},
		},
		UI: UIConfig{
			Window: WindowConfig{
				Width:  1280,
				Height: 720,
				Title:  "Simulador Transporte - COMBI-DEFAULT",
			},
			Theme: "dark",
			FPS:   60,
		},
	}
}
