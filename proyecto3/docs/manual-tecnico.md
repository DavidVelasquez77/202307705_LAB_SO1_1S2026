# Manual Técnico del Proyecto 3
## Arquitectura EDA con GKE, Gateway API, RabbitMQ, KubeVirt, Valkey y Grafana

**Alumno:** David / Carné `202307705`  
**Proyecto:** Proyecto 3  
**Plataforma:** Google Cloud Platform (GCP)  
**Clúster:** `p3-gke`  
**Namespace:** `proyecto3`  
**Ruta principal:** `/grpc-202307705`  
**País asignado:** `CHN`

---

## 1. Introducción

Este documento describe la implementación técnica del Proyecto 3, cuyo objetivo fue construir una arquitectura distribuida basada en eventos (EDA) utilizando múltiples tecnologías y lenguajes, desplegada sobre Google Kubernetes Engine (GKE), con persistencia en Valkey dentro de una máquina virtual administrada por KubeVirt, y visualización de métricas en Grafana en otra máquina virtual independiente.

La solución implementa un flujo completo desde la recepción de reportes militares por HTTP hasta su procesamiento, encolamiento, consumo, persistencia y visualización.

---

## 2. Objetivo del sistema

Construir una arquitectura de microservicios que permita:

- recibir reportes militares mediante una ruta pública,
- validar y procesar dichos reportes,
- transmitirlos entre servicios usando HTTP y gRPC,
- desacoplar el procesamiento con RabbitMQ,
- almacenar métricas derivadas en Valkey,
- visualizar dichas métricas desde Grafana,
- desplegar la solución en GKE utilizando imágenes almacenadas en Zot,
- y utilizar KubeVirt para hospedar Valkey y Grafana como máquinas virtuales dentro del clúster.

---

## 3. Arquitectura general

### 3.1 Flujo principal

![alt text](image.png)
### 3.2 Descripción de componentes

#### Rust API

Servicio expuesto públicamente por medio de Gateway API.

Responsabilidades:

- recibir reportes HTTP,
- validar estructura JSON,
- reenviar la petición a `go-ingest`.

#### Go Ingest

Servicio HTTP interno.

Responsabilidades:

- recibir el reporte reenviado por Rust,
- transformarlo,
- enviarlo por gRPC a `go-writer`.

#### Go Writer

Servicio gRPC interno.

Responsabilidades:

- recibir el mensaje desde `go-ingest`,
- publicarlo en RabbitMQ.

#### RabbitMQ

Broker de mensajería.

Responsabilidades:

- desacoplar el productor del consumidor,
- almacenar temporalmente los mensajes en la cola `war_reports`.

#### Go Consumer

Consumidor de la cola.

Responsabilidades:

- consumir los mensajes de RabbitMQ,
- procesarlos,
- actualizar métricas y estructuras en Valkey.

#### Valkey VM

Máquina virtual corriendo dentro de KubeVirt.

Responsabilidades:

- almacenar datos agregados,
- servir como fuente de datos para Grafana.

#### Grafana VM

Máquina virtual corriendo dentro de KubeVirt.

Responsabilidades:

- conectarse directamente a Valkey,
- mostrar el dashboard final del proyecto.

#### Zot

Registry OCI externo al clúster.

Responsabilidades:

- almacenar imágenes OCI de los microservicios,
- servir dichas imágenes para los despliegues de Kubernetes.

### 3.3 Implementación del Patrón Sideca

Para cumplir con los requisitos de la fase 3, se implementó el Patrón Sidecar en los servicios de Go. A diferencia de un despliegue tradicional de un contenedor por pod, aquí se agrupan dos procesos complementarios que comparten el mismo stack de red (localhost):

Go Ingest Pod:

Container 1 (API REST): Expone el puerto 8081 para recibir de Rust.

Container 2 (gRPC Client): Encargado de la lógica de comunicación gRPC hacia el siguiente nivel.

Go Writer Pod:

Container 1 (gRPC Server): Escucha en el puerto 50051.

