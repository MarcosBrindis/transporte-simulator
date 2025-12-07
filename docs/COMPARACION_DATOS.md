# COMPARACIÓN JSON: ¿Qué es Igual y Qué es Diferente?

## ESCENARIO: 3 Buses publicando en RabbitMQ en el MISMO segundo (t=1000ms)

### TIMESTAMMP SIMILAR (Muy cercanos):

```
t = 1000ms exacto

BUS-0000:
  "timestamp": "2025-01-07T15:45:33.000Z"  ← 1000ms

BUS-0001:
  "timestamp": "2025-01-07T15:45:33.005Z"  ← 1005ms (5ms más tarde)

BUS-0002:
  "timestamp": "2025-01-07T15:45:33.002Z"  ← 1002ms (2ms más tarde)

✓ SIMILARES: Todos publican casi simultáneamente
  Diferencia máxima: ~100ms por offset de lanzamiento
```

### DEVICE_ID - COMPLETAMENTE DISTINTO:

```
BUS-0000:
  "device_id": "BUS-0000"  ← Único identificador

BUS-0001:
  "device_id": "BUS-0001"  ← TOTALMENTE DIFERENTE

BUS-0002:
  "device_id": "BUS-0002"  ← TOTALMENTE DIFERENTE

✓ DISTINTOS: Cada bus tiene su propia identidad
  Rango: BUS-0000 a BUS-0999
```

### SENSOR_TYPE - IGUAL:

```
BUS-0000:
  "sensor_type": "hybrid_gps_mpu"  ← Tipo de sensor

BUS-0001:
  "sensor_type": "hybrid_gps_mpu"  ← IGUAL

BUS-0002:
  "sensor_type": "hybrid_gps_mpu"  ← IGUAL

✓ IGUAL: Todos publican mismo tipo de sensor
```

### SPEED_KMH - DISTINTO:

```
BUS-0000:
  "speed_kmh": 31.2  ← En etapa "Movimiento Confirmado" (30 km/h base)
                        31.2 = 30 + 1.2 (variación aleatorio)

BUS-0001:
  "speed_kmh": 28.9  ← DIFERENTE
                        28.9 = 30 + (-1.1) (variación negativa)

BUS-0002:
  "speed_kmh": 32.1  ← DIFERENTE
                        32.1 = 30 + 2.1 (variación positiva)

✓ DISTINTOS: Cada bus varía ±3 km/h aleatoriamente
  Razón: speedVariation = (rand.Float64() * 6) - 3
```

### ACCELERATION_MS2 - DISTINTO:

```
BUS-0000:
  "acceleration_ms2": 0.08  ← Aceleración en m/s²

BUS-0001:
  "acceleration_ms2": -0.05  ← NEGATIVA (desacelerando)

BUS-0002:
  "acceleration_ms2": 0.12  ← DIFERENTE

✓ DISTINTOS: Jitter aleatorio en aceleración
  Razón: accelJitter = rand.Float64() * 0.2
         aplicado como: (rand.Float64()*jitter - jitter/2)
         rango: -0.1 a +0.1 m/s²
```

### TURN_RATE_DPS - DISTINTO:

```
BUS-0000:
  "turn_rate_dps": 1.5  ← Tasa de giro en grados/segundo

BUS-0001:
  "turn_rate_dps": 0.8  ← DIFERENTE

BUS-0002:
  "turn_rate_dps": 2.1  ← DIFERENTE

✓ DISTINTOS: Cada bus gira a diferente velocidad
  Rango: -2 a +2 dps
```

### VEHICLE_STATE - IGUAL:

```
BUS-0000:
  "vehicle_state": "MOVIMIENTO_CONFIRMADO"  ← Estado

BUS-0001:
  "vehicle_state": "MOVIMIENTO_CONFIRMADO"  ← IGUAL (mismo tiempo)

BUS-0002:
  "vehicle_state": "MOVIMIENTO_CONFIRMADO"  ← IGUAL (mismo tiempo)

✓ IGUAL: Todos están en misma etapa (15s duración)
  En t=1000ms, todos en "Movimiento Confirmado" (etapa 0)
  Cambiarían a "Aproximándose" aproximadamente en t=15000ms
```

## MENSAJE JSON COMPLETO COMPARATIVO

### BUS-0000:

