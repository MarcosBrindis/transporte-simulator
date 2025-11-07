package main

import (
	"fmt"
	"log"

	"github.com/MarcosBrindi/transporte-simulator/internal/config"
	"github.com/MarcosBrindi/transporte-simulator/internal/eventbus"
	"github.com/MarcosBrindi/transporte-simulator/internal/mqtt"
	"github.com/MarcosBrindi/transporte-simulator/internal/scenario"
	"github.com/MarcosBrindi/transporte-simulator/internal/sensors"
	"github.com/MarcosBrindi/transporte-simulator/internal/statemanager"
	"github.com/MarcosBrindi/transporte-simulator/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	fmt.Println("🚀 === SIMULADOR DE TRANSPORTE PÚBLICO ===")
	fmt.Println("📡 FASE 8: MQTT Publisher")
	fmt.Println()

	// Cargar configuración
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		fmt.Printf("⚠️  Error cargando config: %v\n", err)
		fmt.Println("📝 Usando configuración por defecto")
		cfg = config.Default()
	}

	fmt.Printf("🆔 Device ID: %s\n", cfg.DeviceID)
	fmt.Println()

	// Crear Event Bus
	bus := eventbus.NewEventBus()
	defer bus.Close()

	// Crear ruta
	route := scenario.NewDefaultRoute()
	fmt.Printf("🗺️  %s\n", route)
	fmt.Println()

	// ========== NUEVO: Inicializar MQTT Publisher ==========
	var mqttPublisher *mqtt.Publisher
	if cfg.MQTT.Enabled {
		mqttPublisher = mqtt.NewPublisher(cfg.MQTT, cfg.DeviceID, bus)
		err := mqttPublisher.Start()
		if err != nil {
			fmt.Printf("⚠️  [MQTT] No se pudo conectar: %v\n", err)
			fmt.Println("ℹ️  [MQTT] El sistema continuará sin MQTT")
		}
	} else {
		fmt.Println("ℹ️  [MQTT] Deshabilitado en configuración")
	}
	// =======================================================

	// Crear sensores
	gps := sensors.NewGPSSimulator(bus, cfg.Sensors.GPS, route)
	mpu := sensors.NewMPU6050Simulator(bus, cfg.Sensors.MPU6050)
	vl53l0x := sensors.NewVL53L0XSimulator(bus, cfg.Sensors.VL53L0X)
	camera := sensors.NewCameraSimulator(bus, cfg.Sensors.Camera)

	// Crear State Manager
	stateMgr := statemanager.NewStateManager(bus, *cfg)

	// Iniciar sensores y state manager
	gps.Start()
	mpu.Start()
	vl53l0x.Start()
	camera.Start()
	stateMgr.Start()

	// Goroutine para actualizar velocidad del MPU basada en GPS
	gpsChannel := bus.Subscribe(eventbus.EventGPS)
	go func() {
		for event := range gpsChannel {
			data := event.Data.(eventbus.GPSData)
			mpu.UpdateSpeed(data.Speed)
		}
	}()

	// Goroutine para actualizar estado de VL53L0X según vehículo
	vehicleChannel := bus.Subscribe(eventbus.EventVehicle)
	go func() {
		for event := range vehicleChannel {
			data := event.Data.(eventbus.VehicleStateData)
			vl53l0x.UpdateVehicleState(data.IsStopped)
			camera.UpdateVehicleState(data.IsStopped)
		}
	}()

	// Goroutine para actualizar estado de puerta en cámara
	doorChannel := bus.Subscribe(eventbus.EventDoor)
	go func() {
		for event := range doorChannel {
			data := event.Data.(eventbus.DoorData)
			camera.UpdateDoorState(data.IsOpen)
		}
	}()

	// Cargar y ejecutar escenario
	scenarioToRun := scenario.GetParadaNormal()

	// Opción alternativa: Cargar desde YAML
	// scenarioToRun, err := scenario.LoadScenario("scenarios/parada_normal.yaml")
	// if err != nil {
	//     fmt.Printf("⚠️  Error: %v\n", err)
	//     scenarioToRun = scenario.GetParadaNormal()
	// }

	// Crear ejecutor de escenario
	executor := scenario.NewExecutor(scenarioToRun, gps, bus)
	executor.Start()

	// Crear juego Ebiten
	game := ui.NewGame(bus, cfg, route, stateMgr, executor)

	// Configurar ventana
	ebiten.SetWindowSize(cfg.UI.Window.Width, cfg.UI.Window.Height)
	ebiten.SetWindowTitle(cfg.UI.Window.Title)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	fmt.Println("🎮 Iniciando UI con Ebiten...")
	fmt.Println("⚠️  Cierra la ventana para salir")
	fmt.Println()

	// Ejecutar juego
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}

	// Cleanup
	fmt.Println("\n🛑 Deteniendo sistema...")
	executor.Stop()
	game.Stop()
	gps.Stop()
	mpu.Stop()
	vl53l0x.Stop()
	camera.Stop()
	stateMgr.Stop()

	// ========== NUEVO: Detener MQTT ==========
	if mqttPublisher != nil {
		mqttPublisher.Stop()
	}
	// ==========================================

	fmt.Println("¡Hasta luego!")
}