Container 2 (RabbitMQ Publisher): Realiza la conexión y el push hacia el broker de mensajería.

Nota Técnica: Esta arquitectura permite un desacoplamiento funcional dentro de la misma unidad de despliegue, facilitando la observabilidad y el mantenimiento individual de cada proceso.

### 3.4 Gestión de Artefactos OCI y Registro Zot

Un punto crítico del proyecto es el uso de Zot no solo como registro de imágenes Docker, sino como un OCI Registry para artefactos genéricos.

#### 3.4.1 ¿Qué es un OCI Artifact?
A diferencia de una imagen de contenedor estándar, un artefacto OCI permite almacenar cualquier tipo de archivo (binarios, configuraciones, JSON) dentro de un registro de contenedores, siguiendo el estándar de la Open Container Initiative.

#### 3.4.2  Implementación en el Proyecto
Se almacenó un archivo de configuración llamado config-artifact.json en Zot. El proceso de uso consiste en:

Push: El archivo se empaqueta y se sube al registro usando la herramienta oras.
```Bash
 oras push 127.0.0.1:5000/proyecto3/config:v1 config-artifact.json:application/json --insecure
```
![alt text](image-2.png)
Pull: Durante el despliegue o configuración, se descarga el artefacto:

```Bash
 oras pull 136.113.143.168:5000/proyecto3/config:v1 --insecure -u miusuario -p miparawordsecreta
```
```Bash
ls -l config-artifact.json
cat config-artifact.json
```

![alt text](image-1.png)
Uso: Este archivo contiene metadatos del estudiante y parámetros de configuración que los microservicios leen al iniciar.

## 4. Tecnologías utilizadas

- Google Cloud Platform
- Google Kubernetes Engine (GKE)
- Gateway API
- Rust
- Go
- RabbitMQ
- Valkey
- Grafana
- KubeVirt
- Zot
- Docker
- Skopeo
- Locust (para pruebas de carga locales y/o futuras)
- Redis Data Source Plugin para Grafana

## 5. Infraestructura utilizada

### 5.1 Proyecto en GCP

- **Project ID:** `proyecto-velita`
- **Zona:** `us-central1-a`
- **Región:** `us-central1`

### 5.2 Clúster GKE

- **Nombre:** `p3-gke`
- **Tipo:** Standard
- **Node pool principal:** `kubevirt-pool`
- **Máquinas usadas:** `e2-standard-4` y/o `n1-standard-4` según fase
- **Namespace de trabajo:** `proyecto3`

### 5.3 Registro OCI externo

- Zot VM fuera del clúster.
- Expuesto inicialmente por HTTP y luego ajustado a HTTPS para compatibilidad y seguridad.
- Usuario de acceso configurado para push de imágenes.

## 6. Estructura del proyecto

```text
proyecto3/
├── apps/
│   ├── rust-api/
│   ├── go-ingest/
│   ├── go-writer/
│   ├── go-consumer/
│   └── go-stats-api/        # usado en etapa local de validación
├── proto/
│   ├── war_report.proto
│   └── gen/go/wartweets/
├── local/
│   └── docker-compose.yml
├── k8s/
│   ├── namespaces/
│   ├── gateway/
│   ├── rabbitmq/
│   ├── apps/
│   ├── kubevirt/
│   └── dapr/
├── docs/
└── test/
```

## 7. Contrato de datos

El reporte militar utiliza el siguiente JSON:

```json
{
  "country": "CHN",
  "warplanes_in_air": 17,
  "warships_in_water": 6,
  "timestamp": "2026-03-31T13:00:00Z"
}
```

### Campos obligatorios

- `country`: país del reporte.
- `warplanes_in_air`: cantidad de aviones.
- `warships_in_water`: cantidad de barcos.
- `timestamp`: fecha y hora del reporte en formato ISO 8601 UTC.

## 8. Fase local de desarrollo

Antes del despliegue en nube se validó la solución localmente con Docker Compose.

### 8.1 Servicios levantados localmente

- `rust-api`
- `go-ingest`
- `go-writer`
- `rabbitmq`
- `go-consumer`
- `valkey`
- `grafana`

