package simulator

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/MarcosBrindi/transporte-simulator/internal/config"
	"github.com/MarcosBrindi/transporte-simulator/internal/eventbus"
	"github.com/MarcosBrindi/transporte-simulator/internal/mqtt"
	"github.com/MarcosBrindi/transporte-simulator/internal/scenario"
	"github.com/MarcosBrindi/transporte-simulator/internal/sensors"
	"github.com/MarcosBrindi/transporte-simulator/internal/statemanager"
	amqp "github.com/rabbitmq/amqp091-go"
)

// VehicleSimulator simula un vehículo independiente
type VehicleSimulator struct {
	ID             string
	Bus            *eventbus.EventBus
	GPS            *sensors.GPSSimulator
	MPU            *sensors.MPU6050Simulator
	VL53L0X        *sensors.VL53L0XSimulator
	Camera         *sensors.CameraSimulator
	StateManager   *statemanager.StateManager
	Publisher      *mqtt.RabbitMQPublisher
	SpeedVariation float64
	AccelJitter    float64
}

// SimulateVehicle crea y ejecuta la simulación de un vehículo
func SimulateVehicle(
	ctx context.Context,
	id int,
	sharedConn *amqp.Connection,
	cfg *config.Config,
	route *scenario.Route,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	deviceID := fmt.Sprintf("BUS-%04d", id)

	// Crear canal propio para este vehículo
	ch, err := sharedConn.Channel()
	if err != nil {
		fmt.Printf("❌ [%s] Error creando canal: %v\n", deviceID, err)
		return
	}
	defer ch.Close()

	// Crear Event Bus local
	bus := eventbus.NewEventBus()
	defer bus.Close()

	// Crear sensores
	gps := sensors.NewGPSSimulator(bus, cfg.Sensors.GPS, route)
	mpu := sensors.NewMPU6050Simulator(bus, cfg.Sensors.MPU6050)
	vl53l0x := sensors.NewVL53L0XSimulator(bus, cfg.Sensors.VL53L0X)
	camera := sensors.NewCameraSimulator(bus, cfg.Sensors.Camera)

	// Crear State Manager
	stateMgr := statemanager.NewStateManager(bus, *cfg)

	// Crear Publisher con canal compartido
	publisher := mqtt.NewRabbitMQPublisher(ch, cfg.RabbitMQ, deviceID, bus)

	// Iniciar componentes
	gps.Start()
	mpu.Start()
	vl53l0x.Start()
	camera.Start()
	stateMgr.Start()
	err = publisher.Start()
	if err != nil {
		fmt.Printf("❌ [%s] Error iniciando publisher: %v\n", deviceID, err)
		return
	}

	fmt.Printf("🚌 [%s] Vehículo iniciado\n", deviceID)

	// Goroutine: actualizar velocidad del MPU basada en GPS
	gpsEvents := bus.Subscribe(eventbus.EventGPS)
	go func() {
		for event := range gpsEvents {
			data := event.Data.(eventbus.GPSData)
			mpu.UpdateSpeed(data.Speed)
		}
	}()

	// Goroutine: actualizar estado de VL53L0X según vehículo
	vehicleEvents := bus.Subscribe(eventbus.EventVehicle)
	go func() {
		for event := range vehicleEvents {
			data := event.Data.(eventbus.VehicleStateData)
			vl53l0x.UpdateVehicleState(data.IsStopped)
			camera.UpdateVehicleState(data.IsStopped)
		}
	}()

	// Goroutine: actualizar estado de puerta en cámara
	doorEvents := bus.Subscribe(eventbus.EventDoor)
	go func() {
		for event := range doorEvents {
			data := event.Data.(eventbus.DoorData)
			camera.UpdateDoorState(data.IsOpen)
		}
	}()

	// Simular patrón de conducción con variaciones
	baseSpeed := 30.0
	speedVariation := (rand.Float64() * 6) - 3 // ±3 km/h
	accelJitter := rand.Float64() * 0.2

	// Loop de simulación
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	stages := []struct {
		name     string
		duration time.Duration
		speed    float64
	}{
		{"Movimiento confirmado", 15 * time.Second, baseSpeed + speedVariation},
		{"Aproximándose", 10 * time.Second, 20.0 + speedVariation},
		{"Detenido", 20 * time.Second, 0.0},
		{"Arrancando", 5 * time.Second, 15.0 + speedVariation},
		{"Crucero", 15 * time.Second, baseSpeed + speedVariation},
		{"Desacelerando", 5 * time.Second, 10.0 + speedVariation},
	}

	currentStage := 0
	stageStart := time.Now()

	for {
		select {
		case <-ctx.Done():
			// Shutdown graceful
			fmt.Printf("🛑 [%s] Deteniendo vehículo\n", deviceID)
			publisher.Stop()
			gps.Stop()
			mpu.Stop()
			vl53l0x.Stop()
			camera.Stop()
			stateMgr.Stop()
			return

		case <-ticker.C:
			now := time.Now()
			elapsedInStage := now.Sub(stageStart)
			stage := stages[currentStage]

			// Cambiar de etapa si es necesario
			if elapsedInStage > stage.duration {
				currentStage = (currentStage + 1) % len(stages)
				stageStart = now
				stage = stages[currentStage]
			}

			// Aplicar variación
			actualSpeed := stage.speed + (rand.Float64()*accelJitter - accelJitter/2)
			if actualSpeed < 0 {
				actualSpeed = 0
			}

			gps.SetSpeed(actualSpeed)
		}
	}
}
