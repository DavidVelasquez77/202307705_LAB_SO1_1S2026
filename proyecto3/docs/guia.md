# Guia de Exposicion CNCF

## Tema

`El Duelo Cloud-Native: gRPC vs Dapr en Sistemas Distribuidos`

## Objetivo de esta guia

Esta guia sirve como apoyo para exponer el proyecto durante aproximadamente una hora. Incluye:

- un guion hablado
- el orden recomendado de la presentacion
- que cosas mostrar en la demo
- ideas clave que debes repetir
- preguntas probables con respuestas generales

No es necesario decir cada palabra exactamente igual. La idea es que lo uses como base y hables con naturalidad.

---

## Idea central que debes transmitir

El mensaje principal de la charla es este:

> Construí el mismo sistema distribuido con la misma base funcional, pero comparando dos formas de integración cloud-native: una basada en gRPC y otra basada en Dapr.

Tambien puedes resumirlo asi:

> Mismo problema, dos estilos de integración.

Y tambien asi:

> La diferencia no está en el dato que entra, sino en cómo viaja entre servicios.

---

## Estructura recomendada de 1 hora

1. Introduccion: 5 minutos
2. Problema y objetivo: 7 minutos
3. Arquitectura general: 10 minutos
4. Duelo tecnico entre gRPC y Dapr: 13 minutos
5. Demo local: 15 minutos
6. Hallazgos y dificultades: 5 minutos
7. Conclusiones: 5 minutos

---

## 1. Introduccion

### Que decir

> Buenas tardes a todos.
> 
> El tema de mi charla es `El Duelo Cloud-Native: gRPC vs Dapr en Sistemas Distribuidos`.
> 
> En este proyecto construí una arquitectura distribuida para procesar reportes militares simulados en tiempo real, y el punto principal fue comparar dos enfoques de comunicación entre microservicios: gRPC y Dapr.
> 
> La idea no fue solamente hacer que el sistema funcionara, sino entender qué cambia cuando resolvemos el mismo problema con dos estilos distintos de integración cloud-native.
> 
> Ambos flujos reciben los mismos datos, terminan en RabbitMQ, almacenan en Valkey y muestran métricas en Grafana. Pero internamente la forma de integración cambia bastante, y ahí es donde está el duelo entre ambas tecnologías.

### Que mostrar

- titulo de la charla
- tu nombre
- nombre del proyecto

### Frases utiles

- `No busqué solo que funcionara; busqué compararlo.`
- `El objetivo no es declarar un ganador absoluto, sino entender los tradeoffs.`

---

## 2. Problema y objetivo

### Que decir

> El sistema recibe reportes militares simulados con cuatro campos: país, cantidad de aviones en el aire, cantidad de barcos en el mar y timestamp.
> 
> Cada uno de esos eventos debe entrar por una API, ser validado, procesado, encolado, almacenado y finalmente visualizado.
> 
> Entonces, más que una API aislada, esto realmente es un pipeline distribuido completo.
> 
> Lo interesante del proyecto es que implementé dos caminos internos para el mismo tipo de evento.
> 
> El primero usa gRPC como mecanismo directo de comunicación entre servicios.
> 
> El segundo usa Dapr con pub/sub sobre RabbitMQ, lo que genera un modelo mucho más desacoplado.

### Que mostrar

- el ejemplo del JSON de entrada
- la estructura de mensaje del proyecto

### Idea clave para remarcar

> El dato que entra es el mismo; lo que cambia es la forma en que se integra internamente.

---

## 3. Arquitectura general

### Que decir

> La arquitectura tiene varios componentes.
> 
> Primero, Locust genera tráfico y simula carga.
> 
> Después, una API en Rust recibe los requests externos.
> 
> Luego tengo servicios en Go que se encargan del procesamiento interno.
> 
> RabbitMQ funciona como broker de mensajería.
> 
> Valkey se encarga del almacenamiento y de mantener estructuras de datos agregadas.
> 
> Finalmente, Grafana consume esas métricas y las muestra en dashboards.
> 
> El punto donde realmente cambia el diseño es en el camino que siguen los datos después de entrar por Rust.

### Que mostrar

- el diagrama general de arquitectura

### Que decir mientras señalas el diagrama

> Aquí está el punto de entrada.
> 
> Aquí están los servicios de procesamiento.
> 
> Aquí está la parte de mensajería.
> 
> Aquí está la persistencia.
> 
> Y aquí está la visualización.

---

## 4. Explicacion del flujo gRPC

### Que decir

