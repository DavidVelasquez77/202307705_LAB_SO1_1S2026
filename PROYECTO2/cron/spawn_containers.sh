#!/bin/bash

echo "Limpiando contenedores anteriores de prueba..."
docker rm -f low_1 low_2 low_3 highcpu_1 highcpu_2 highram_1 highram_2 2>/dev/null

echo "Creando contenedores de bajo consumo..."
docker run -d --rm --name low_1 alpine sh -c "sleep 600"
docker run -d --rm --name low_2 alpine sh -c "sleep 600"
docker run -d --rm --name low_3 alpine sh -c "sleep 600"

# Pool de workloads high
high_options=("highcpu_1" "highcpu_2" "highram_1" "highram_2")

# Mezclar y tomar 2 aleatorios
selected=($(printf "%s\n" "${high_options[@]}" | shuf | head -n 2))

echo "Creando contenedores high aleatorios..."
for item in "${selected[@]}"; do
    case "$item" in
        highcpu_1)
            docker run -d --rm --name highcpu_1 alpine sh -c "while true; do :; done"
            ;;
        highcpu_2)
            docker run -d --rm --name highcpu_2 alpine sh -c "while true; do :; done"
            ;;
        highram_1)
            docker run -d --rm --name highram_1 python:3.12-alpine sh -c "python3 -c 'x = bytearray(300 * 1024 * 1024); import time; time.sleep(600)'"
            ;;
        highram_2)
            docker run -d --rm --name highram_2 python:3.12-alpine sh -c "python3 -c 'x = bytearray(250 * 1024 * 1024); import time; time.sleep(600)'"
            ;;
    esac
done

echo
echo "Contenedores activos:"
docker ps --format "table {{.Names}}\t{{.Image}}\t{{.Status}}"