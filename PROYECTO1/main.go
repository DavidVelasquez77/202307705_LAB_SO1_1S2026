package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const MyCarnet = "202307705" 

var apiMap = map[string]string{
	"API1": "http://192.168.122.250:8080", // VM1 - Puerto 8080
	"API2": "http://192.168.122.250:8081", // VM1 - Puerto 8081
	"API3": "http://192.168.122.246:8080", // VM2 - Puerto 8080
}

type HealthResponse struct {
	Status    string `json:"status"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
	VM        string `json:"VM"`
	Carnet    string `json:"carnet"`
}

type CallResponse struct {
	ApiName    string `json:"apiname"`
	Message    string `json:"message"`
	Connection bool   `json:"connection"`
	Carnet     string `json:"carnet"`
}

var currentAPI string
var currentPort string
var currentVM string

func main() {
	// Banderas para configurar quién soy al ejecutar el programa
	flag.StringVar(&currentAPI, "name", "UNKNOWN", "Nombre de la API (API1, API2, API3)")
	flag.StringVar(&currentPort, "port", "8080", "Puerto donde escuchará el servidor")
	flag.Parse()

	// Detectar nombre de la VM automáticamente
	hostname, _ := os.Hostname()
	currentVM = hostname

	if currentAPI == "UNKNOWN" {
		fmt.Println("ERROR: Debes especificar el nombre de la API con -name")
		fmt.Println("Ejemplo: go run main.go -name API1 -port 8080")
		os.Exit(1)
	}

	http.HandleFunc("/", router)

	fmt.Printf("Iniciando %s en %s (%s) puerto %s...\n", currentAPI, currentVM, MyCarnet, currentPort)
	err := http.ListenAndServe("0.0.0.0:"+currentPort, nil)
	if err != nil {
		fmt.Printf("Error al iniciar servidor: %s\n", err)
	}
}

func router(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// 1. Endpoint /health 
	if path == "/health" {
		handleHealth(w, r)
		return
	}

	// 2. Endpoint de llamadas 
	if strings.Contains(path, "/call-api") {
		handleCall(w, r)
		return
	}

	http.NotFound(w, r)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	
	resp := HealthResponse{
		Status:    "UP",
		Message:   currentAPI + " is Ready",
		Timestamp: time.Now().Format("2006-01-02 15:04:05"), 
		VM:        currentVM,
		Carnet:    MyCarnet,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleCall(w http.ResponseWriter, r *http.Request) {
	
	parts := strings.Split(r.URL.Path, "/")
	
	if len(parts) < 4 {
		http.Error(w, "URL Inválida", http.StatusBadRequest)
		return
	}

	// Identificar a quién llamamos 
	targetPart := parts[len(parts)-1] // "call-api2"
	targetName := strings.ToUpper(strings.Replace(targetPart, "call-", "", 1)) // "API2"
	
	carnetInURL := parts[2]
	if carnetInURL != MyCarnet {
		http.Error(w, "Carnet incorrecto en URL", http.StatusBadRequest)
		return
	}

	targetURL, exists := apiMap[targetName]
	if !exists {
		respondCall(w, targetName, "ERROR: API desconocida en el mapa", false)
		return
	}

	// Hacer petición HTTP al /health del destino
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(targetURL + "/health")

	if err != nil {
		msg := fmt.Sprintf("ERROR: The %s located on the remote VM is not working", targetName)
		respondCall(w, targetName, msg, false)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var healthData HealthResponse
	json.Unmarshal(body, &healthData)

	// Validar Status UP
	if healthData.Status == "UP" {
		msg := fmt.Sprintf("The %s located on the %s is working", targetName, healthData.VM)
		respondCall(w, targetName, msg, true)
	} else {
		msg := fmt.Sprintf("ERROR: The %s located on the remote VM is not working (Status: %s)", targetName, healthData.Status)
		respondCall(w, targetName, msg, false)
	}
}

func respondCall(w http.ResponseWriter, apiName, msg string, connection bool) {
	resp := CallResponse{
		ApiName:    apiName,
		Message:    msg,
		Connection: connection,
		Carnet:     MyCarnet,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}