> En el flujo gRPC, el evento entra por Rust y se manda al servicio `go-ingest`.
> 
> `go-ingest` valida el payload, normaliza los datos y los transforma al formato esperado por protobuf.
> 
> Después de eso, `go-ingest` llama directamente por gRPC a `go-writer`.
> 
> `go-writer` recibe esa solicitud y la publica en RabbitMQ, en la cola `war_reports`.
> 
> Luego `go-consumer` consume desde RabbitMQ y guarda la información en Valkey.
> 
> En este modelo, la comunicación es muy explícita. Cada servicio sabe a quién le está hablando y qué contrato está usando.

### Que mostrar

- resaltar en el diagrama el camino Rust -> go-ingest -> go-writer -> RabbitMQ -> go-consumer -> Valkey

### Frases utiles

- `gRPC me da contratos claros y comunicación directa.`
- `Aquí hay menos abstracción intermedia y más control explícito.`

---

## 5. Explicacion del flujo Dapr

### Que decir

> En el flujo Dapr, el evento también entra por Rust, pero en lugar de mandarse al flujo gRPC, se reenvía a `go-dapr-publisher`.
> 
> Ese servicio usa el SDK de Dapr para publicar un evento en un componente pub/sub configurado sobre RabbitMQ.
> 
> Después, el sidecar de Dapr se encarga de entregar ese evento al `go-dapr-subscriber`.
> 
> Finalmente, `go-dapr-subscriber` recibe el evento y lo guarda en Valkey.
> 
> La diferencia importante aquí es que el productor no necesita conocer directamente al consumidor.
> 
> Simplemente publica el evento y Dapr se encarga del enrutamiento y la entrega.

### Que mostrar

- resaltar en el diagrama el camino Rust -> go-dapr-publisher -> Dapr -> RabbitMQ -> Dapr -> go-dapr-subscriber -> Valkey

### Frases utiles

- `Dapr me da desacoplamiento y un modelo natural para eventos.`
- `Aquí el productor no necesita conocer al consumidor final.`

---

## 6. Comparacion tecnica entre gRPC y Dapr

### Que decir

> Una vez vistos ambos flujos, ahora sí podemos compararlos.
> 
> gRPC me da contratos explícitos, control fino y comunicación directa.
> 
> Eso es muy útil cuando necesito eficiencia, bajo overhead y relaciones claras entre cliente y servidor.
> 
> Pero al mismo tiempo, eso genera más acoplamiento, porque un servicio conoce directamente al otro.
> 
> En cambio, Dapr me permite desacoplar servicios, trabajar con pub/sub más fácilmente y abstraer parte de la lógica de integración.
> 
> Eso es muy útil cuando mi sistema está más orientado a eventos o cuando quiero simplificar ciertos aspectos operativos.
> 
> Pero también agrega otra capa: sidecars, componentes y más piezas que tengo que entender y operar.

### Que mostrar

- una tabla comparativa con dos columnas

### Comparacion resumida para decir

> Si quiero llamadas directas y contratos estrictos, gRPC me parece excelente.
> 
> Si quiero integración desacoplada y un flujo orientado a eventos, Dapr me parece muy fuerte.

### Frases fuertes

- `gRPC me dio control; Dapr me dio desacoplamiento.`
- `No es una pelea de cuál gana siempre, sino de cuál conviene según el problema.`

---

## 7. Demo local

### Introduccion a la demo

> Para la demo final dejé el sistema completamente funcional en local.
> 
> La versión original del proyecto estaba pensada para nube y Kubernetes, pero esta versión local me permitió validar la lógica, la carga, la mensajería, la persistencia y la observabilidad sin depender de infraestructura externa.

### Orden recomendado de demo

1. Mostrar que el stack está arriba
2. Mostrar health checks
3. Probar ambos endpoints manualmente
4. Mostrar RabbitMQ
5. Mostrar logs de guardado en Valkey
6. Mostrar Valkey directamente
7. Mostrar Grafana
8. Mostrar Locust

---

### 7.1 Mostrar stack arriba

#### Que decir

> Aquí se puede ver que todos los contenedores principales están arriba y listos para la prueba.

#### Que mostrar

```bash
docker compose -f local/docker-compose.yml ps
```

#### Servicios a mencionar

- rust-api
- go-ingest
- go-writer
- go-consumer
- go-dapr-publisher
- go-dapr-subscriber
- rabbitmq
- valkey
- grafana

---

### 7.2 Health checks

#### Que decir

> Antes de meter carga, verifico que cada servicio responda correctamente.

#### Que mostrar

```bash
curl http://localhost:8080/health
curl http://localhost:8081/health
curl http://localhost:8082/health
curl http://localhost:8083/health
curl http://localhost:8084/health
```

---

### 7.3 Probar ambos endpoints manualmente

#### Que decir

> Ahora voy a enviar exactamente el mismo tipo de evento por ambas rutas para demostrar que la entrada es la misma, aunque internamente cada flujo siga caminos distintos.

#### Que mostrar

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

---

### 7.4 RabbitMQ

