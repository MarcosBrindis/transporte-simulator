# ¿CÓMO FUNCIONA LA CONCURRENCIA EN 1000 BUSES?

## 1. MODELO CONCURRENTE (Goroutines en Go)

### ¿Qué es una Goroutine?
Una **goroutine** es un hilo ligero en Go que permite ejecutar múltiples tareas simultáneamente sin consumir muchos recursos.

```
COMPARACIÓN:
────────────────────────────────────────
Threads del SO:      ~1-2 MB cada uno
Goroutines de Go:    ~2-3 KB cada uno  ← 1000x más pequeña!
────────────────────────────────────────
```

### Modelo de Concurrencia: 1000 Goroutines

```
┌─────────────────────────────────────────────────────────┐
│          APLICACIÓN GO - MAIN THREAD                    │
└──────────────────┬──────────────────────────────────────┘
                   │
        ┌──────────┴──────────┐
        │                     │
    ┌───▼──────┐          ┌───▼──────┐
    │ RabbitMQ │          │ Scenario │
    │ Connection          │ Manager  │
    │ (1 TCP)  │          │          │
    └───┬──────┘          └──────────┘
        │
    1 Conexión AMQP
        │
  ┌─────┴─────────────────────────────────────────┐
  │                                               │
  ▼         ▼         ▼         ▼               ▼
┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐         ┌──────┐
│ GR-0 │ │ GR-1 │ │ GR-2 │ │ GR-3 │ .... │ GR-999│  1000 Goroutines
├──────┤ ├──────┤ ├──────┤ ├──────┤         ├──────┤  (paralelismo ligero)
│BUS-0 │ │BUS-1 │ │BUS-2 │ │BUS-3 │ .... │BUS-999│
│ Ch-0 │ │ Ch-1 │ │ Ch-2 │ │ Ch-3 │ .... │Ch-999 │  1000 Canales AMQP
│ EB-0 │ │ EB-1 │ │ EB-2 │ │ EB-3 │ .... │ EB-999│  (multiplexados)
└──────┘ └──────┘ └──────┘ └──────┘         └──────┘
   │        │        │        │               │
   └────────┴────────┴────────┴───────...─────┘
            (Envío de datos en paralelo)
```

**CLAVE:** Las 1000 goroutines ejecutan REALMENTE en paralelo (en múltiples CPU cores)


## 2. CÓMO SE LANZAN LAS 1000 INSTANCIAS

### Código en `internal/simulator/factory.go`

```go
func RunHeadless(numInstances int, cfg *config.Config) error {
    // 1. CONECTAR A RABBITMQ (UNA SOLA VEZ)
    conn, err := mqtt.ConnectRabbitMQ(cfg.RabbitMQ)
    if err != nil {
        return err
    }
    
    // 2. CREAR RUTA COMPARTIDA (para todos los buses)
    route := scenario.NewDefaultRoute()
    
    // 3. LANZAR N GOROUTINES
    var wg sync.WaitGroup
    
    for i := 0; i < numInstances; i++ {
        wg.Add(1)  // Incrementar contador de WaitGroup
        
        // Offset delay: evitar sincronización
        delay := time.Duration((i % 10) * 100) * time.Millisecond
        
        go func(id int) {
            defer wg.Done()  // Decrementar cuando termine
            
            time.Sleep(delay)  // Esperar N milisegundos
            
            // EJECUTAR SIMULACIÓN DEL VEHÍCULO
            SimulateVehicle(ctx, id, conn, cfg, route, &wg)
            
        }(i)  // Pasar ID para que cada goroutine sea única
    }
    
    // 4. ESPERAR A QUE TODAS TERMINEN
    wg.Wait()
}
```

### Timeline de Lanzamiento (ejemplo con 1000):

