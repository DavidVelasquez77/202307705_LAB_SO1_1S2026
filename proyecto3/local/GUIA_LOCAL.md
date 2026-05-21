# Guia Local

## Ubicacion

Todos los comandos asumen que estas en la raiz del proyecto:

```bash
cd /home/vela/Documentos/SOPES1/proyecto3
```

## Arrancar todo

Levanta todo el entorno local:

```bash
docker compose -f local/docker-compose.yml up -d --build
```

Ver contenedores arriba:

```bash
docker compose -f local/docker-compose.yml ps
```

## Parar o reiniciar

Parar todo:

```bash
docker compose -f local/docker-compose.yml down
```

Parar todo y borrar datos persistidos de Valkey/Grafana:

```bash
docker compose -f local/docker-compose.yml down -v
```

Reiniciar todo limpio:

```bash
docker compose -f local/docker-compose.yml down -v
docker compose -f local/docker-compose.yml up -d --build
```

Rebuild de un servicio especifico:

```bash
docker compose -f local/docker-compose.yml up -d --build go-consumer
docker compose -f local/docker-compose.yml up -d --build go-dapr-subscriber
```

## Health checks

```bash
curl http://localhost:8080/health
curl http://localhost:8081/health
curl http://localhost:8082/health
curl http://localhost:8083/health
curl http://localhost:8084/health
```

Rutas:

- `8080`: rust-api
- `8081`: go-ingest
- `8082`: go-dapr-publisher
- `8083`: go-dapr-subscriber
- `8084`: go-stats-api

## Probar los dos flujos manualmente

Flujo gRPC:

```bash
curl -X POST http://localhost:8080/grpc-202307705 \
  -H "Content-Type: application/json" \
  -d '{"country":"CHN","warplanes_in_air":15,"warships_in_water":8,"timestamp":"2026-03-12T20:15:30Z"}'
```

Flujo Dapr:

```bash
curl -X POST http://localhost:8080/dapr-202307705 \
  -H "Content-Type: application/json" \
  -d '{"country":"ESP","warplanes_in_air":20,"warships_in_water":5,"timestamp":"2026-03-12T20:16:00Z"}'
```

## RabbitMQ

UI:

```text
http://localhost:15672
```

Credenciales por defecto:

- usuario: `guest`
- password: `guest`

Ver todas las colas:

```bash
curl -u guest:guest -s "http://localhost:15672/api/queues/%2F" | jq '[.[] | {name, messages, messages_ready, messages_unacknowledged}]'
```

Cola del flujo gRPC:

```bash
curl -u guest:guest -s "http://localhost:15672/api/queues/%2F/war_reports" | jq '{name, messages, messages_ready, messages_unacknowledged}'
```

Cola del flujo Dapr:

```bash
curl -u guest:guest -s "http://localhost:15672/api/queues/%2F/go-dapr-subscriber-war-reports" | jq '{name, messages, messages_ready, messages_unacknowledged}'
```

Monitorear ambas colas en vivo:

```bash
watch -n 1 'curl -u guest:guest -s "http://localhost:15672/api/queues/%2F" | jq "[.[] | {name, ready: .messages_ready, unacked: .messages_unacknowledged, total: .messages}]"'
```

## Valkey

Ver total de reportes guardados:

```bash
docker exec valkey valkey-cli GET stats:total_reports
```

Ver conteo por pais:

```bash
docker exec valkey valkey-cli HGETALL stats:reports_by_country
```

Ver ultimos reportes:

```bash
docker exec valkey valkey-cli LRANGE reports 0 4
```

Ver reportes por pais:

```bash
docker exec valkey valkey-cli LRANGE reports:country:CHN 0 4
```

Ver top de aviones por pais:

```bash
docker exec valkey valkey-cli ZREVRANGE leaderboard:warplanes_by_country 0 4 WITHSCORES
```

Ver top de barcos por pais:

```bash
docker exec valkey valkey-cli ZREVRANGE leaderboard:warships_by_country 0 4 WITHSCORES
```

Ver timeline por pais:

```bash
docker exec valkey valkey-cli XRANGE stream:timeline:CHN - +
```

Monitorear el total en vivo:

```bash
watch -n 1 'docker exec valkey valkey-cli GET stats:total_reports'
```

## Logs utiles

Ver como el flujo gRPC guarda en Valkey:

```bash
docker logs -f go-consumer
```

Ver como el flujo Dapr guarda en Valkey:

```bash
docker logs -f go-dapr-subscriber
```

Ver rust-api:

```bash
docker logs -f rust-api
```

Ver go-ingest:

```bash
docker logs -f go-ingest
```

Ver go-writer:

```bash
docker logs -f go-writer
```

Ver sidecar Dapr del subscriber:

```bash
docker logs -f go-dapr-subscriber-dapr
```

## Stats API

Resumen:

```bash
curl http://localhost:8084/stats/summary
```

Top aviones:

```bash
curl http://localhost:8084/stats/top-warplanes
```

Top barcos:

```bash
curl http://localhost:8084/stats/top-warships
```

Modas:

```bash
curl http://localhost:8084/stats/modes
```

Pais asignado:

```bash
curl http://localhost:8084/stats/assigned-country
```

Timeline:

```bash
curl http://localhost:8084/stats/timeline/CHN
```

## Grafana

Abrir en navegador:

```text
http://localhost:3000
```

Credenciales:

- usuario: `admin`
- password: `admin123`

Confirmar que responde:

```bash
curl -I http://localhost:3000
```

## Locust

Activar entorno virtual:

```bash
source .venv/bin/activate
```

Abrir la interfaz web de Locust:

```bash
locust -f test/locust/locustfile.py --host http://localhost:8080
```

Abrir en navegador:

```text
http://localhost:8089
```

Valores sugeridos para ver ambas colas:

- `Number of users`: `20`
- `Spawn rate`: `4`

Prueba headless corta:

```bash
locust -f test/locust/locustfile.py --host http://localhost:8080 --headless -u 5 -r 1 -t 20s --only-summary
```

## Verificacion recomendada en 4 terminales

Terminal 1:

```bash
docker logs -f go-consumer
```

Terminal 2:

```bash
docker logs -f go-dapr-subscriber
```

Terminal 3:

```bash
watch -n 1 'docker exec valkey valkey-cli GET stats:total_reports'
```

Terminal 4:

```bash
watch -n 1 'curl -u guest:guest -s "http://localhost:15672/api/queues/%2F" | jq "[.[] | {name, ready: .messages_ready, unacked: .messages_unacknowledged, total: .messages}]"'
```

## Configuracion local de delays

Los delays actuales estan en:

```bash
grep DELAY local/.env
```

Rebuild despues de cambiar delays:

```bash
docker compose -f local/docker-compose.yml up -d --build go-consumer go-dapr-subscriber
docker compose -f local/docker-compose.yml up -d --force-recreate go-dapr-subscriber-dapr
```
