# GUÍA DE INSTALACIÓN Y DESPLIEGUE
## Proyecto 1 - Sistemas Operativos 1

## 1. Requisitos Previos

### Software Necesario

- **SO Host:** Linux (Ubuntu/Debian)
- **Virtualización:** KVM/QEMU con libvirt
- **Herramientas:** Docker, curl, virsh, ctr

### Instalación Rápida

```bash
# Instalar KVM y herramientas
sudo apt update
sudo apt install -y qemu-kvm libvirt-daemon-system libvirt-clients virt-manager curl

# Iniciar libvirt
sudo systemctl enable libvirtd
sudo systemctl start libvirtd
```

---

## 2. Configuración de Red

### Asignación de IPs

| VM | IP | Rol | Puertos |
|----|-------|-----|---------|
| VM1 | 192.168.122.250 | APIs 1 y 2 | 8080, 8081 |
| VM2 | 192.168.122.246 | API 3 | 8080 |
| VM3 | 192.168.122.141 | Zot Registry | 5000 |

**Verificar IPs:**
```bash
sudo virsh domifaddr <VM-NAME>

#Ejemplo
sudo virsh domifaddr S01-VM1
sudo virsh domifaddr S01-VM2
sudo virsh domifaddr S01-VM3
```

---

## 3. Construcción de Imágenes

```bash
# Construir imágenes
docker build -t api1-202307705:latest .
docker build -t api2-202307705:latest .
docker build -t api3-202307705:latest .

# Verificar
docker images | grep 202307705
```

---

## 4. Push al Registro Zot (VM3)

### Configurar Registry Inseguro

```bash
# Editar daemon.json
sudo nano /etc/docker/daemon.json
```

Agregar:
```json
{
  "insecure-registries": ["192.168.122.141:5000"]
}
```

```bash
sudo systemctl restart docker
```

### Etiquetar y Subir

```bash
# Tag
docker tag api1-202307705:latest 192.168.122.141:5000/api1-202307705:latest
docker tag api2-202307705:latest 192.168.122.141:5000/api2-202307705:latest
docker tag api3-202307705:latest 192.168.122.141:5000/api3-202307705:latest

# Push
docker push 192.168.122.141:5000/api1-202307705:latest
docker push 192.168.122.141:5000/api2-202307705:latest
docker push 192.168.122.141:5000/api3-202307705:latest

# Verificar
curl http://192.168.122.141:5000/v2/_catalog
```

---

## 5. Despliegue en VM1 (API1 y API2)

### Configurar Containerd

```bash
sudo nano /etc/containerd/config.toml
```

Agregar:
```toml
[plugins."io.containerd.grpc.v1.cri".registry.mirrors."192.168.122.141:5000"]
  endpoint = ["http://192.168.122.141:5000"]
[plugins."io.containerd.grpc.v1.cri".registry.configs."192.168.122.141:5000".tls]
  insecure_skip_verify = true
```

```bash
sudo systemctl restart containerd
```

### Pull y Ejecutar

```bash
# Pull imágenes
sudo ctr images pull --plain-http 192.168.122.141:5000/api1-202307705:latest
sudo ctr images pull --plain-http 192.168.122.141:5000/api2-202307705:latest

# Ejecutar API1 (puerto 8080)
sudo ctr run -d --net-host \
  192.168.122.141:5000/api1-202307705:latest \
  contenedor-api1 \
  /app/servidor-api -name API1 -port 8080

# Ejecutar API2 (puerto 8081)
sudo ctr run -d --net-host \
  192.168.122.141:5000/api2-202307705:latest \
  contenedor-api2 \
  /app/servidor-api -name API2 -port 8081

# Verificar
sudo ctr tasks list
```

---

## 6. Despliegue en VM2 (API3)

```bash
# Configurar containerd (mismo que VM1)
# Luego pull y ejecutar

sudo ctr images pull --plain-http 192.168.122.141:5000/api3-202307705:latest

sudo ctr run -d --net-host \
  192.168.122.141:5000/api3-202307705:latest \
  contenedor-api3 \
  /app/servidor-api -name API3 -port 8080

# Verificar
sudo ctr tasks list
```

---

## 7. Pruebas de Validación

### Health Checks

```bash
curl http://192.168.122.250:8080/health  # API1
curl http://192.168.122.250:8081/health  # API2
curl http://192.168.122.246:8080/health  # API3
```

### Comunicación entre APIs

```bash
# API2 llama a API1
curl http://192.168.122.250:8081/api2/202307705/call-api1

# API1 llama a API3
curl http://192.168.122.250:8080/api1/202307705/call-api3

# API3 llama a API1
curl http://192.168.122.246:8080/api3/202307705/call-api1
```

---

## 8. Solución de Problemas

### Error: "HTTP response to HTTPS client"

```bash
# Agregar registry inseguro en /etc/docker/daemon.json
{"insecure-registries": ["192.168.122.141:5000"]}
sudo systemctl restart docker
```

### Error: "Connection refused"

```bash
# Verificar que el contenedor está corriendo
sudo ctr tasks list

# Verificar puertos
sudo netstat -tulpn | grep 808
```

### Error ctr pull

```bash
# Usar SIEMPRE --plain-http
sudo ctr images pull --plain-http 192.168.122.141:5000/api1-202307705:latest
```

---