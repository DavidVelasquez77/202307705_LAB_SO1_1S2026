#!/bin/bash

echo "Iniciando contenedores de prueba..."

# Contenedor de bajo consumo
docker run -d --rm --name low_test alpine sh -c "sleep 300"

# Contenedor de alto CPU
docker run -d --rm --name highcpu_test alpine sh -c "while true; do :; done"

echo "Contenedores creados:"
docker ps --format "table {{.Names}}\t{{.Image}}\t{{.Status}}"