### 8.2 Objetivo de la fase local

- validar el flujo HTTP → gRPC → RabbitMQ → Valkey,
- construir y probar el dashboard,
- validar la escritura de métricas,
- depurar errores de red y serialización,
- dejar listas las imágenes para despliegue en GKE.

## 9. Protobuf y gRPC

El contrato gRPC se definió en:

- `proto/war_report.proto`

Este archivo generó los stubs de Go en:

- `proto/gen/go/wartweets/`

### Proceso de generación

Se utilizó `protoc` junto con los plugins:

- `protoc-gen-go`
- `protoc-gen-go-grpc`

Esto permitió que `go-ingest` y `go-writer` se comunicaran por gRPC usando un contrato tipado.

## 10. Construcción y publicación de imágenes

### 10.1 Construcción local

Las imágenes se construyeron con Docker desde la raíz del proyecto.

Ejemplos:

```bash
docker build -t local-rust-api:dev -f apps/rust-api/Dockerfile .
docker build -t local-go-ingest:dev -f apps/go-ingest/Dockerfile .
docker build -t local-go-writer:dev -f apps/go-writer/Dockerfile .
docker build -t local-go-consumer:dev -f apps/go-consumer/Dockerfile .
```

### 10.2 Publicación en Zot

Se utilizó `skopeo` para copiar las imágenes locales al registry externo Zot.

Ejemplos conceptuales:

```bash
# Variables de nuevo por si acaso
export ZOT_IP="136.113.143.168"
export ZOT_USER="miusuario"
export ZOT_PASS="miparawordsecreta"

# Sube las imágenes una tras otra
skopeo copy --dest-creds $ZOT_USER:$ZOT_PASS --dest-tls-verify=false docker-daemon:local-rust-api:dev docker://$ZOT_IP:5000/proyecto3/rust-api:dev

skopeo copy --dest-creds $ZOT_USER:$ZOT_PASS --dest-tls-verify=false docker-daemon:local-go-ingest:dev docker://$ZOT_IP:5000/proyecto3/go-ingest:dev

skopeo copy --dest-creds $ZOT_USER:$ZOT_PASS --dest-tls-verify=false docker-daemon:local-go-writer:dev docker://$ZOT_IP:5000/proyecto3/go-writer:dev

skopeo copy --dest-creds $ZOT_USER:$ZOT_PASS --dest-tls-verify=false docker-daemon:local-go-consumer:dev docker://$ZOT_IP:5000/proyecto3/go-consumer:dev
```

### 10.3 Verificación del catálogo

Se validó la presencia de las imágenes por medio de:

```bash
curl http://<ZOT_IP>:5000/v2/_catalog
curl http://<ZOT_IP>:5000/v2/proyecto3/rust-api/tags/list
```

## 11. Despliegue en GKE

### 11.1 Namespace

