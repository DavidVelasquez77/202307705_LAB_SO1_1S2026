# Guía de Instalación
## Proyecto 2 - Sistemas Operativos 1

**Estudiante:** Josue David Velásquez Ixchop  
**Carnet:** 202307705

---

## 1. Introducción

Esta guía describe el proceso de instalación, configuración y verificación del sistema desarrollado para el Proyecto 2 de Sistemas Operativos 1.

El sistema está compuesto por los siguientes elementos:

- Un **módulo de kernel en C** que crea un archivo en `/proc`.
- Un **daemon en Go** que lee la información del módulo, la procesa y toma decisiones.
- Un entorno con **Docker**, donde se ejecutan contenedores de prueba.
- Un servicio **Valkey** para almacenamiento de métricas.
- Un servicio **Grafana** para visualización.
- Un **cronjob** que genera automáticamente contenedores de prueba.

---

## 2. Requisitos previos

Antes de instalar el sistema, se debe contar con lo siguiente:

- Sistema operativo **Ubuntu 24.04 LTS** o compatible.
- Acceso a terminal.
- Permisos de `sudo`.
- Conexión a internet para descarga de dependencias e imágenes Docker.

---

## 3. Dependencias necesarias

Instalar las dependencias del sistema con los siguientes comandos:

```bash
sudo apt update
sudo apt install -y build-essential gcc make git curl wget \
  linux-headers-$(uname -r) \
  golang-go \
  docker.io docker-compose-v2 \
  cron procps bc
```

---

## 4. Verificación de dependencias

Comprobar que las herramientas principales quedaron instaladas correctamente:

```bash
uname -r
go version
docker --version
docker compose version
ls -l /lib/modules/$(uname -r)/build
```

Se debe verificar que:

- Go responda con su versión.
- Docker responda con su versión.
- Docker Compose responda con su versión.
- Exista la ruta `/lib/modules/$(uname -r)/build`.

---

## 5. Activación de Docker

Habilitar y arrancar el servicio de Docker:

```bash
sudo systemctl enable --now docker
sudo systemctl status docker
```

Agregar el usuario actual al grupo `docker`:

```bash
sudo usermod -aG docker $USER
```

> Después de esto, se recomienda cerrar sesión e iniciar nuevamente.

Para validar Docker:

```bash
docker run hello-world
```

---

## 6. Estructura del proyecto

La estructura utilizada para el proyecto es la siguiente:

```
PROYECTO2/
├── cron/
│   ├── spawn_containers.sh
│   └── cron.log
├── daemon/
│   ├── daemon
│   ├── go.mod
│   ├── main.go
│   └── monitor_logs.jsonl
├── grafana/
│   └── docker-compose.yml
└── kernel/
    ├── continfo.c
    ├── Makefile
    ├── continfo.ko
    ├── load_module.sh
    └── unload_module.sh
```

---

## 7. Compilación del módulo de kernel

Entrar al directorio del módulo:

```bash
cd /home/vela/Documentos/SOPES1/PROYECTO2/kernel
```

Compilar el módulo:

```bash
make clean
make
```

Si la compilación es exitosa, se generará el archivo `continfo.ko`.

---

## 8. Carga del módulo de kernel

Para cargar el módulo:

```bash
cd /home/vela/Documentos/SOPES1/PROYECTO2/kernel
sudo insmod continfo.ko
```

Verificar mensajes del kernel:

```bash
sudo dmesg | tail -n 10
```

---

## 9. Verificación del archivo /proc

El módulo crea el archivo `/proc/continfo_pr2_so1_202307705`.

Para visualizar su contenido:

```bash
cat /proc/continfo_pr2_so1_202307705
```

Este archivo expone información del sistema y de los procesos.

---

## 10. Descarga del módulo de kernel

Para descargar el módulo:

```bash
cd /home/vela/Documentos/SOPES1/PROYECTO2/kernel
sudo rmmod continfo
```

Verificar nuevamente:

```bash
sudo dmesg | tail -n 10
```

---

## 11. Compilación del daemon en Go

Entrar al directorio del daemon:

```bash
cd /home/vela/Documentos/SOPES1/PROYECTO2/daemon
```

Inicializar módulo Go si es necesario:

```bash
go mod init proyecto2/daemon
```

Descargar dependencias:

```bash
go get github.com/redis/go-redis/v9
```

Compilar el daemon:

```bash
go build -o daemon
```

Esto generará el ejecutable `daemon`.

---

## 12. Levantamiento de Valkey y Grafana

Entrar al directorio de Grafana:

```bash
cd /home/vela/Documentos/SOPES1/PROYECTO2/grafana
```

Levantar los servicios:

```bash
docker compose up -d
```

Verificar los contenedores activos:

```bash
docker ps
docker compose ps
```

Se deben observar contenedores similares a:

- `grafana_so1`
- `valkey_so1`

---

## 13. Verificación de Valkey

Para comprobar que Valkey responde correctamente:

```bash
docker exec -it valkey_so1 valkey-cli ping
```

La respuesta esperada es:

```
PONG
```

---

## 14. Acceso a Grafana

