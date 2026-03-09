# Manual de Usuario  
## Proyecto 2 - Sistemas Operativos 1

**Estudiante:** Josue David Velásquez Ixchop  
**Carnet:** 202307705  

---

## 1. Introducción

El presente manual de usuario describe el funcionamiento general del sistema desarrollado para el Proyecto 2 de Sistemas Operativos 1.

El sistema permite monitorear recursos del sistema operativo Linux mediante un módulo de kernel, analizar métricas con un daemon escrito en Go, generar carga de prueba con contenedores Docker y almacenar resultados en Valkey.

Este manual está orientado al usuario que necesita ejecutar, supervisar y detener el sistema correctamente.

---

## 2. Objetivo del sistema

El sistema tiene como propósito:

- Obtener información del sistema y procesos desde el kernel.
- Exponer esa información mediante un archivo virtual en `/proc`.
- Leer y analizar dichas métricas desde user space.
- Detectar contenedores candidatos a eliminación según uso de CPU o memoria.
- Generar carga de prueba de forma automática mediante cron.
- Registrar resultados en archivo local y en Valkey.
- Utilizar Grafana como plataforma de visualización.

---

## 3. Requisitos para el uso del sistema

Para utilizar correctamente el sistema se necesita:

- Ubuntu 24.04 LTS o sistema Linux compatible.
- Tener instaladas las dependencias del proyecto.
- Tener Docker activo.
- Tener compilado el módulo del kernel.
- Tener compilado el daemon de Go.
- Tener levantados los servicios de Valkey y Grafana.

---

## 4. Componentes principales del sistema

El sistema está compuesto por:

- **Módulo de kernel**: genera el archivo `/proc/continfo_pr2_so1_202307705`.
- **Daemon en Go**: analiza métricas del sistema y contenedores.
- **Script de carga**: crea contenedores de prueba.
- **Cronjob**: ejecuta automáticamente el script de contenedores.
- **Valkey**: almacena métricas y logs.
- **Grafana**: se utiliza para visualización.

---

## 5. Inicio del sistema

Para iniciar correctamente el sistema, se recomienda seguir este orden:

1. Levantar Valkey y Grafana.
2. Cargar el módulo del kernel.
3. Ejecutar el script de contenedores de prueba.
4. Ejecutar el daemon.

---

## 6. Levantar Valkey y Grafana

Entrar al directorio correspondiente:

