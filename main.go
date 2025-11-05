package main

import (
	"fmt"
	"log"
	"time"

	"github.com/MarcosBrindi/transporte-simulator/internal/config"
	"github.com/MarcosBrindi/transporte-simulator/internal/eventbus"
	"github.com/MarcosBrindi/transporte-simulator/internal/scenario"
	"github.com/MarcosBrindi/transporte-simulator/internal/sensors"
	"github.com/MarcosBrindi/transporte-simulator/internal/statemanager"
	"github.com/MarcosBrindi/transporte-simulator/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	fmt.Println("🚀 === SIMULADOR DE TRANSPORTE PÚBLICO ===")
	fmt.Println("📡 FASE 6: Cámara/YOLO + Passenger Tracker")
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
	camera.Start() // ← NUEVO
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

	// ← NUEVO: Goroutine para actualizar estado de puerta en cámara
	doorChannel := bus.Subscribe(eventbus.EventDoor)
	go func() {
		for event := range doorChannel {
			data := event.Data.(eventbus.DoorData)
			camera.UpdateDoorState(data.IsOpen)
		}
	}()

	// Simular vehículo con paradas
	go func() {
		for {
			// FASE 1: Detenido en parada (15 segundos)
			fmt.Println("\n🛑 [Simulación] Vehículo DETENIDO en parada")
			gps.SetSpeed(0.0)
			time.Sleep(15 * time.Second)

			// FASE 2: Arrancando
			fmt.Println("🚀 [Simulación] Arrancando (10 km/h)")
			gps.SetSpeed(10.0)
			time.Sleep(2 * time.Second)

			// FASE 3: Acelerando
			fmt.Println("⚡ [Simulación] Acelerando (30 km/h)")
			gps.SetSpeed(30.0)
			time.Sleep(10 * time.Second)

			// FASE 4: Velocidad crucero
			fmt.Println("🏎️ [Simulación] Velocidad crucero (50 km/h)")
			gps.SetSpeed(50.0)
			time.Sleep(15 * time.Second)

			// FASE 5: Frenando
			fmt.Println("🔽 [Simulación] Frenando (30 km/h)")
			gps.SetSpeed(30.0)
			time.Sleep(2 * time.Second)

			fmt.Println("🔽 [Simulación] Frenando más (10 km/h)")
			gps.SetSpeed(10.0)
			time.Sleep(2 * time.Second)

			fmt.Println("🛑 [Simulación] Deteniéndose")
			gps.SetSpeed(0.0)
			time.Sleep(3 * time.Second)
		}
	}()

	// Crear juego Ebiten
	game := ui.NewGame(bus, cfg, route, stateMgr)

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
	game.Stop()
	gps.Stop()
	mpu.Stop()
	vl53l0x.Stop()
	camera.Stop()
	stateMgr.Stop()

	fmt.Println(" ¡Hasta luego!")
}