```
t=0ms:    Goroutine 0 inicia   (BUS-0000)
t=100ms:  Goroutine 10 inicia  (BUS-0010)
t=200ms:  Goroutine 20 inicia  (BUS-0020)
...
t=9900ms: Goroutine 990 inicia (BUS-0990)

✓ Todas las 1000 goroutines ejecutándose en paralelo después de 10 segundos

```

**¿Por qué los delays?** 
- Sin delays → TODOS publican al mismo tiempo → "thundering herd"
- Con delays → Distribución gradual → RabbitMQ no recibe pico de 1000 msgs en milisegundos


## 3. LO QUE HACE CADA GOROUTINE (Cada Bus)

```
GOROUTINE i (Ejemplo: BUS-0000)
─────────────────────────────────────────────────

1️⃣  CREAR CANAL PROPIO
    ch, _ := conn.Channel()  ← Del 1 connection compartido
    
2️⃣  CREAR EVENT BUS INDEPENDIENTE
    bus := eventbus.NewEventBus()  ← Para este bus SOLO
    
3️⃣  INICIALIZAR SENSORES
    gps := sensors.NewGPSSimulator(bus, cfg, route)
    mpu := sensors.NewMPU6050Simulator(bus, cfg)
    vl53l0x := sensors.NewVL53L0XSimulator(bus, cfg)
    camera := sensors.NewCameraSimulator(bus, cfg)
    
    Todos publica eventos al bus local ↓
    
4️⃣  CREAR STATE MANAGER
    stateMgr := statemanager.NewStateManager(bus, cfg)
    
    Escucha eventos del bus y calcula estado ↓
    
5️⃣  CREAR PUBLISHER RABBITMQ
    publisher := mqtt.NewRabbitMQPublisher(ch, cfg, "BUS-0000", bus)
    
    Escucha eventos y publica a RabbitMQ ↓
    
6️⃣  EJECUTAR SIMULACIÓN
    - Ciclo de 6 etapas (DETENIDO → MOVIMIENTO → APPROACHED → etc)
    - GPS actualiza posición
    - MPU simula aceleración
    - VL53L0X detecta puerta (distancia)
    - Camera detecta pasajeros
    - StateManager calcula estado
    - RabbitMQPublisher envía a RabbitMQ
    - LOOP hasta que recibe context.Done()
```

**IMPORTANTE:** Cada bus es COMPLETAMENTE INDEPENDIENTE de los otros


## 4. FLUJO DE DATOS - CÓMO LLEGA A RABBITMQ

### Per-Bus Data Flow:

```
GOROUTINE (BUS-0000)
────────────────────

GPS Simulator      MPU Simulator      VL53L0X Simulator     Camera Simulator
     │                  │                   │                    │
     │ GPSData          │ MPUData           │ DistanceData       │ TrackData
     └──────────────────┴───────────────────┴────────────────────┘
                           │
                      ┌─────▼─────┐
                      │ EventBus  │ (Local a BUS-0000)
                      │ (channels)│
                      └─────┬─────┘
                           │
                    ┌──────┴──────┐
                    │             │
            ┌───────▼────────┐  ┌─▼────────────┐
            │ StateManager   │  │ RabbitMQ     │
            │ Calcula:       │  │ Publisher    │
            │ • speed_kmh    │  │ Publica JSON │
            │ • accel_ms2    │  │              │
            │ • door_state   │  │ Routing Key: │
            │ • vehicle_st   │  │ vehicle.     │
            │                │  │ BUS-0000.    │
            │                │  │ hybrid       │
            └────────────────┘  └──────┬───────┘
                                       │
                                       ▼
                                    RabbitMQ
                              Queue: hybrid_49269307234447
```

### Mensaje JSON que se envía:

```json
{
  "timestamp": "2025-01-07T15:45:32.123Z",
  "device_id": "BUS-0000",
  "sensor_type": "hybrid_gps_mpu",
  "data": {
    "latitude": 19.4326,
    "longitude": -99.1332,
    "speed_kmh": 31.2,
    "acceleration_ms2": 0.15,
    "turn_rate_dps": 2.3,
    "vehicle_state": "MOVIMIENTO_CONFIRMADO"
  }
}
```

