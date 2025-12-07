package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/MarcosBrindi/transporte-simulator/internal/config"
	"github.com/MarcosBrindi/transporte-simulator/internal/eventbus"
	"github.com/MarcosBrindi/transporte-simulator/internal/mqtt"
	"github.com/MarcosBrindi/transporte-simulator/internal/scenario"
	"github.com/MarcosBrindi/transporte-simulator/internal/sensors"
	"github.com/MarcosBrindi/transporte-simulator/internal/simulator"
	"github.com/MarcosBrindi/transporte-simulator/internal/statemanager"
	"github.com/MarcosBrindi/transporte-simulator/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	// Definir flags
	headless := flag.Bool("headless", false, "Ejecutar en modo headless (sin UI)")
	instances := flag.Int("instances", 1, "Número de instancias a ejecutar (1-1000)")
	flag.Parse()

	fmt.Println("=== SIMULADOR DE TRANSPORTE PÚBLICO ===")
	fmt.Println("FASE Final")
	fmt.Println()

	// Cargar configuración
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		fmt.Printf("Error cargando config: %v\n", err)
		fmt.Println("Usando configuración por defecto")
		cfg = config.Default()
	}

	fmt.Printf("Device ID: %s\n", cfg.DeviceID)
	fmt.Println()

	// ========== Modo Headless ==========
	if *headless {
		fmt.Printf("🚀 Modo HEADLESS: Lanzando %d instancias\n", *instances)
		fmt.Println()
		simulator.RunHeadless(*instances, cfg)
		fmt.Println("\n✅ Simulación finalizada")
		return
	}

	// ========== Modo UI (original) ==========
	fmt.Println("🎮 Modo UI: Iniciando simulación con interfaz gráfica")
	fmt.Println()

	// Crear Event Bus
	bus := eventbus.NewEventBus()
	defer bus.Close()

	// Crear ruta usando coordenadas del config
	route := scenario.NewRouteFromCoordinates(
		cfg.Sensors.GPS.InitialPosition.Latitude,
		cfg.Sensors.GPS.InitialPosition.Longitude,
	)
	fmt.Printf("  %s\n", route)
	fmt.Println()

	// ========== Inicializar Publishers (MQTT y RabbitMQ) ==========
	var mqttPublisher *mqtt.Publisher
	var rabbitPublisher *mqtt.RabbitMQPublisher

	// MQTT Publisher
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

	// RabbitMQ Publisher (UI mode: su propia conexión)
	if cfg.RabbitMQ.Enabled {
		conn, err := mqtt.ConnectRabbitMQ(cfg.RabbitMQ)
		if err != nil {
			fmt.Printf("⚠️  [RabbitMQ] No se pudo conectar: %v\n", err)
			fmt.Println("ℹ️  [RabbitMQ] El sistema continuará sin RabbitMQ")
		} else {
			ch, err := conn.Channel()
			if err != nil {
				fmt.Printf("⚠️  [RabbitMQ] Error creando canal: %v\n", err)
			} else {
				rabbitPublisher = mqtt.NewRabbitMQPublisher(ch, cfg.RabbitMQ, cfg.DeviceID, bus)
				err := rabbitPublisher.Start()
				if err != nil {
					fmt.Printf("⚠️  [RabbitMQ] Error iniciando publisher: %v\n", err)
				}
			}
		}
	} else {
		fmt.Println("ℹ️  [RabbitMQ] Deshabilitado en configuración")
	}
	// ===============================================================

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
	game := ui.NewGame(bus, cfg, route, stateMgr, executor, gps, mpu, vl53l0x, camera)

	// Configurar ventana
	ebiten.SetWindowSize(cfg.UI.Window.Width, cfg.UI.Window.Height)
	ebiten.SetWindowTitle(cfg.UI.Window.Title)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	fmt.Println("Iniciando UI con Ebiten...")
	fmt.Println("Cierra la ventana para salir")
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

	// Detener Publishers
	if mqttPublisher != nil {
		mqttPublisher.Stop()
	}
	if rabbitPublisher != nil {
		rabbitPublisher.Stop()
	}

	fmt.Println("👋 ¡Hasta luego!")
}