Abrir en el navegador: [http://localhost:3000](http://localhost:3000)

Credenciales por defecto:

| Campo      | Valor   |
|------------|---------|
| Usuario    | `admin` |
| Contraseña | `admin` |

> En el primer ingreso, Grafana solicitará el cambio de contraseña.

---

## 15. Script de generación de contenedores

Entrar al directorio del cron:

```bash
cd /home/vela/Documentos/SOPES1/PROYECTO2/cron
```

Dar permisos de ejecución al script:

```bash
chmod +x spawn_containers.sh
```

Ejecutar manualmente el script:

```bash
./spawn_containers.sh
```

Este script crea los siguientes contenedores de prueba:

- `low_1`
- `low_2`
- `low_3`
- `highcpu_1`
- `highcpu_2`
- `highram_1`

Verificar contenedores activos:

```bash
docker ps
```

---

## 16. Verificación de consumo de contenedores

Para observar uso de CPU y RAM de los contenedores:

```bash
docker stats --no-stream
```

Se espera observar:

- `highcpu_1` y `highcpu_2` con uso alto de CPU.
- `highram_1` con uso alto de memoria RAM.
- `low_1`, `low_2`, `low_3` con uso bajo de recursos.

---

## 17. Configuración del cronjob

Para registrar el script como tarea programada:

```bash
crontab -e
```

Agregar la siguiente línea:

```
*/2 * * * * /home/vela/Documentos/SOPES1/PROYECTO2/cron/spawn_containers.sh >> /home/vela/Documentos/SOPES1/PROYECTO2/cron/cron.log 2>&1
```

Guardar y salir.

Verificar el cronjob:

```bash
crontab -l
```

---

## 18. Verificación del cronjob

Revisar el log generado por cron:

```bash
cat /home/vela/Documentos/SOPES1/PROYECTO2/cron/cron.log
```

Este archivo mostrará la ejecución periódica del script y la recreación de los contenedores de prueba.

---

## 19. Ejecución del daemon

Antes de ejecutar el daemon, debe estar cargado el módulo del kernel:

```bash
cd /home/vela/Documentos/SOPES1/PROYECTO2/kernel
sudo insmod continfo.ko
```

Luego ejecutar el daemon:

```bash
cd /home/vela/Documentos/SOPES1/PROYECTO2/daemon
./daemon
```

El daemon realiza las siguientes acciones:

- Lee el archivo `/proc/continfo_pr2_so1_202307705`.
- Analiza métricas del sistema y procesos.
- Detecta contenedores candidatos a eliminación.
- Guarda logs en archivo JSON.
- Guarda métricas en Valkey.

---

## 20. Verificación del log local

El daemon genera el archivo `monitor_logs.jsonl`. Para visualizarlo:

```bash
cd /home/vela/Documentos/SOPES1/PROYECTO2/daemon
cat monitor_logs.jsonl
```

---

## 21. Verificación del almacenamiento en Valkey

Para consultar los logs almacenados en Valkey:

```bash
docker exec -it valkey_so1 valkey-cli LRANGE monitor:logs 0 -1
```

Para consultar métricas individuales:

```bash
docker exec -it valkey_so1 valkey-cli GET monitor:ram_total
docker exec -it valkey_so1 valkey-cli GET monitor:ram_free
docker exec -it valkey_so1 valkey-cli GET monitor:ram_used
docker exec -it valkey_so1 valkey-cli GET monitor:last_update
docker exec -it valkey_so1 valkey-cli GET monitor:top_rss
docker exec -it valkey_so1 valkey-cli GET monitor:top_cpu
```

---

## 22. Detención del sistema

### Detener el daemon

En la terminal donde se ejecuta el daemon, presionar:

```
Ctrl + C
```

### Detener y eliminar contenedores de prueba

```bash
cd /home/vela/Documentos/SOPES1/PROYECTO2/cron
docker rm -f low_1 low_2 low_3 highcpu_1 highcpu_2 highram_1
```

### Descargar el módulo de kernel

```bash
cd /home/vela/Documentos/SOPES1/PROYECTO2/kernel
sudo rmmod continfo
```

### Detener Grafana y Valkey

```bash
cd /home/vela/Documentos/SOPES1/PROYECTO2/grafana
docker compose down
```

---

## 23. Pruebas de instalación exitosas

La instalación se considera exitosa si se cumplen las siguientes verificaciones:

- [x] El módulo de kernel compila correctamente.
- [x] El módulo crea el archivo `/proc/continfo_pr2_so1_202307705`.
- [x] El daemon compila y ejecuta correctamente.
- [x] Grafana y Valkey se levantan con Docker Compose.
- [x] El script `spawn_containers.sh` crea contenedores correctamente.
- [x] El cronjob ejecuta el script automáticamente.
- [x] El daemon genera `monitor_logs.jsonl`.
- [x] El daemon guarda métricas en Valkey.

---

## 24. Problemas comunes

### Error al cargar el módulo

Si `insmod` muestra error, verificar:

```bash
ls -l /lib/modules/$(uname -r)/build
```

### Error por módulo ya cargado

Si aparece `File exists`:

```bash
sudo rmmod continfo
sudo insmod continfo.ko
```

### Error de Docker

Verificar servicio:

```bash
sudo systemctl status docker
```

### El daemon no encuentra /proc

Verificar que el módulo esté cargado:

```bash
ls -l /proc/continfo_pr2_so1_202307705
```

### Valkey no responde

Probar:

```bash
docker exec -it valkey_so1 valkey-cli ping
```

### Cronjob no ejecuta

Verificar:

```bash
crontab -l
cat /home/vela/Documentos/SOPES1/PROYECTO2/cron/cron.log
```
