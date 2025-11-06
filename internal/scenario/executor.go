package scenario

import (
	"fmt"
	"sync"
	"time"

	"github.com/MarcosBrindi/transporte-simulator/internal/eventbus"
)

// SpeedController es la interfaz para controlar la velocidad del vehículo
type SpeedController interface {
	SetSpeed(speed float64)
}

// Executor ejecuta escenarios
type Executor struct {
	scenario        *Scenario
	speedController SpeedController // ← Cambiado de *sensors.GPSSimulator a interfaz
	bus             *eventbus.EventBus

	// Control
	mu               sync.RWMutex
	running          bool
	paused           bool
	startTime        time.Time
	currentStepIndex int
}

// NewExecutor crea un nuevo ejecutor de escenarios
func NewExecutor(scenario *Scenario, speedController SpeedController, bus *eventbus.EventBus) *Executor {
	return &Executor{
		scenario:         scenario,
		speedController:  speedController, // ← Acepta cualquier tipo que implemente la interfaz
		bus:              bus,
		running:          false,
		paused:           false,
		currentStepIndex: 0,
	}
}

// Start inicia la ejecución del escenario
func (e *Executor) Start() {
	e.mu.Lock()
	e.running = true
	e.paused = false
	e.startTime = time.Now()
	e.currentStepIndex = 0
	e.mu.Unlock()

	fmt.Printf("🎬 [Executor] Iniciando escenario: %s\n", e.scenario.Name)
	fmt.Printf("📋 [Executor] %s\n", e.scenario.Description)
	fmt.Printf("⏱️  [Executor] Duración: %.0fs\n", e.scenario.GetDuration().Seconds())
	fmt.Println()

	go e.execute()
}

// Stop detiene la ejecución
func (e *Executor) Stop() {
	e.mu.Lock()
	e.running = false
	e.mu.Unlock()

	fmt.Println("🛑 [Executor] Escenario detenido")
}

// Pause pausa la ejecución
func (e *Executor) Pause() {
	e.mu.Lock()
	e.paused = true
	e.mu.Unlock()

	fmt.Println("⏸️  [Executor] Escenario pausado")
}

// Resume reanuda la ejecución
func (e *Executor) Resume() {
	e.mu.Lock()
	e.paused = false
	e.mu.Unlock()

	fmt.Println("▶️  [Executor] Escenario reanudado")
}

// IsRunning retorna si está corriendo
func (e *Executor) IsRunning() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.running
}

