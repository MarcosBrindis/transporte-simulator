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

type MQTTConfig struct {
	Enabled  bool              `yaml:"enabled"`
	Broker   string            `yaml:"broker"`
	Port     int               `yaml:"port"`
	Username string            `yaml:"username"`
	Password string            `yaml:"password"`
	Topics   map[string]string `yaml:"topics"`
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

	// Reemplazar {{device_id}} en títulos y topics
	config = replaceDeviceIDPlaceholders(config)

	return &config, nil
}

// replaceDeviceIDPlaceholders reemplaza {{device_id}} en strings
func replaceDeviceIDPlaceholders(config Config) Config {
	deviceID := config.DeviceID

	// Reemplazar en título de UI
	config.UI.Window.Title = strings.ReplaceAll(
		config.UI.Window.Title,
		"{{device_id}}",
		deviceID,
	)

	// Reemplazar en topics MQTT
	for key, topic := range config.MQTT.Topics {
		config.MQTT.Topics[key] = strings.ReplaceAll(
			topic,
			"{{device_id}}",
			deviceID,
		)
	}

	return config
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
					Latitude:  19.4326,
					Longitude: -99.1332,
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
			Enabled: false,
		},
		UI: UIConfig{
			Window: WindowConfig{
				Width:  1280,
				Height: 720,
				Title:  "Simulador Transporte",
			},
			Theme: "dark",
			FPS:   60,
		},
	}
}
