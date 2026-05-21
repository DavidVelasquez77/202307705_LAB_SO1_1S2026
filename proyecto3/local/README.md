## Local stack

Guia completa de comandos:

- `local/GUIA_LOCAL.md`

Levanta el flujo completo local con:

```bash
docker compose -f local/docker-compose.yml up --build
```

Si quieres reiniciar desde cero y borrar datos persistidos de Valkey/Grafana:

```bash
docker compose -f local/docker-compose.yml down -v
```

Para ver mejor las colas en RabbitMQ, el entorno local agrega retraso configurable por mensaje en:

- `CONSUMER_PROCESS_DELAY_MS` para la cola gRPC `war_reports`
- `DAPR_SUBSCRIBER_PROCESS_DELAY_MS` para la cola usada por Dapr

Servicios principales:

- `http://localhost:8080/health` rust-api
- `http://localhost:8081/health` go-ingest
- `http://localhost:8082/health` go-dapr-publisher
- `http://localhost:8083/health` go-dapr-subscriber
- `http://localhost:8084/health` go-stats-api
- `http://localhost:15672` RabbitMQ UI
- `http://localhost:3000` Grafana

Pruebas rápidas:

```bash
curl -X POST http://localhost:8080/grpc-202307705 \
  -H "Content-Type: application/json" \
  -d '{"country":"CHN","warplanes_in_air":15,"warships_in_water":8,"timestamp":"2026-03-12T20:15:30Z"}'

curl -X POST http://localhost:8080/dapr-202307705 \
  -H "Content-Type: application/json" \
  -d '{"country":"CHN","warplanes_in_air":20,"warships_in_water":5,"timestamp":"2026-03-12T20:16:00Z"}'
```