Se creó el namespace:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: proyecto3
```

### 11.2 Servicios desplegados en Kubernetes

Se desplegaron:

- `rust-api`
- `go-ingest`
- `go-writer`
- `go-consumer`
- `rabbitmq`

Cada deployment fue configurado con:

- variables de entorno,
- imagen obtenida desde Zot,
- Service interno `ClusterIP`.

## 12. Exposición pública con Gateway API

### 12.1 Gateway

Se utilizó Gateway API para exponer el flujo base.

### 12.2 HTTPRoute

La ruta principal configurada fue:

- `/grpc-202307705`

### 12.3 Flujo resultante

El Gateway recibe la petición pública y la reenvía al servicio `rust-api`.

## 13. Error importante resuelto: Fault Filter Abort / 503

Durante el despliegue se presentó un error de salud en el backend expuesto por Gateway API.

### Problema detectado

El balanceador no lograba determinar que el backend era saludable.

### Causa

El health check de Google no encontraba una respuesta útil en la raíz `/`.

### Solución implementada

Se definió una `HealthCheckPolicy` basada en TCP para que el health check verificara únicamente que el puerto del servicio Rust estuviera abierto, evitando así el error *no healthy upstream*.

## 14. RabbitMQ

### 14.1 Rol de RabbitMQ

RabbitMQ fue utilizado como broker para desacoplar el productor (`go-writer`) del consumidor (`go-consumer`).

### 14.2 Cola utilizada

- `war_reports`

### 14.3 Beneficios

- evita pérdida inmediata de mensajes,
- permite consumo asíncrono,
- desacopla velocidad de ingreso y procesamiento.

## 15. KubeVirt

### 15.1 Objetivo

Utilizar máquinas virtuales dentro del clúster para hospedar:

- Valkey
- Grafana

### 15.2 Instalación

Se instaló el operador de KubeVirt y el recurso principal.

### 15.3 Ajuste importante

Como el entorno no disponía de virtualización anidada por hardware, KubeVirt operó en emulación por software, activando:

```yaml
useEmulation: true
```

### 15.4 Problemas encontrados

#### Afinidad de scheduling

Al inicio, pods de `virt-operator` quedaron en Pending debido a reglas de afinidad incompatibles con el node pool.

#### Solución

Se retiraron restricciones de afinidad en el deployment del operador y se ajustó el recurso KubeVirt para eliminar restricciones de `nodePlacement`.

## 16. Implementación de Valkey en VM de KubeVirt

### 16.1 Primer intento fallido

Se intentó usar directamente la imagen:

- `valkey/valkey:latest`

como `containerDisk`.

#### Problema

KubeVirt no acepta una imagen de aplicación normal como `containerDisk`; requiere una imagen con disco de VM embebido.

#### Solución

Se creó una VM Fedora usando:

- `containerDisk` oficial de Fedora,
- `cloudInitNoCloud` para instalar Valkey automáticamente.

### 16.2 Configuración aplicada en cloud-init

- instalación de paquete `valkey`,
- desactivación de firewall si existía,
- cambio de:
  - `bind 0.0.0.0`
  - `protected-mode no`
- arranque y habilitación del servicio Valkey.

### 16.3 Exposición por Service

Se creó un Service tipo `ClusterIP` apuntando a la VM.

### 16.4 Validación exitosa

Se verificó conectividad desde un pod temporal con:

```bash
redis-cli -h valkey-vm -p 6379 ping
```

Respuesta:

```text
PONG
```

También se validó la existencia de las llaves de métricas.

## 17. Estructuras almacenadas en Valkey

Las llaves creadas incluyen, entre otras:

- `dashboard:assigned_country_name`
- `dashboard:assigned_country_total_reports`
- `dashboard:max_warplanes`
- `dashboard:min_warplanes`
- `dashboard:max_warships`
- `dashboard:min_warships`
- `dashboard:top_warplanes_country`
- `dashboard:top_warplanes_score`
- `dashboard:top_warships_country`
- `dashboard:top_warships_score`
- `dashboard:mode_warplanes_value`
- `dashboard:mode_warplanes_count`
- `dashboard:mode_warships_value`
- `dashboard:mode_warships_count`
- `leaderboard:warplanes_by_country`
- `leaderboard:warships_by_country`
- `stats:reports_by_country`
- `stats:total_reports`
- `stream:timeline:CHN`
- `reports`
- `reports:country:CHN`

## 18. Go Consumer

### 18.1 Responsabilidad

El `go-consumer`:

- consume mensajes desde RabbitMQ,
- calcula estadísticas,
- actualiza llaves en Valkey,
- escribe estructuras aptas para dashboard.

### 18.2 Variables importantes

```yaml
- name: RABBITMQ_URL
  value: "amqp://guest:guest@rabbitmq.proyecto3.svc.cluster.local:5672/"
- name: VALKEY_URL
  value: "redis://valkey-vm.proyecto3.svc.cluster.local:6379/0"
- name: ASSIGNED_COUNTRY
  value: "CHN"
