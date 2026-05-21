package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

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
	// En local el sidecar puede tardar unos segundos en estar listo.
	client, err := newDaprClientWithRetry(20, time.Second)
	if err != nil {
		log.Fatalf("Error iniciando el cliente de Dapr: %v", err)
	}
	daprClient = client
	defer daprClient.Close()

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

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

func newDaprClientWithRetry(maxAttempts int, delay time.Duration) (dapr.Client, error) {
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		client, err := dapr.NewClient()
		if err == nil {
			return client, nil
		}

		lastErr = err
		log.Printf("Esperando sidecar de Dapr (%d/%d): %v", attempt, maxAttempts, err)
		time.Sleep(delay)
	}

	return nil, lastErr
}