```json
{
  "timestamp": "2025-01-07T15:45:33.000Z",
  "device_id": "BUS-0000",
  "sensor_type": "hybrid_gps_mpu",
  "data": {
    "latitude": 19.4326,
    "longitude": -99.1332,
    "speed_kmh": 31.2,
    "acceleration_ms2": 0.08,
    "turn_rate_dps": 1.5,
    "vehicle_state": "MOVIMIENTO_CONFIRMADO"
  }
}
```

### BUS-0001:

```json
{
  "timestamp": "2025-01-07T15:45:33.005Z",
  "device_id": "BUS-0001",
  "sensor_type": "hybrid_gps_mpu",
  "data": {
    "latitude": 19.4326,
    "longitude": -99.1332,
    "speed_kmh": 28.9,
    "acceleration_ms2": -0.05,
    "turn_rate_dps": 0.8,
    "vehicle_state": "MOVIMIENTO_CONFIRMADO"
  }
}
```

### BUS-0002:

```json
{
  "timestamp": "2025-01-07T15:45:33.002Z",
  "device_id": "BUS-0002",
  "sensor_type": "hybrid_gps_mpu",
  "data": {
    "latitude": 19.4326,
    "longitude": -99.1332,
    "speed_kmh": 32.1,
    "acceleration_ms2": 0.12,
    "turn_rate_dps": 2.1,
    "vehicle_state": "MOVIMIENTO_CONFIRMADO"
  }
}
```

## DIFERENCIAS SINTETIZADAS

```
┌─────────────────────────────────────────────────────────────────┐
│ CAMPO              │ BUS-0000 │ BUS-0001 │ BUS-0002 │ ¿IGUAL?  │
├─────────────────────────────────────────────────────────────────┤
│ timestamp          │ 000Z     │ 005Z     │ 002Z     │ ~SÍ      │
│ device_id          │ BUS-0000 │ BUS-0001 │ BUS-0002 │ NO       │
│ sensor_type        │ hybrid   │ hybrid   │ hybrid   │ SÍ       │
│ latitude           │ 19.4326  │ 19.4326  │ 19.4326  │ SÍ*      │
│ longitude          │ -99.1332 │ -99.1332 │ -99.1332 │ SÍ*      │
│ speed_kmh          │ 31.2     │ 28.9     │ 32.1     │ NO       │
│ acceleration_ms2   │ 0.08     │ -0.05    │ 0.12     │ NO       │
│ turn_rate_dps      │ 1.5      │ 0.8      │ 2.1      │ NO       │
│ vehicle_state      │ MOV_CON  │ MOV_CON  │ MOV_CON  │ SÍ       │
└─────────────────────────────────────────────────────────────────┘

* En realidad, la posición CAMBIARÍA lentamente en cada bus porque
  cada uno tiene su propio GPS simulator, pero en t=1000ms son iguales
  porque acaban de iniciar y la ruta es compartida.
  
  Después de 5 segundos, algunas posiciones podrían divergir levemente
  por diferencias en speed_kmh (31.2 vs 28.9 km/h)
```

## DESGLOSE POR ETAPA DE SIMULACIÓN

### ETAPA 0: Movimiento Confirmado (0-15s)

```
BASE: 30 km/h durante 15 segundos

BUS-0000:  30 + 1.2  = 31.2 km/h
BUS-0001:  30 - 1.1  = 28.9 km/h  ← 2.3 km/h más lento
BUS-0002:  30 + 2.1  = 32.1 km/h  ← 3.2 km/h más rápido

En 15 segundos:
  BUS-0000 viaja: 31.2 * 15/3600 = 0.130 km
  BUS-0001 viaja: 28.9 * 15/3600 = 0.120 km  ← 10m menos
  BUS-0002 viaja: 32.1 * 15/3600 = 0.134 km  ← 4m más
  
Divergencia: Los buses se alejan ligeramente entre sí
```

### ETAPA 1: Aproximándose (15-25s)

```
BASE: 20 km/h durante 10 segundos

BUS-0000:  20 + 1.2  = 21.2 km/h
BUS-0001:  20 - 1.1  = 18.9 km/h  ← Más lento
BUS-0002:  20 + 2.1  = 22.1 km/h  ← Más rápido

(Misma variación se reutiliza: speedVariation calculada una sola vez)
```

### ETAPA 2: Detenido (25-45s)

