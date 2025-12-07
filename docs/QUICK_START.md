# Quick Start Guide - Headless Mode

## Compilation
```bash
cd c:\Users\Marko\OneDrive\Documentos\IDS\Cuatrimestre_7\concurrencia\transporte-simulator
go build
```

## Common Commands

### 1. UI Mode (Original - Single Instance)
```bash
./transporte-simulator.exe
```
- Opens Ebiten window with visual simulator
- Single vehicle (BUS-49269307234447)
- Publishes to RabbitMQ (if enabled in config.yaml)
- Close window to exit

### 2. Headless Mode - Small Test (2 instances)
```bash
./transporte-simulator.exe -headless -instances=2
```
- No UI, pure CLI
- 2 concurrent buses (BUS-0000, BUS-0001)
- Rapid feedback to verify RabbitMQ is working
- Press Ctrl+C to stop

### 3. Headless Mode - Medium Test (10 instances)
```bash
./transporte-simulator.exe -headless -instances=10
```
- 10 buses publishing simultaneously
- Watch for message output in console
- Good for testing backend consumption

### 4. Headless Mode - Large Test (100 instances)
```bash
./transporte-simulator.exe -headless -instances=100
```
- 100 concurrent buses
- ~1000+ messages/minute
- Useful for load testing

### 5. Headless Mode - Full Load Test (1000 instances)
```bash
./transporte-simulator.exe -headless -instances=1000
```
- 1000 buses (BUS-0000 through BUS-0999)
- All publishing to same RabbitMQ queue
- ~2000+ messages/minute
- **WARNING**: Only run if RabbitMQ backend can handle it

## What Each Instance Does

1. **Connects** to RabbitMQ (34.233.205.241:5672)
2. **Creates own channel** on shared connection
3. **Initializes sensors**:
   - GPS: Simulates movement from fixed route
   - MPU6050: Simulates acceleration/rotation
   - VL53L0X: Detects door open/close
   - Camera: Detects passenger tracks
4. **Publishes messages** every ~2 seconds:
   - Hybrid GPS+MPU data to `vehicle.BUS-XXXX.hybrid`
   - Passenger events to `vehicle.BUS-XXXX.passenger`
5. **Runs 6-stage scenario**:
   - MovementConfirmed (15s) → Approaching (10s) → Stopped (20s) 
   - → Starting (5s) → Cruise (15s) → Decelerating (5s)
6. **Applies realism**:
   - Speed variation: 30 ± 3 km/h
   - Acceleration jitter: ±0.1 m/s²

## Monitoring

### Check RabbitMQ Queue
```bash
# Using RabbitMQ Management UI (http://34.233.205.241:15672)
# Queue name: hybrid_49269307234447
# Watch for increasing message count
```

### In Console Output
Watch for messages like:
```
📤 [RabbitMQ] Device ID: BUS-0000
🚌 [BUS-0000] Vehículo iniciado
✅ [RabbitMQ] Publicador iniciado
```

## Troubleshooting

### "No se pudo conectar" to RabbitMQ
```
⚠️  [RabbitMQ] No se pudo conectar: ...
```
- Check RabbitMQ is running at 34.233.205.241:5672
- Verify credentials in config.yaml (guest/guest)
- Check network connectivity

### Too slow / High memory usage
```bash
# Reduce instances
./transporte-simulator.exe -headless -instances=50
# Instead of 1000
```

### Stuck/Not responding
```bash
# Press Ctrl+C to force shutdown
# Then rebuild and try again
go build
```

## Configuration

Edit `config.yaml` to change:

```yaml
rabbitmq:
  enabled: true
  host: "34.233.205.241"
  port: 5672
  username: "guest"
  password: "guest"
  vhost: "/"
  exchange: "amq.topic"
```

## Expected Message Output

Each instance logs when it:
1. ✅ Initializes (all 4 sensors)
2. 📤 Starts publishing
3. 🚗 Changes vehicle state
4. 🚪 Opens/closes door
5. 👥 Detects passengers

Run with 2 instances to see clear output:
```bash
./transporte-simulator.exe -headless -instances=2
```

## Integration with Python Backend

Your Python code should listen to:
- **Exchange**: `amq.topic`
- **Routing Key**: `vehicle.#.hybrid` (all buses)
- **Queue**: `hybrid_49269307234447`

Example listener:
```python
import pika
import json

conn = pika.BlockingConnection(pika.ConnectionParameters('34.233.205.241'))
ch = conn.channel()
ch.exchange_declare(exchange='amq.topic', exchange_type='topic')
result = ch.queue_declare(queue='hybrid_49269307234447', durable=True)
ch.queue_bind(exchange='amq.topic', queue='hybrid_49269307234447', routing_key='vehicle.#.hybrid')

def on_message(ch, method, properties, body):
    msg = json.loads(body)
    print(f"Bus {msg['device_id']}: {msg['data']['speed_kmh']} km/h")

ch.basic_consume(queue='hybrid_49269307234447', on_message_callback=on_message)
ch.start_consuming()
```

## Quick Decision Tree

```
┌─ Want to see UI?
│  └─ YES  → ./transporte-simulator.exe
│
└─ Want headless mode?
   └─ Testing?  → ./transporte-simulator.exe -headless -instances=2
   └─ Load test? → ./transporte-simulator.exe -headless -instances=100
   └─ Full test? → ./transporte-simulator.exe -headless -instances=1000
```

---

**Status**: Phase 1 Complete ✅
**Next Phase**: Route variations, metrics, dashboard