// execute ejecuta el escenario
func (e *Executor) execute() {
	for e.IsRunning() {
		e.mu.RLock()
		paused := e.paused
		currentStep := e.currentStepIndex
		e.mu.RUnlock()

		if paused {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Verificar si hay más pasos
		if currentStep >= len(e.scenario.Steps) {
			// Escenario completado
			fmt.Printf("✅ [Executor] Escenario '%s' completado\n", e.scenario.Name)
			e.Stop()
			break
		}

		// Obtener siguiente paso
		step := e.scenario.Steps[currentStep]

		// Esperar hasta el tiempo del paso
		elapsed := time.Since(e.startTime).Seconds()
		if elapsed < step.Time {
			sleepDuration := time.Duration((step.Time - elapsed) * float64(time.Second))
			time.Sleep(sleepDuration)
		}

		// Ejecutar paso
		e.executeStep(step)

		// Avanzar al siguiente paso
		e.mu.Lock()
		e.currentStepIndex++
		e.mu.Unlock()
	}
}

// executeStep ejecuta un paso individual
func (e *Executor) executeStep(step ScenarioStep) {
	elapsed := time.Since(e.startTime).Seconds()

	fmt.Printf("🎬 [Executor] [%.1fs] Acción: %s", elapsed, step.Action)
	if step.Value != nil {
		fmt.Printf(" (valor: %v)", step.Value)
	}
	fmt.Println()

	switch step.Action {
	case ActionSetSpeed:
		e.handleSetSpeed(step)

	case ActionWaitDoorOpen:
		e.handleWaitDoorOpen(step)

	case ActionWaitDoorClose:
		e.handleWaitDoorClose(step)

	case ActionWait:
		e.handleWait(step)

	case ActionLog:
		e.handleLog(step)

	case ActionPause:
		e.Pause()

	case ActionResume:
		e.Resume()

	default:
		fmt.Printf("⚠️  [Executor] Acción desconocida: %s\n", step.Action)
	}
}

// handleSetSpeed cambia la velocidad del GPS
func (e *Executor) handleSetSpeed(step ScenarioStep) {
	var speed float64

	// Convertir valor a float64
	switch v := step.Value.(type) {
	case float64:
		speed = v
	case int:
		speed = float64(v)
	default:
		fmt.Printf("⚠️  [Executor] Valor inválido para set_speed: %v\n", step.Value)
		return
	}

	e.speedController.SetSpeed(speed) // ← Usa la interfaz
	fmt.Printf("   🚗 Velocidad establecida: %.1f km/h\n", speed)
}

// handleWaitDoorOpen espera a que se abra la puerta
func (e *Executor) handleWaitDoorOpen(_ ScenarioStep) {
	fmt.Println("   🚪 Esperando apertura de puerta...")

	// Suscribirse a eventos de puerta
	doorChannel := e.bus.Subscribe(eventbus.EventDoor)

	// Esperar hasta que la puerta se abra
	timeout := time.After(30 * time.Second)
	for {
		select {
		case event, ok := <-doorChannel:
			if !ok {
				// Channel cerrado, sistema detenido
				fmt.Println("   ⚠️  Channel de puerta cerrado")
				return
			}

			// Verificar que el evento tenga datos
			if event.Data == nil {
				continue
			}

			// Type assertion segura
			data, ok := event.Data.(eventbus.DoorData)
			if !ok {
				fmt.Println("   ⚠️  Tipo de dato incorrecto en evento de puerta")
				continue
			}

			if data.IsOpen {
				fmt.Println("   ✅ Puerta abierta")
				return
			}
		case <-timeout:
			fmt.Println("   ⏰ Timeout esperando apertura de puerta")
			return
		}
	}
}

// handleWaitDoorClose espera a que se cierre la puerta
func (e *Executor) handleWaitDoorClose(_ ScenarioStep) {
	fmt.Println("   🚪 Esperando cierre de puerta...")

	doorChannel := e.bus.Subscribe(eventbus.EventDoor)

	timeout := time.After(30 * time.Second)
	for {
		select {
		case event, ok := <-doorChannel:
			if !ok {
				// Channel cerrado, sistema detenido
				fmt.Println("   ⚠️  Channel de puerta cerrado")
				return
			}

			// Verificar que el evento tenga datos
			if event.Data == nil {
				continue
			}

			// Type assertion segura
			data, ok := event.Data.(eventbus.DoorData)
			if !ok {
				fmt.Println("   ⚠️  Tipo de dato incorrecto en evento de puerta")
				continue
			}

			if !data.IsOpen {
				fmt.Println("   ✅ Puerta cerrada")
				return
			}
		case <-timeout:
			fmt.Println("   ⏰ Timeout esperando cierre de puerta")
			return
		}
	}
}

// handleWait espera N segundos
func (e *Executor) handleWait(step ScenarioStep) {
	var seconds float64

	switch v := step.Value.(type) {
	case float64:
		seconds = v
	case int:
		seconds = float64(v)
	default:
		fmt.Printf("⚠️  [Executor] Valor inválido para wait: %v\n", step.Value)
		return
	}

	fmt.Printf("   ⏱️  Esperando %.1f segundos...\n", seconds)
	time.Sleep(time.Duration(seconds * float64(time.Second)))
}

// handleLog imprime un mensaje
func (e *Executor) handleLog(step ScenarioStep) {
	message, ok := step.Value.(string)
	if !ok {
		fmt.Printf("⚠️  [Executor] Valor inválido para log: %v\n", step.Value)
		return
	}

	fmt.Printf("   📢 %s\n", message)
}

// GetProgress retorna el progreso del escenario (0.0 a 1.0)
func (e *Executor) GetProgress() float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.running {
		return 0.0
	}

	elapsed := time.Since(e.startTime).Seconds()
	total := e.scenario.GetDuration().Seconds()

	progress := elapsed / total
	if progress > 1.0 {
		return 1.0
	}
	return progress
}

// GetCurrentStep retorna el paso actual
func (e *Executor) GetCurrentStep() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.currentStepIndex
}