```bash
cd /home/vela/Documentos/SOPES1/PROYECTO2/grafana

Levantar los servicios:

docker compose up -d
Verificar:

docker ps
Se deben observar contenedores similares a:

grafana_so1

valkey_so1

7. Cargar el módulo de kernel
Entrar al directorio del módulo:

cd /home/vela/Documentos/SOPES1/PROYECTO2/kernel
Cargar el módulo:

sudo insmod continfo.ko
Verificar mensajes del kernel:

sudo dmesg | tail -n 10
8. Consultar el archivo /proc
Una vez cargado el módulo, se crea el archivo:

/proc/continfo_pr2_so1_202307705
Para visualizarlo:

cat /proc/continfo_pr2_so1_202307705
Este archivo muestra información del sistema y procesos, incluyendo:

RAM total

RAM libre

RAM usada

procesos activos

PID

PPID

nombre del proceso

VSZ

RSS

porcentaje de memoria

tiempo acumulado de CPU

9. Ejecutar los contenedores de prueba
Entrar al directorio del script:

cd /home/vela/Documentos/SOPES1/PROYECTO2/cron
Ejecutar el script:

./spawn_containers.sh
El sistema genera los siguientes contenedores de prueba:

low_1

low_2

low_3

highcpu_1

highcpu_2

highram_1

Verificar contenedores activos:

docker ps
10. Verificar consumo de recursos de los contenedores
Para observar el uso de CPU y memoria:

docker stats --no-stream
Comportamiento esperado:

highcpu_1 y highcpu_2 consumen alto porcentaje de CPU.

highram_1 consume una cantidad considerable de memoria RAM.

low_1, low_2 y low_3 consumen pocos recursos.

11. Ejecutar el daemon
Entrar al directorio del daemon:

cd /home/vela/Documentos/SOPES1/PROYECTO2/daemon
Ejecutar el daemon:

./daemon
12. Funciones del daemon
El daemon realiza las siguientes funciones:

Lee el contenido del archivo /proc/continfo_pr2_so1_202307705.

Toma dos muestras con un intervalo de tiempo.

Calcula el cambio de uso de CPU entre muestras.

Detecta procesos candidatos a eliminación.

Mapea PIDs con contenedores Docker.

Detecta contenedores candidatos según política definida.

Elimina contenedores de alto consumo cuando corresponde.

Respeta la restricción de no dejar menos de dos contenedores de tipo high.

Guarda información en un archivo JSON local.

Guarda información en Valkey.

13. Salida esperada del daemon
Durante la ejecución, el daemon muestra en consola secciones como las siguientes:

Resumen del sistema

Contenedores activos

Top 5 por RSS

Top 5 por delta CPU entre muestras

Posibles candidatos a eliminar

Contenedores candidatos a eliminar

Acción realizada

Ejemplo de mensajes esperados:

=== CONTENEDORES CANDIDATOS A ELIMINAR ===
NAME=highcpu_2 IMAGE=alpine PID=88693 DELTA_CPU=10003000000 RSS=772 KB VSZ=1640 KB MEM=0 REASON=high_cpu
NAME=highcpu_1 IMAGE=alpine PID=88630 DELTA_CPU=10001000000 RSS=776 KB VSZ=1640 KB MEM=0 REASON=high_cpu
NAME=highram_1 IMAGE=python:3.12-alpine PID=88754 DELTA_CPU=0 RSS=315816 KB VSZ=318424 KB MEM=0 REASON=high_ram
14. Política de eliminación aplicada
El daemon sigue estas reglas:

No elimina contenedores low_*.

No elimina grafana_so1.

No elimina valkey_so1.

Prioriza eliminar contenedores highcpu_*.

Si no hay highcpu_*, considera highram_*.

No elimina más contenedores high si ya solo quedan dos activos.

Ejemplo de decisión:

Contenedores high activos: 3
Eliminando contenedor candidato: highcpu_2 (motivo: high_cpu)
Contenedor eliminado correctamente: highcpu_2
O bien:

Contenedores high activos: 2
No se elimina highcpu_1 porque ya solo quedan 2 contenedores high
15. Verificación del log local
El daemon genera el archivo:

monitor_logs.jsonl
Para revisarlo:

cd /home/vela/Documentos/SOPES1/PROYECTO2/daemon
cat monitor_logs.jsonl
Este archivo registra por cada ciclo:

timestamp

memoria total, libre y usada

top procesos por RAM

top procesos por delta de CPU

contenedor candidato

razón de eliminación

acción tomada

Ejemplo de campos guardados:

deleted_container

deleted_reason

action_taken

16. Verificación de datos en Valkey
Para consultar los logs guardados en Valkey:

docker exec -it valkey_so1 valkey-cli LRANGE monitor:logs 0 -1
Para consultar métricas individuales:

docker exec -it valkey_so1 valkey-cli GET monitor:ram_total
docker exec -it valkey_so1 valkey-cli GET monitor:ram_free
docker exec -it valkey_so1 valkey-cli GET monitor:ram_used
docker exec -it valkey_so1 valkey-cli GET monitor:last_update
docker exec -it valkey_so1 valkey-cli GET monitor:top_rss
docker exec -it valkey_so1 valkey-cli GET monitor:top_cpu
17. Uso del cronjob
El cronjob permite ejecutar automáticamente el script de creación de contenedores.

Para verificar el cronjob configurado:

crontab -l
Para revisar la salida del cron:

cat /home/vela/Documentos/SOPES1/PROYECTO2/cron/cron.log
El cronjob recrea automáticamente los contenedores de prueba en el intervalo configurado.

18. Acceso a Grafana
Abrir en el navegador:

http://localhost:3000
Credenciales iniciales:

Usuario: admin

Contraseña: admin

Grafana puede utilizarse como plataforma para futura visualización de métricas del sistema.

19. Detención del daemon
Para detener el daemon, en la terminal donde se está ejecutando presionar:

Ctrl + C
20. Detener los contenedores de prueba
Entrar al directorio del script:

cd /home/vela/Documentos/SOPES1/PROYECTO2/cron
Eliminar los contenedores de prueba:

docker rm -f low_1 low_2 low_3 highcpu_1 highcpu_2 highram_1
21. Descargar el módulo del kernel
Entrar al directorio del módulo:

cd /home/vela/Documentos/SOPES1/PROYECTO2/kernel
Descargar el módulo:

sudo rmmod continfo
22. Detener Grafana y Valkey
Entrar al directorio correspondiente:

cd /home/vela/Documentos/SOPES1/PROYECTO2/grafana
Detener servicios:

docker compose down
23. Problemas comunes
El archivo /proc no existe
Verificar que el módulo esté cargado:

ls -l /proc/continfo_pr2_so1_202307705
El daemon no ejecuta correctamente
Verificar que el módulo esté cargado y que exista el archivo /proc.

Docker no responde
Verificar el servicio:

sudo systemctl status docker
Valkey no responde
Probar conectividad:

docker exec -it valkey_so1 valkey-cli ping
El cronjob no genera contenedores
Verificar:

crontab -l
cat /home/vela/Documentos/SOPES1/PROYECTO2/cron/cron.log
El módulo ya está cargado
Si aparece el error File exists:

sudo rmmod continfo
sudo insmod continfo.ko
24. Recomendaciones de uso
Ejecutar primero Valkey y Grafana.

Cargar siempre el módulo antes del daemon.

Ejecutar contenedores de prueba antes del análisis.

Revisar periódicamente monitor_logs.jsonl.

Verificar el uso de espacio en disco de los logs.

Limpiar logs si se realizan muchas pruebas continuas.

Para limpiar logs:

> /home/vela/Documentos/SOPES1/PROYECTO2/cron/cron.log
> /home/vela/Documentos/SOPES1/PROYECTO2/daemon/monitor_logs.jsonl