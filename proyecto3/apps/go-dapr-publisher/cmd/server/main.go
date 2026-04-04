package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	dapr "github.com/dapr/go-sdk/client"
)

type WarReport struct {
	Country         string `json:"country"`
	WarplanesInAir  int    `json:"warplanes_in_air"`
	WarshipsInWater int    `json:"warships_in_water"`
	Timestamp       string `json:"timestamp"`
}

// 1. Variable global para reutilizar la conexión gRPC
var daprClient dapr.Client

func main() {
	// 2. Iniciamos el cliente de Dapr UNA SOLA VEZ al arrancar
	client, err := dapr.NewClient()
	if err != nil {
		log.Fatalf("Error iniciando el cliente de Dapr: %v", err)
	}
	daprClient = client
	defer daprClient.Close()

	http.HandleFunc("/publish", func(w http.ResponseWriter, r *http.Request) {
		var payload WarReport
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// 3. Usamos la conexión global para publicar
		err = daprClient.PublishEvent(context.Background(), "rabbitmq-pubsub", "war-reports", payload)
		
		// 4. Validamos si Dapr falló
		if err != nil {
			log.Printf("Error al publicar en RabbitMQ vía Dapr: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"status":"error","flow":"dapr"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"published","flow":"dapr"}`))
	})

	log.Println("Publisher Dapr escuchando en :8082")
	log.Fatal(http.ListenAndServe(":8082", nil))
}