#### Que decir

> Aquí se observa la parte de mensajería. En RabbitMQ se puede ver cómo ambos flujos interactúan con colas distintas.

#### Que mostrar

```bash
curl -u guest:guest -s "http://localhost:15672/api/queues/%2F" | jq '[.[] | {name, messages, messages_ready, messages_unacknowledged}]'
```

#### Que explicar

> La cola `war_reports` corresponde al flujo gRPC.
> 
> La cola `go-dapr-subscriber-war-reports` corresponde al flujo Dapr.
> 
> En gRPC normalmente se observa más `ready`, mientras que en Dapr pueden verse más `unacked`, porque el sidecar toma el mensaje y confirma después de procesarlo.

---

### 7.5 Logs de guardado en Valkey

#### Que decir

> Esta parte es importante porque aquí ya no solo vemos mensajes encolados, sino evidencia de que ambos flujos terminan guardando información real en Valkey.

#### Que mostrar

Terminal para gRPC:

```bash
docker logs -f go-consumer
```

Terminal para Dapr:

```bash
docker logs -f go-dapr-subscriber
```

#### Que explicar

> `go-consumer` representa el flujo gRPC guardando en Valkey.
> 
> `go-dapr-subscriber` representa el flujo Dapr guardando en Valkey.

---

### 7.6 Valkey

#### Que decir

> Ahora voy a mostrar directamente el almacenamiento para comprobar que no solo se reciben eventos, sino que también se construyen métricas útiles.

#### Que mostrar

```bash
docker exec valkey valkey-cli GET stats:total_reports
docker exec valkey valkey-cli HGETALL stats:reports_by_country
docker exec valkey valkey-cli ZREVRANGE leaderboard:warplanes_by_country 0 4 WITHSCORES
docker exec valkey valkey-cli ZREVRANGE leaderboard:warships_by_country 0 4 WITHSCORES
```

#### Que decir

> Aquí se ven reportes acumulados, conteos por país y rankings. Eso confirma que el pipeline completo está funcionando.

---

### 7.7 Grafana

#### Que decir

> El valor final del sistema no es solo mover mensajes, sino convertirlos en observabilidad y métricas útiles.

#### Que mostrar

- dashboard local en Grafana
- maximos
- minimos
- top paises
- moda
- timeline
- pais asignado

#### Que decir

> Aquí podemos ver claramente la salida final del sistema: métricas agregadas, rankings y evolución temporal del país asignado.

---

### 7.8 Locust

#### Que decir

> Finalmente, una parte clave fue probar ambos flujos bajo carga usando Locust.
> 
> Esto me permitió comparar no solo funcionalidad, sino comportamiento operativo y estabilidad.

#### Que mostrar

- interfaz de Locust
- una corrida mezclando ambas rutas

#### Que decir

> Con Locust pude inyectar tráfico hacia ambas rutas al mismo tiempo y observar cómo reaccionaban RabbitMQ, Valkey y Grafana.

---

## 8. Hallazgos y dificultades

### Que decir

> Durante la implementación encontré varios problemas reales de sistemas distribuidos.
> 
> Por ejemplo, al inicio el flujo gRPC abría conexiones nuevas por cada request, lo cual generaba overhead y degradación bajo carga.
> 
> También el flujo Dapr dependía mucho de la sincronización correcta entre servicio y sidecar.
> 
> Además, el comportamiento visual en RabbitMQ no era idéntico entre ambos flujos, porque en Dapr el patrón entre `ready` y `unacked` es distinto.
> 
> Todo eso fue parte importante del aprendizaje, porque en sistemas distribuidos no basta con tener microservicios corriendo; también hay que entender cómo se comportan bajo presión y cómo se observan en producción.

### Frases utiles

- `Los problemas interesantes aparecieron bajo carga, no en el caso feliz.`
- `La observabilidad fue tan importante como la lógica.`

---

## 9. Resultados

### Que decir

> Después de optimizar el flujo y estabilizar el stack local, ambos caminos lograron procesar correctamente eventos, almacenarlos y reflejarlos en el dashboard.
> 
> Lo importante para mí no fue solo que respondieran a requests, sino que:
> - RabbitMQ recibiera mensajes
> - Valkey guardara correctamente
> - Grafana mostrara datos coherentes
> - y que Locust pudiera generar carga sin errores en escenarios razonables.

---

## 10. Conclusiones

### Que decir

> Como conclusión, mi experiencia fue que gRPC y Dapr no son enemigos absolutos.
> 
> gRPC me parece mejor cuando necesito control, contratos estrictos y comunicación directa.
> 
> Dapr me parece mejor cuando necesito desacoplamiento, pub/sub y una integración más orientada a eventos.
> 
> Entonces, la respuesta no es cuál gana siempre, sino cuál conviene para el tipo de sistema distribuido que estoy construyendo.
> 
> En resumen, este proyecto me permitió comparar dos estilos cloud-native sobre la misma arquitectura y entender sus diferencias no solo en teoría, sino también en comportamiento operativo, observabilidad y carga.