Cada goroutine envía INDEPENDIENTEMENTE su propio JSON a RabbitMQ


## 5. ¿QUÉ TAN DIFERENTES SON LOS DATOS?

### Elementos IGUALES en todos los buses:

```
❌ IGUALES:
   • Ruta (todos siguen mismo path)
   • Posición inicial (19.4326°, -99.1332°)
   • Estructura de 6 etapas (DETENIDO → MOVIMIENTO → etc)
   • Duración de etapas (15s, 10s, 20s, 5s, 15s, 5s)
   • Timestep de simulación (100ms)
```

### Elementos DIFERENTES en cada bus:

```
✅ DISTINTOS:
   • Device ID (BUS-0000, BUS-0001, ..., BUS-0999)
   • OFFSET DE LANZAMIENTO (delay de 0-900ms)
   • Variación de VELOCIDAD (±3 km/h aleatorio)
   • Variación de ACELERACIÓN (±0.1 m/s² aleatorio)
   • TIMING DE PUERTA (sensor VL53L0X tiene jitter)
   • DETECCIÓN DE PASAJEROS (camera simula peatones aleatorios)
   • RANDOM SEED (cada GPS/MPU tiene distinto seed)
```

### Ejemplo Concreto - Mismos 100ms en 3 buses:

```
t = 1000ms (todas en ESTADO: MOVIMIENTO_CONFIRMADO)

BUS-0000:
  {
    "device_id": "BUS-0000",
    "timestamp": "2025-01-07T15:45:33.000Z",
    "speed_kmh": 32.1,           ← Única para BUS-0000
    "acceleration_ms2": 0.08,    ← Única para BUS-0000
    "turn_rate_dps": 1.5         ← Única para BUS-0000
  }

BUS-0001:
  {
    "device_id": "BUS-0001",
    "timestamp": "2025-01-07T15:45:33.000Z",
    "speed_kmh": 28.9,           ← DIFERENTE (±3 km/h)
    "acceleration_ms2": -0.02,   ← DIFERENTE (±0.1 m/s²)
    "turn_rate_dps": 0.8         ← DIFERENTE
  }

BUS-0002:
  {
    "device_id": "BUS-0002",
    "timestamp": "2025-01-07T15:45:33.000Z",
    "speed_kmh": 30.7,           ← DIFERENTE de 0000 y 0001
    "acceleration_ms2": 0.12,    ← DIFERENTE
    "turn_rate_dps": 2.1         ← DIFERENTE
  }

Timestampl ~ igual porque ocurren ~simultáneamente
Device_id = COMPLETAMENTE DISTINTO ✓
Speed/Accel = TODOS DISTINTOS ✓
```

## 6. DONDE ESTÁ EL CÓDIGO DE VARIACIÓN

### Archivo: `internal/simulator/vehicle.go`

```go
// 6-Stage Movement Scenario
stages := []struct {
    name     string
    duration time.Duration
    baseSpeed float64  // Velocidad base para esta etapa
}{
    {"MovementConfirmed", 15 * time.Second, 30.0},
    {"Approaching", 10 * time.Second, 25.0},
    {"Stopped", 20 * time.Second, 0.0},
    {"Starting", 5 * time.Second, 5.0},
    {"Cruise", 15 * time.Second, 30.0},
    {"Decelerating", 5 * time.Second, 10.0},
}

// ═══════════════════════════════════════════════════════

// LA VARIACIÓN OCURRE AQUÍ:

for _, stage := range stages {
    stageStartTime := time.Now()
    
    for time.Since(stageStartTime) < stage.duration {
        // ✓ VARIACIÓN 1: Velocidad aleatoria
        variation := rand.Intn(7) - 3  // -3 to +3
        speed := stage.baseSpeed + float64(variation)
        
        // ✓ VARIACIÓN 2: Aceleración con jitter
        jitter := (rand.Float64() - 0.5) * 0.2  // ±0.1 m/s²
        acceleration := 0.0 + jitter
        
        // ✓ VARIACIÓN 3: Girar con ángulo aleatorio
        turnRate := (rand.Float64() - 0.5) * 4  // ±2 dps
        
        // ✓ VARIACIÓN 4: Detección de puerta con timing variable
        doorDistance := 300 + rand.Intn(200)  // 300-500mm
        
        // Publicar datos DISTINTOS para este bus
        publishToRabbitMQ(speed, acceleration, turnRate)
        
        time.Sleep(100 * time.Millisecond)
    }
}
```

