package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "os"
)

type Response struct {
    Message  string `json:"message"`
    Server   string `json:"server"`
    Carnet   string `json:"carnet"`
    Autor    string `json:"autor"`
}

func handler(w http.ResponseWriter, r *http.Request) {
    // Obtiene el nombre del hostname
    hostname, err := os.Hostname()
    if err != nil {
        hostname = "Desconocido"
    }

    // Respuesta JSON
    resp := Response{
        Message: "PROYECTO 1 - Sistemas Operativos 1",
        Server:  hostname,     
        Carnet:  "202307705",
        Autor:   "Vela",       
    }

    // Configurar cabeceras y enviar respuesta
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resp)
}

func main() {
    
    http.HandleFunc("/", handler)
    
    fmt.Println("Servidor corriendo en el puerto 8080...")
    fmt.Println("Ingresa a: http://localhost:8080")
    
    // Escuchar en todas las interfaces (0.0.0.0) para que sea accesible desde fuera
    err := http.ListenAndServe("0.0.0.0:8080", nil)
    if err != nil {
        fmt.Println("Error al iniciar el servidor:", err)
    }
}
