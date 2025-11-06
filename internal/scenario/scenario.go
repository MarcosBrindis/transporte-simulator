package scenario

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Scenario representa un escenario completo
type Scenario struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Duration    int            `yaml:"duration"` // Duración total en segundos
	Steps       []ScenarioStep `yaml:"steps"`
}

// ScenarioStep es un paso del escenario
type ScenarioStep struct {
	Time   float64     `yaml:"time"`   // Tiempo en segundos desde el inicio
	Action string      `yaml:"action"` // Tipo de acción
	Value  interface{} `yaml:"value"`  // Valor de la acción (puede ser float, string, etc.)
}

// ActionType define los tipos de acciones posibles
const (
	ActionSetSpeed      = "set_speed"       // Cambiar velocidad
	ActionWaitDoorOpen  = "wait_door_open"  // Esperar a que se abra la puerta
	ActionWaitDoorClose = "wait_door_close" // Esperar a que se cierre la puerta
	ActionWait          = "wait"            // Esperar N segundos
	ActionLog           = "log"             // Imprimir mensaje
	ActionPause         = "pause"           // Pausar simulación
	ActionResume        = "resume"          // Reanudar simulación
)

// LoadScenario carga un escenario desde un archivo YAML
func LoadScenario(filename string) (*Scenario, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error leyendo escenario: %w", err)
	}

	var scenario Scenario
	err = yaml.Unmarshal(data, &scenario)
	if err != nil {
		return nil, fmt.Errorf("error parseando YAML: %w", err)
	}

	// Validar escenario
	if err := scenario.Validate(); err != nil {
		return nil, fmt.Errorf("escenario inválido: %w", err)
	}

	return &scenario, nil
}

// Validate valida que el escenario sea correcto
func (s *Scenario) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("el escenario debe tener un nombre")
	}

	if len(s.Steps) == 0 {
		return fmt.Errorf("el escenario debe tener al menos un paso")
	}

	// Verificar que los tiempos estén ordenados
	lastTime := -1.0
	for i, step := range s.Steps {
		if step.Time < 0 {
			return fmt.Errorf("paso %d: el tiempo no puede ser negativo", i)
		}
		if step.Time < lastTime {
			return fmt.Errorf("paso %d: los pasos deben estar ordenados por tiempo", i)
		}
		lastTime = step.Time

		// Validar acción
		if !isValidAction(step.Action) {
			return fmt.Errorf("paso %d: acción '%s' no válida", i, step.Action)
		}
	}

	return nil
}

// isValidAction verifica si una acción es válida
func isValidAction(action string) bool {
	validActions := []string{
		ActionSetSpeed,
		ActionWaitDoorOpen,
		ActionWaitDoorClose,
		ActionWait,
		ActionLog,
		ActionPause,
		ActionResume,
	}

	for _, valid := range validActions {
		if action == valid {
			return true
		}
	}
	return false
}

// GetDuration retorna la duración total del escenario
func (s *Scenario) GetDuration() time.Duration {
	if s.Duration > 0 {
		return time.Duration(s.Duration) * time.Second
	}

	// Si no está especificado, usar el tiempo del último paso + 5 segundos
	if len(s.Steps) > 0 {
		lastTime := s.Steps[len(s.Steps)-1].Time
		return time.Duration(lastTime+5) * time.Second
	}

	return 60 * time.Second // Por defecto 60 segundos
}

// String implementa fmt.Stringer
func (s *Scenario) String() string {
	return fmt.Sprintf("Escenario: %s (%d pasos, %.0fs)", s.Name, len(s.Steps), s.GetDuration().Seconds())
}