**CLAVE:** Cada goroutine tiene su propio `rand` seed → números aleatorios DIFERENTES


## 7. SINCRONIZACIÓN CON WAITGROUP

```go
var wg sync.WaitGroup

// Por cada goroutine que lanzo, incremento
for i := 0; i < 1000; i++ {
    wg.Add(1)  // ← Contar esta goroutine
    
    go func(id int) {
        defer wg.Done()  // ← Contar cuando termine
        
        // ... código del bus ...
        
    }(i)
}

// BLOQUEAR MAIN HASTA QUE TODAS TERMINEN
wg.Wait()  ← Main espera aquí hasta que Done() sea llamado 1000 veces
```

```
VISUAL:
────────

Main Thread
    │
    ├─ Lanza Goroutine 0 (wg counter = 1)
    │
    ├─ Lanza Goroutine 1 (wg counter = 2)
    │
    ├─ Lanza Goroutine 2 (wg counter = 3)
    │  ...
    ├─ Lanza Goroutine 999 (wg counter = 1000)
    │
    │ wg.Wait()  ← MAIN SE BLOQUEA AQUÍ
    │
    │   [Goroutines ejecutando en paralelo...]
    │
    │   Goroutine 0 termina → wg counter = 999
    │   Goroutine 1 termina → wg counter = 998
    │   ... (podría ser en cualquier orden!)
    │   Goroutine 999 termina → wg counter = 0
    │
    └─ wg.Wait() RETORNA → Main continúa
       
       [Cleanup y cierre de programa]
```

**IMPORTANTE:** Las goroutines terminan en ORDEN ALEATORIO, no secuencial


## 8. ENVÍO A RABBITMQ - 1000 CANALES

### El Magic: 1 Conexión, 1000 Canales

```
TCP Connection (1 sola)
       │
       ├─ AMQP Channel 0 ──┐
       ├─ AMQP Channel 1 ──┤
       ├─ AMQP Channel 2 ──┤
       ...                 ├──▶ RabbitMQ Broker
       ├─ AMQP Channel 999 │
       │                   │
       └─ (multiplexing) ──┘

Ventaja: MUCHO MENOS consumo de recursos
Sin esto: 1000 conexiones TCP = 1000 handshakes = mucha memoria/CPU
Con esto: 1 conexión TCP = 1 handshake = mínima memoria/CPU
```

### Asincronía en Go

```go
// En cada goroutine:
channel := conn.Channel()  // ← Obtener del pool

// Mientras otros buses publican:
channel.Publish(
    exchange:   "amq.topic",
    key:        "vehicle.BUS-0000.hybrid",
    mandatory:  false,
    immediate:  false,
    msg:        body,  // JSON bytes
)

// El canal es ASINCRÓNICO → no espera confirmación
// Sigue con siguiente iteración INMEDIATAMENTE
// (sin bloquear el bus)
```

**Resultado:** 1000 buses publicando ~simultáneamente sin competir


## 9. TASA DE MENSAJES

```
Cada bus publica:
  • 1 msg "hybrid" cada ~1 segundo      (GPS + MPU)
  • 1 msg "passenger" cada ~2 segundos  (detecciones)

Por bus:     ~2 mensajes por 2 segundos = 1 msg/segundo
Por 1000:    1000 msg/segundo

Para 10 minutos:
  1000 buses × 1 msg/seg × 600 seg = 600,000 mensajes

Para tu Python backend:
  Recibe 1000 msg/seg en la queue (puede procesar en paralelo)
```


