package eventbus

import (
	"sync"
)

// EventBus es el bus central de eventos usando Pub/Sub pattern
type EventBus struct {
	subscribers map[EventType][]chan Event
	mu          sync.RWMutex
}

// NewEventBus crea una nueva instancia del Event Bus
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[EventType][]chan Event),
	}
}

// Subscribe crea una suscripción a un tipo de evento específico
// Retorna un canal read-only para recibir eventos
func (eb *EventBus) Subscribe(eventType EventType) <-chan Event {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	// Crear canal con buffer para evitar bloqueos
	ch := make(chan Event, 10)

	// Agregar al map de suscriptores
	eb.subscribers[eventType] = append(eb.subscribers[eventType], ch)

	return ch
}

// Publish publica un evento a todos los suscriptores de ese tipo
func (eb *EventBus) Publish(event Event) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	// Obtener suscriptores para este tipo de evento
	if subs, ok := eb.subscribers[event.Type]; ok {
		// Enviar a todos los suscriptores
		for _, ch := range subs {
			// Non-blocking send (si el canal está lleno, descarta)
			select {
			case ch <- event:
				// Enviado exitosamente
			default:
				// Canal lleno, descartamos (evita deadlocks)
			}
		}
	}
}

// Close cierra todos los canales de suscriptores
func (eb *EventBus) Close() {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	for _, subs := range eb.subscribers {
		for _, ch := range subs {
			close(ch)
		}
	}

	// Limpiar map
	eb.subscribers = make(map[EventType][]chan Event)
}
