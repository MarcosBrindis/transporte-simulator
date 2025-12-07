package simulator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MarcosBrindi/transporte-simulator/internal/config"
	"github.com/MarcosBrindi/transporte-simulator/internal/scenario"
	amqp "github.com/rabbitmq/amqp091-go"
)

// RunHeadless ejecuta múltiples instancias de vehículos sin UI
func RunHeadless(numInstances int, cfg *config.Config) error {
	fmt.Println("\n🚀 === MODO HEADLESS (SIN UI) ===")
	fmt.Printf("📊 Instancias a ejecutar: %d\n", numInstances)
	fmt.Println()

	// Conectar a RabbitMQ UNA sola vez
	url := fmt.Sprintf("amqp://%s:%s@%s:%d%s",
		cfg.RabbitMQ.Username,
		cfg.RabbitMQ.Password,
		cfg.RabbitMQ.Host,
		cfg.RabbitMQ.Port,
		cfg.RabbitMQ.VHost,
	)

	fmt.Printf("📡 [Headless] Conectando a RabbitMQ: %s:%d\n", cfg.RabbitMQ.Host, cfg.RabbitMQ.Port)
	conn, err := amqp.Dial(url)
	if err != nil {
		return fmt.Errorf("error conectando a RabbitMQ: %w", err)
	}
	defer conn.Close()

	fmt.Println("✅ [Headless] Conexión a RabbitMQ establecida")
	fmt.Printf("🔑 [Headless] Exchange: %s\n", cfg.RabbitMQ.Exchange)
	fmt.Println()

	// Crear contexto para cancelación
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// WaitGroup para sincronizar goroutines
	var wg sync.WaitGroup

	// Crear ruta (compartida para todas)
	route := scenario.NewDefaultRoute()

	// Lanzar N vehículos
	fmt.Printf("🚌 Lanzando %d vehículos...\n", numInstances)
	for i := 0; i < numInstances; i++ {
		wg.Add(1)

		// Offset de inicio para evitar sincronización perfecta (cada 100ms)
		delayMs := (i % 10) * 100
		go func(id int, delayMs int) {
			time.Sleep(time.Duration(delayMs) * time.Millisecond)
			SimulateVehicle(ctx, id, conn, cfg, route, &wg)
		}(i, delayMs)

		// Log cada 100 instancias
		if (i+1)%100 == 0 {
			fmt.Printf("  ✓ %d vehículos lanzados\n", i+1)
		}
	}

	fmt.Printf("✅ Todos los %d vehículos están en ejecución\n", numInstances)
	fmt.Println("\n⏹️  Presiona Ctrl+C para detener...")
	fmt.Println()

	// Esperar a que terminen (presionar Ctrl+C)
	wg.Wait()

	fmt.Println("\n🛑 [Headless] Simulación finalizada")
	return nil
}

// RunWithUI ejecuta una instancia con interfaz gráfica
func RunWithUI(cfg *config.Config) error {
	// Esta función será llamada desde main.go
	// Contiene la lógica actual de la UI
	fmt.Println("\n🎮 === MODO UI (CON INTERFAZ GRÁFICA) ===")
	fmt.Println("Device ID: " + cfg.DeviceID)
	fmt.Println()

	// La lógica actual se mantiene en main.go
	return nil
}
