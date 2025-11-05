package main

import (
	"fmt"
	"log"

	"github.com/MarcosBrindi/transporte-simulator/internal/config"
	"github.com/MarcosBrindi/transporte-simulator/internal/eventbus"
	"github.com/MarcosBrindi/transporte-simulator/internal/scenario"
	"github.com/MarcosBrindi/transporte-simulator/internal/sensors"
	"github.com/MarcosBrindi/transporte-simulator/internal/statemanager"
	"github.com/MarcosBrindi/transporte-simulator/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	fmt.Println("=== SIMULADOR DE TRANSPORTE PÚBLICO ===")
	fmt.Println("FASE 4: UI con Ebiten")
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

	// Crear State Manager
	stateMgr := statemanager.NewStateManager(bus, *cfg)

	// Iniciar sensores y state manager
	gps.Start()
	mpu.Start()
	stateMgr.Start()

	// Goroutine para actualizar velocidad del MPU basada en GPS
	gpsChannel := bus.Subscribe(eventbus.EventGPS)
	go func() {
		for event := range gpsChannel {
			data := event.Data.(eventbus.GPSData)
			mpu.UpdateSpeed(data.Speed)
		}
	}()

	// Simular vehículo moviéndose (automático)
	go func() {
		gps.SetSpeed(30.0) // 30 km/h constante para demo
	}()

	// Crear juego Ebiten
	game := ui.NewGame(bus, cfg, route)

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
	fmt.Println("\nDeteniendo sistema...")
	game.Stop()
	gps.Stop()
	mpu.Stop()
	stateMgr.Stop()

	fmt.Println("¡Hasta luego!")
}