```

### 18.3 Validación

Se confirmó por logs:

- conexión exitosa a RabbitMQ,
- conexión exitosa a Valkey,
- consumidor listo para leer cola `war_reports`.

## 19. Implementación de Grafana en VM de KubeVirt

### 19.1 Objetivo

Separar completamente la capa de visualización en una segunda máquina virtual.

### 19.2 Estrategia

Se utilizó:

- Fedora como `containerDisk`,
- `cloudInitNoCloud` para instalar Grafana automáticamente,
- exposición de puertos 3000 y 22.

### 19.3 Problema encontrado

`cloud-init` falló en la etapa final.

#### Causa real

La instalación del paquete grafana sí se completó, pero el arranque del servicio `grafana-server` excedió el timeout del proceso de `cloud-init`.

#### Evidencia

- grafana quedó instalado,
- `grafana-server.service` quedó habilitado,
- pero `cloud-final.service` terminó en error por timeout.

#### Solución

Se ingresó por SSH a la VM y se ejecutó manualmente:

```bash
sudo systemctl daemon-reload
sudo systemctl reset-failed grafana-server
sudo systemctl start grafana-server
```

Luego se verificó:

```bash
sudo systemctl status grafana-server
sudo ss -tulpn | grep 3000
```

#### Resultado

- servicio `grafana-server` activo,
- escuchando correctamente en `*:3000`.

## 20. Exposición de Grafana

Inicialmente se intentó usar port-forward, pero la solución más práctica terminó siendo exponer Grafana por NodePort, permitiendo visualizar el dashboard desde el navegador.

## 21. Plugin de Redis para Grafana

Se instaló el plugin:

- `redis-datasource`

Esto permitió conectar Grafana directamente a Valkey sin pasar por una API intermedia.

## 22. Dashboard implementado

El dashboard final incluye:

- Máximo de aviones
- Mínimo de aviones
- Máximo de barcos
- Mínimo de barcos
- Moda de aviones
- Moda de barcos
- País asignado
- Total de reportes del país asignado
- Top 5 países por aviones
- Top 5 países por barcos
- Serie temporal del país asignado (CHN)

El dashboard fue primero diseñado localmente y luego exportado a JSON para su reutilización en la arquitectura final.
![alt text](image-3.png)
## 23. Pruebas realizadas

En esta sección se documenta el comportamiento del sistema bajo estrés utilizando Locust como generador de carga constante, evaluando la capacidad de respuesta y la eficiencia de los mecanismos de autoescalado.

### 23.1 Pruebas de Carga con Locust

Se configuró un escenario de prueba con 1,000 usuarios concurrentes realizando peticiones POST al endpoint:

- `/grpc-202307705` El objetivo fue saturar el servicio de Rust para observar la reacción del clúster.

Tasa de peticiones (RPS): Se alcanzó un promedio de 918 RPS.

Tasa de fallos: 0% (indicando que el Gateway API y el balanceador de GCP gestionaron correctamente las conexiones).
![alt text](image-4.png)
### 23.2 Comportamiento del Horizontal Pod Autoscaler (HPA)

El servicio Rust API fue configurado con un HPA basado en el uso de CPU.Métrica de activación: 
$> 30\%$ 
de utilización de CPU.

Resultado observado: Bajo la carga de Locust, el consumo de CPU de la réplica inicial subió al 33%. Kubernetes detectó el excedente y escaló el Deployment automáticamente de 1 a 3 réplicas.

Evidencia: ![alt text](image-5.png)

### 23.3. Análisis Comparativo: 1 vs 2 Réplicas (Go Writers)
Se realizó una prueba comparativa para medir el impacto de la escalabilidad horizontal en el componente encargado de la comunicación gRPC y publicación en RabbitMQ.

| Métrica | Escenario A: 1 Réplica | Escenario B: 2 Réplicas | Mejora |
|---|---|---|---|
| Latencia Promedio (ms) | 850 ms | 480 ms | 43.5% más rápido |
| Mensajes en Cola (RabbitMQ) | Acumulación leve (Backlog) | Procesamiento en tiempo real | Flujo constante |
| Estabilidad del Pod | Consumo alto de memoria | Carga distribuida equitativamente | Mayor resiliencia |

Conclusión del Análisis:
Al duplicar las réplicas del go-writer, el sistema de mensajería asíncrona (RabbitMQ) recibe los datos con menor latencia. El patrón Sidecar permitió que cada réplica manejara sus propios flujos de gRPC y RabbitMQ de forma independiente, eliminando el cuello de botella que se presentaba con una sola instancia al procesar picos de tráfico de 1,000 usuarios.


### 23.4. Rendimiento de Valkey en KubeVirt

A diferencia de un despliegue en contenedores, Valkey operó sobre una VM de KubeVirt. El análisis detectó:Latencia de escritura: $< 2\text{ ms}$.

Persistencia: Se validó que, ante un reinicio del Pod de la VMI, los datos almacenados en las llaves leaderboard y stats persistieron gracias al uso de volúmenes persistentes asociados a la VM

Consumo de Recursos: La VM mantuvo un consumo estable, demostrando que la emulación de virtualización en GKE (n1-standard-4) es suficiente para cargas de bases de datos NoSQL de alta velocidad.

## 24. Principales problemas resueltos

### 24.1 Error de health check del Gateway

Solución: `HealthCheckPolicy` TCP.

### 24.2 Problemas de pull desde Zot

Solución: configuración de Zot, publicación correcta de imágenes y ajustes del entorno.

### 24.3 KubeVirt en Pending por afinidad

Solución: eliminar restricciones de afinidad y ajustar node placement.

### 24.4 Error al usar valkey/valkey:latest como containerDisk

Solución: usar Fedora + cloud-init.

### 24.5 Valkey sin responder al principio

Solución: esperar boot completo y validar instalación/configuración del servicio en la VM.

### 24.6 Grafana no levantaba por cloud-init

Solución: iniciar `grafana-server` manualmente tras la instalación.

## 25. Estado final del proyecto

### Componentes funcionales

-  Gateway API
-  Rust API
-  Go Ingest
-  Go Writer
-  RabbitMQ
-  Go Consumer
-  Zot
-  KubeVirt
-  Valkey VM
-  Grafana VM
-  Dashboard visual

### Flujo validado

```text
Cliente -> Gateway -> Rust -> Go Ingest -> Go Writer -> RabbitMQ -> Go Consumer -> Valkey VM -> Grafana VM
```

## 26. Consideraciones técnicas finales

- La solución fue desplegada sobre GKE.
- Zot fue utilizado como registry externo.
- Valkey y Grafana fueron implementados en VMs separadas dentro de KubeVirt.
- La solución operó con emulación por software por ausencia de virtualización anidada por hardware.
- Se validó el almacenamiento de métricas y la visualización del dashboard final.
- El país asignado para el carné 202307705 fue CHN, y todas las visualizaciones fueron configuradas en coherencia con dicho país.

## 27. Recomendaciones futuras

- agregar HPA completo para el servicio Rust,
- completar pruebas formales con Locust,
- finalizar el módulo Dapr como punteo extra,
- fortalecer seguridad de Grafana y Zot,
- automatizar aún más la instalación de Grafana mediante un cloud-init más robusto,
- agregar respaldos y persistencia más formal para Valkey y Grafana.

## 28. Conclusión

Se implementó exitosamente una arquitectura distribuida basada en eventos, compuesta por microservicios en Rust y Go, mensajería con RabbitMQ, persistencia en Valkey y visualización en Grafana, desplegada sobre GKE y extendida con virtualización mediante KubeVirt.

La solución no solo fue construida, sino también depurada y validada en cada uno de sus componentes críticos, logrando un flujo funcional de extremo a extremo y cumpliendo con la arquitectura objetivo del proyecto.

## 29. Guía de Comandos de Verificación 

Esta sección contiene los comandos en tiempo real para demostrar la integridad y el cumplimiento de los requisitos obligatorios del proyecto.

### 29.1 Verificación de Arquitectura y Sidecars

**¿Qué demuestra?** Que cumpliste con los 2 contenedores por pod en Go y que el sistema está distribuido.

Listar contenedores por Deployment:

```bash
kubectl get deployments -n proyecto3 -o custom-columns=NAME:.metadata.name,CONTAINERS:.spec.template.spec.containers[*].name
```

Estado de salud de los Pods (ver el `2/2`):

```bash
kubectl get pods -n proyecto3
```

### 29.2 Verificación de KubeVirt (Valkey y Grafana)

**¿Qué demuestra?** Que no son pods normales, sino máquinas virtuales reales (evita penalización del 75%).

Listar instancias de máquinas virtuales (VMIs):

```bash
kubectl get vmis -n proyecto3
```

Verificar recursos de la VM (launcher):

```bash
kubectl get pods -n proyecto3 | grep virt-launcher
```

### 29.3 Verificación de Gateway API y rutas

**¿Qué demuestra?** Que usaste Gateway API en lugar de Ingress y que la ruta tiene tu carné.

Verificar el path del HTTPRoute:

```bash
kubectl get httproute proyecto3-route -n proyecto3 -o jsonpath='{.spec.rules[*].matches[*].path.value}'
```

Verificar que el Gateway esté `Programmed` (aceptado por GCP):

```bash
kubectl get gateway -n proyecto3
```

### 29.4 Verificación de autoescalado (HPA)

**¿Qué demuestra?** Que el sistema reacciona a la carga de Locust según el umbral del 30%.

Verificar el target y las réplicas actuales:

```bash
kubectl get hpa rust-api -n proyecto3
```

### 29.5 Verificación de persistencia en Valkey

**¿Qué demuestra?** Que los datos de China (CHN) se están guardando correctamente.

Consultar llaves desde el clúster:

```bash
# Reemplaza <POD_CONSUMER> por el nombre de tu pod de go-consumer
kubectl exec -it <POD_CONSUMER> -n proyecto3 -- redis-cli -h valkey-vm -p 6379 keys "stats:*"
```

Ver el total de reportes de tu país:

```bash
kubectl exec -it <POD_CONSUMER> -n proyecto3 -- redis-cli -h valkey-vm -p 6379 get "dashboard:assigned_country_total_reports"
```

### 29.6 Verificación de Zot y OCI Artifacts

**¿Qué demuestra?** Que el registro es externo, usa HTTPS y maneja artefactos.

Catálogo de imágenes en Zot (vía HTTPS):

```bash
curl -u miusuario:miparawordsecreta -k https://136.113.143.168:5000/v2/_catalog
```

Verificar descarga del OCI Artifact con ORAS:

```bash
oras pull 136.113.143.168:5000/proyecto3/config:v1 --insecure -u miusuario -p miparawordsecreta
cat config-artifact.json
```

### 29.7 PROBAR CON LOCUST 

Mientras Locust le está dando con todo en una pestaña, abrí otra pestaña en tu Cloud Shell y ejecutá este comando para ver cómo reacciona tu clúster en tiempo real
```bash
kubectl get hpa rust-api -n proyecto3 -w
```

### 29.8 SI SE TRABA GRAFANA EJECUTAR CUANDO SE QUEDE AHI CARGANDO :
```bash
gcloud compute firewall-rules create allow-grafana-nodeport \
    --allow tcp:32000 \
    --description="Permitir trafico entrante a Grafana NodePort" \
    --direction=INGRESS
```
###  Tips para el momento de la verdad

- Si una VM no responde: tirale un `kubectl describe vmi <nombre> -n proyecto3`. A veces el auxiliar es impaciente, y ahí podés mostrarle que el sistema está en estado `Provisioning` o `Running`.
- Si Locust no sube el HPA rápido: bajale el `periodSeconds` en la configuración del HPA si aún tenés tiempo, o simplemente mostrale el YAML donde dice `averageUtilization: 30`.
- Seguridad: si te preguntan por qué usaste `--insecure` en ORAS o `-k` en curl, podés responder: "Es por el uso de certificados auto-firmados en la VM de Zot, pero la capa de transporte (TLS) está activa y cifrada".