```
BASE: 0 km/h durante 20 segundos

BUS-0000:  0 + 1.2  = 1.2 km/h   ← PEQUEÑO MOVIMIENTO (error/variación)
BUS-0001:  0 - 1.1  = -1.1 km/h  ← REVERSA? (no, se clampea a 0)
BUS-0002:  0 + 2.1  = 2.1 km/h   ← PEQUEÑO MOVIMIENTO

NOTA: El código probablemente clampea a 0 en esta etapa:
    if actualSpeed < 0 {
        actualSpeed = 0
    }
```

### ETAPA 3: Arrancando (45-50s)

```
BASE: 15 km/h durante 5 segundos

BUS-0000:  15 + 1.2  = 16.2 km/h
BUS-0001:  15 - 1.1  = 13.9 km/h
BUS-0002:  15 + 2.1  = 17.1 km/h

Nuevos números aleatorios para aceleración:
BUS-0000:  0.15 m/s² (acelerando)
BUS-0001:  -0.08 m/s² (desacelerando, pero de 13.9 a más)
BUS-0002:  0.18 m/s² (acelerando fuerte)
```

## ACUMULACIÓN EN 1000 BUSES

Para 1000 buses en la misma etapa (Movimiento Confirmado):

```
speedVariation de cada bus: aleatorio entre -3.0 y +3.0 km/h

DISTRIBUCIÓN (esperada):

-3.0: █ (1-2 buses)
-2.5: ███ (3-5 buses)
-2.0: █████ (5-10 buses)
-1.5: ███████ (10-20 buses)
-1.0: █████████ (20-40 buses)
-0.5: ██████████ (40-80 buses)
 0.0: ██████████ (80-100 buses)  ← PICO
+0.5: ██████████ (80-100 buses)  ← PICO
+1.0: █████████ (40-80 buses)
+1.5: ███████ (20-40 buses)
+2.0: █████ (10-20 buses)
+2.5: ███ (5-10 buses)
+3.0: █ (1-2 buses)

Resultado: Una distribución normal alrededor de 30 km/h
           Con velocidades desde 27 km/h hasta 33 km/h
           Todos DISTINTOS pero realistas
```

## ¿CÓMO AFECTA AL PYTHON BACKEND?

Tu Python backend recibe:

```python
# En t=1000ms

msg_bus_0000 = {
    "device_id": "BUS-0000",
    "speed_kmh": 31.2,
    "acceleration_ms2": 0.08,
    ...
}

msg_bus_0001 = {
    "device_id": "BUS-0001",
    "speed_kmh": 28.9,  ← DIFERENTE
    "acceleration_ms2": -0.05,  ← DIFERENTE
    ...
}

msg_bus_0002 = {
    "device_id": "BUS-0002",
    "speed_kmh": 32.1,  ← DIFERENTE
    "acceleration_ms2": 0.12,  ← DIFERENTE
    ...
}

# Para 1000 buses: 1000 valores distintos en cada campo
# Tu backend debe procesar los 1000 en paralelo (si usa multithreading)
# O procesar secuencialmente (recomendado para análisis)
```

## RESUMEN FINAL

| Aspecto | ¿Igual o Diferente? | Variación |
|---------|-------------------|-----------|
| Device ID | DIFERENTE | BUS-0000 ... BUS-0999 |
| Timestamp | ~IGUAL | ±100ms máximo |
| Velocidad | DIFERENTE | ±3 km/h |
| Aceleración | DIFERENTE | ±0.1 m/s² |
| Giro | DIFERENTE | ±2 dps |
| Estado | IGUAL* | Mismo timestep (15-20s) |
| Ruta | IGUAL | Todos siguen ruta1.geojson |
| Posición | IGUAL** | Mismos puntos (pero distinto tiempo) |

*En el mismo momento de publicación
**Inicialmente igual, divergen con velocidades distintas


## CONCLUSIÓN

**¿Qué tan diferentes son los datos de 1000 buses?**

- **Device IDs**: 100% únicos (BUS-0000 a BUS-0999)
- **Velocidades**: Completamente distintas (27-33 km/h)
- **Aceleraciones**: Completamente distintas (±0.1 m/s²)
- **Estados**: Iguales en mismo momento temporal
- **Posiciones**: Inicialmente iguales, divergen por velocidad

**Para tu Python backend**:

Recibirás 1000 streams de JSON, cada uno con:
- Device ID único
- Valores numéricos ligeramente diferentes (realismo)
- Timestamps muy cercanos (concurrencia)
- Estados sincronizados (simulación)

**Recomendación**: 

Procesa con el device_id como clave primaria y los valores
numéricos (speed, accel) como telemetría diferenciada por bus.