### Ultima frase recomendada

> Muchas gracias.

---

## Cosas que debes mostrar si o si

- diagrama general
- endpoints gRPC y Dapr
- colas de RabbitMQ
- logs de go-consumer y go-dapr-subscriber
- datos guardados en Valkey
- dashboard en Grafana
- Locust generando carga o resultados de Locust
- conclusiones comparativas

---

## Cosas que no debes sobreexplicar

- cada YAML de Kubernetes
- detalles de GCP si tu demo principal es local
- cada linea de código
- Zot o KubeVirt si no forman parte de la demo final
- detalles internos de cada plugin de Grafana

---

## Posibles preguntas y respuestas generales

### 1. Por que comparaste gRPC contra Dapr?

Respuesta sugerida:

> Porque ambos resuelven integración entre servicios, pero desde enfoques distintos. gRPC se enfoca en llamadas directas con contratos estrictos, mientras que Dapr facilita patrones más desacoplados como pub/sub. Me interesaba comparar no solo funcionalidad, sino también complejidad operativa y comportamiento bajo carga.

### 2. Cual de los dos es mejor?

Respuesta sugerida:

> No diría que uno gana siempre. Si necesito control y llamadas directas, prefiero gRPC. Si necesito desacoplar servicios y trabajar con eventos, Dapr me parece más conveniente. Depende del problema.

### 3. Por que usaste RabbitMQ en ambos flujos?

Respuesta sugerida:

> Porque quería mantener constante el broker de mensajería para que la comparación estuviera en la forma de integración entre servicios, no en cambiar toda la infraestructura al mismo tiempo.

### 4. Por que Rust en la API y Go en los servicios?

Respuesta sugerida:

> Porque el proyecto pedía explícitamente una API en Rust y servicios de procesamiento en Go. Además, eso también permitió ver una arquitectura políglota, donde diferentes componentes usan tecnologías distintas pero cooperan dentro del mismo sistema.

### 5. Que fue lo más difícil?

Respuesta sugerida:

> Lo más difícil fue estabilizar el comportamiento bajo carga y entender los problemas reales de integración. No solo era que el request pasara, sino que el pipeline completo se mantuviera bien con RabbitMQ, Dapr, Valkey y Grafana observando todo.

### 6. Por que terminaste usando demo local?

Respuesta sugerida:

> Porque la meta de esta charla es demostrar la comparación funcional y operativa entre ambos flujos. La versión local me permitió enseñar todo el pipeline activo sin depender de nube externa, manteniendo la lógica principal del sistema.

### 7. Como verificaste que ambos flujos sí funcionaban?

Respuesta sugerida:

> Verifiqué los endpoints, las colas en RabbitMQ, los logs de guardado en Valkey, los datos almacenados en Valkey, las métricas en Grafana y las pruebas de carga con Locust.

### 8. Que significa que en RabbitMQ aparezcan ready y unacked?

Respuesta sugerida:

> `ready` significa que el mensaje sigue esperando ser entregado, y `unacked` significa que ya fue entregado a un consumidor pero todavía no ha sido confirmado. En el flujo Dapr era normal ver más `unacked` por la forma en que el sidecar consume y luego confirma.

### 9. Se puede usar Dapr sin RabbitMQ?

Respuesta sugerida:

> Sí. Dapr abstrae el componente, así que podría trabajar con otros brokers compatibles. En este proyecto mantuve RabbitMQ porque era parte de la arquitectura y del objetivo comparativo.

### 10. Que aprendiste realmente del proyecto?

Respuesta sugerida:

> Aprendí que diseñar sistemas distribuidos no es solo conectar servicios. También implica entender contratos, desacoplamiento, mensajería, observabilidad, manejo de carga y comportamiento operativo.

---

## Frases cortas que ayudan mucho durante la charla

- `Mismo problema, dos estilos de integración.`
- `La entrada es la misma; el camino interno es lo que cambia.`
- `gRPC me dio control.`
- `Dapr me dio desacoplamiento.`
- `No solo probé funcionalidad; también probé comportamiento bajo carga.`
- `La observabilidad fue una parte central del proyecto.`
- `No busqué un ganador absoluto, sino entender los tradeoffs.`

---

## Cierre final sugerido

> En conclusión, este proyecto me permitió comparar dos formas reales de construir integración cloud-native dentro de una misma arquitectura distribuida. Más allá del rendimiento, la comparación me ayudó a entender diferencias en acoplamiento, operación, observabilidad y mantenimiento. Muchas gracias.
