#!/bin/bash
sudo apt-get update
sudo apt-get install -y wget apache2-utils

# Descargar Zot
sudo wget https://github.com/project-zot/zot/releases/download/v2.1.15/zot-linux-amd64 -O /usr/bin/zot
sudo chmod +x /usr/bin/zot

# Crear directorios
sudo mkdir -p /etc/zot
sudo mkdir -p /var/lib/registry

# Copiar certificados (asume que los subiste con gcloud compute scp)
sudo cp ~/zot.cert /etc/zot/cert.pem
sudo cp ~/zot.key /etc/zot/key.pem

# Crear archivo de contraseñas de Zot
sudo htpasswd -bc /etc/zot/htpasswd miusuario miparawordsecreta

# Crear configuración de Zot SIN allowReadAccess
cat << 'EOF2' | sudo tee /etc/zot/config.json
{
  "distSpecVersion": "1.1.0",
  "storage": {
    "rootDirectory": "/var/lib/registry",
    "gc": true,
    "gcDelay": "2h",
    "gcInterval": "1h"
  },
  "http": {
    "address": "0.0.0.0",
    "port": "5000",
    "tls": {
      "cert": "/etc/zot/cert.pem",
      "key": "/etc/zot/key.pem"
    },
    "auth": {
      "htpasswd": {
        "path": "/etc/zot/htpasswd"
      }
    }
  },
  "log": {
    "level": "debug"
  }
}
EOF2

# Verificar si hay otra instancia de Zot corriendo y matarla
sudo pkill -f zot

# Iniciar Zot en background
sudo nohup /usr/bin/zot serve /etc/zot/config.json > /var/log/zot.log 2>&1 &
echo "Zot registry iniciado"
