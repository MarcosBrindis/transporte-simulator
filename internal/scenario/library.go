package scenario

// GetParadaNormal retorna el escenario "Parada Normal"
func GetParadaNormal() *Scenario {
	return &Scenario{
		Name:        "Parada Normal",
		Description: "Vehículo circula, se detiene en parada, suben pasajeros, continúa",
		Duration:    90,
		Steps: []ScenarioStep{
			// Inicio: Circulando
			{Time: 0, Action: ActionLog, Value: "🚌 Inicio del recorrido"},
			{Time: 0, Action: ActionSetSpeed, Value: 30.0},

			// Llegando a parada
			{Time: 10, Action: ActionLog, Value: "🛑 Aproximándose a parada"},
			{Time: 10, Action: ActionSetSpeed, Value: 20.0},
			{Time: 12, Action: ActionSetSpeed, Value: 10.0},

			// Detenerse
			{Time: 15, Action: ActionLog, Value: "🛑 Detenido en parada"},
			{Time: 15, Action: ActionSetSpeed, Value: 0.0},

			// Esperar apertura de puerta (simulación automática ~5s)
			{Time: 20, Action: ActionWaitDoorOpen},

			// Pasajeros subiendo (puerta abierta ~10s)
			{Time: 25, Action: ActionLog, Value: "👥 Pasajeros subiendo..."},

			// Esperar cierre de puerta
			{Time: 30, Action: ActionWaitDoorClose},

			// Arrancar
			{Time: 35, Action: ActionLog, Value: "🚀 Reanudando marcha"},
			{Time: 35, Action: ActionSetSpeed, Value: 10.0},
			{Time: 37, Action: ActionSetSpeed, Value: 20.0},
			{Time: 40, Action: ActionSetSpeed, Value: 30.0},

			// Velocidad crucero
			{Time: 45, Action: ActionLog, Value: "🏎️ Velocidad crucero"},
			{Time: 45, Action: ActionSetSpeed, Value: 50.0},

			// Circular
			{Time: 70, Action: ActionLog, Value: "🚌 Circulando..."},

			// Fin
			{Time: 85, Action: ActionLog, Value: "✅ Escenario completado"},
		},
	}
}

// GetParadaConSalidas retorna el escenario "Parada con Salidas"
func GetParadaConSalidas() *Scenario {
	return &Scenario{
		Name:        "Parada con Salidas",
		Description: "Vehículo se detiene, bajan y suben pasajeros",
		Duration:    100,
		Steps: []ScenarioStep{
			// Circulando con pasajeros a bordo
			{Time: 0, Action: ActionLog, Value: "🚌 Circulando con pasajeros"},
			{Time: 0, Action: ActionSetSpeed, Value: 40.0},

			// Aproximándose a parada
			{Time: 15, Action: ActionLog, Value: "🛑 Aproximándose a parada"},
			{Time: 15, Action: ActionSetSpeed, Value: 30.0},
			{Time: 17, Action: ActionSetSpeed, Value: 20.0},
			{Time: 19, Action: ActionSetSpeed, Value: 10.0},

			// Detenerse
			{Time: 22, Action: ActionLog, Value: "🛑 Detenido - bajada de pasajeros"},
			{Time: 22, Action: ActionSetSpeed, Value: 0.0},

			// Esperar apertura
			{Time: 27, Action: ActionWaitDoorOpen},

			// Pasajeros bajando y subiendo
			{Time: 32, Action: ActionLog, Value: "🚶 Pasajeros bajando..."},
			{Time: 40, Action: ActionLog, Value: "👥 Nuevos pasajeros subiendo..."},

			// Esperar cierre
			{Time: 50, Action: ActionWaitDoorClose},

			// Arrancar de nuevo
			{Time: 55, Action: ActionLog, Value: "🚀 Continuando recorrido"},
			{Time: 55, Action: ActionSetSpeed, Value: 15.0},
			{Time: 58, Action: ActionSetSpeed, Value: 30.0},
			{Time: 62, Action: ActionSetSpeed, Value: 45.0},

			// Final
			{Time: 90, Action: ActionLog, Value: "✅ Escenario completado"},
		},
	}
}

// GetCircuitoCompleto retorna un circuito con múltiples paradas
func GetCircuitoCompleto() *Scenario {
	return &Scenario{
		Name:        "Circuito Completo",
		Description: "Recorrido completo con 3 paradas",
		Duration:    180,
		Steps: []ScenarioStep{
			// PARADA 1
			{Time: 0, Action: ActionLog, Value: "🚌 === PARADA 1: Terminal Sur ==="},
			{Time: 0, Action: ActionSetSpeed, Value: 0.0},
			{Time: 5, Action: ActionWaitDoorOpen},
			{Time: 15, Action: ActionWaitDoorClose},
			{Time: 18, Action: ActionSetSpeed, Value: 30.0},

			// Tránsito 1→2
			{Time: 30, Action: ActionSetSpeed, Value: 50.0},

			// PARADA 2
			{Time: 50, Action: ActionLog, Value: "🚌 === PARADA 2: Centro Comercial ==="},
			{Time: 50, Action: ActionSetSpeed, Value: 10.0},
			{Time: 53, Action: ActionSetSpeed, Value: 0.0},
			{Time: 58, Action: ActionWaitDoorOpen},
			{Time: 68, Action: ActionWaitDoorClose},
			{Time: 72, Action: ActionSetSpeed, Value: 30.0},

			// Tránsito 2→3
			{Time: 85, Action: ActionSetSpeed, Value: 45.0},

			// PARADA 3
			{Time: 110, Action: ActionLog, Value: "🚌 === PARADA 3: Terminal Norte ==="},
			{Time: 110, Action: ActionSetSpeed, Value: 10.0},
			{Time: 113, Action: ActionSetSpeed, Value: 0.0},
			{Time: 118, Action: ActionWaitDoorOpen},
			{Time: 128, Action: ActionWaitDoorClose},

			// Fin
			{Time: 135, Action: ActionLog, Value: "✅ Circuito completado"},
			{Time: 135, Action: ActionSetSpeed, Value: 0.0},
		},
	}
}

// GetAllScenarios retorna todos los escenarios disponibles
func GetAllScenarios() map[string]*Scenario {
	return map[string]*Scenario{
		"parada_normal":      GetParadaNormal(),
		"parada_con_salidas": GetParadaConSalidas(),
		"circuito_completo":  GetCircuitoCompleto(),
	}
}

// GetScenarioByName retorna un escenario por nombre
func GetScenarioByName(name string) *Scenario {
	scenarios := GetAllScenarios()
	return scenarios[name]
}

// GetScenarioNames retorna los nombres de todos los escenarios
func GetScenarioNames() []string {
	return []string{
		"parada_normal",
		"parada_con_salidas",
		"circuito_completo",
	}
}