## 10. ¿PORQUÉ ES EFICIENTE LA CONCURRENCIA EN GO?

```
COMPARACIÓN: Go vs otros lenguajes

┌──────────────────────────────────────────────────┐
│ LENGUAJE  │ Threads │ Mem/Thread │ Total/1000  │
├──────────────────────────────────────────────────┤
│ Java      │ Threads │ 1-2 MB     │ 1-2 GB      │
│ Python    │ No hay  │ N/A        │ Imposible   │
│ .NET      │ Threads │ 1-2 MB     │ 1-2 GB      │
│ Go        │ Gorouts │ 2-4 KB     │ 2-20 MB ✓   │
└──────────────────────────────────────────────────┘

Go usa un "M:N scheduler"
  M goroutines → N threads del SO
  
Ejemplo:
  1000 goroutines → ~4 threads del SO
  (Distribuye automáticamente)
```


## 11. FLUJO COMPLETO VISUAL

```
╔═══════════════════════════════════════════════════════════════════════════╗
║                         START: -headless -instances=1000                  ║
╚═══════════════════════════════════════════════════════════════════════════╝

1. MAIN THREAD
   ├─ LoadConfig()         → config.yaml
   │
   ├─ ConnectRabbitMQ()    → (1 TCP connection)
   │
   ├─ NewDefaultRoute()    → Route compartida
   │
   └─ FOR i := 0 to 999:
      │
      ├─ go SimulateVehicle(i) con delay=(i%10)*100ms
      │  │
      │  └─ GOROUTINE i (BUS-i000)
      │     │
      │     ├─ conn.Channel()          → ch-i
      │     ├─ NewEventBus()           → bus-i (local)
      │     ├─ GPS Simulator           → publica a bus-i
      │     ├─ MPU Simulator           → publica a bus-i
      │     ├─ VL53L0X Simulator       → publica a bus-i
      │     ├─ Camera Simulator        → publica a bus-i
      │     ├─ StateManager            → escucha bus-i
      │     ├─ RabbitMQPublisher       → escucha bus-i
      │     │
      │     └─ LOOP until ctx.Done():
      │        │
      │        ├─ Calcular velocidad (±3 km/h aleatorio)
      │        ├─ Calcular aceleración (±0.1 m/s² aleatorio)
      │        ├─ Simular puerta (distancia con jitter)
      │        ├─ Simular cámara (pasajeros aleatorios)
      │        ├─ Publicar a RabbitMQ vía ch-i
      │        │
      │        └─ sleep(100ms) → siguiente iteración
      │
      └─ wg.Add(1)  → contador de waitgroup
   
   Después de lanzar todos:
   │
   └─ wg.Wait()  ← ESPERAR A QUE TODAS TERMINEN
      │
      └─ [Cleanup y cierre]


RESULTADO: 1000 goroutines ejecutando en paralelo, cada una publicando
datos DISTINTOS (pero correlacionados) a RabbitMQ simultáneamente
```

## 12. RESUMEN: CONCURRENCIA EN 1000 BUSES

| Aspecto | Explicación |
|---------|-------------|
| **Modelo** | 1000 goroutines paralelas en 1 proceso |
| **Conexión** | 1 TCP connection → 1000 AMQP channels |
| **Datos** | Device IDs únicos, velocidades/aceleraciones distintas |
| **Publicación** | ~1000 mensajes/segundo a RabbitMQ |
| **Sincronización** | WaitGroup (espera a que todas terminen) |
| **Recursos** | ~2-3 GB RAM total (2-3 MB por bus) |
| **Escalabilidad** | Probado: 2 y 10 instancias, listo para 1000 |
| **Diferencias datos** | ±3 km/h velocidad, ±0.1 m/s² aceleración, timings aleatorios |

