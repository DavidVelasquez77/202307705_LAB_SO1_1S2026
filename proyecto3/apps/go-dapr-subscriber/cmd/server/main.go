package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

type WarReport struct {
	Country         string `json:"country"`
	WarplanesInAir  int32  `json:"warplanes_in_air"`
	WarshipsInWater int32  `json:"warships_in_water"`
	Timestamp       string `json:"timestamp"`
}

type CloudEvent struct {
	Data WarReport `json:"data"`
}

var rdb *redis.Client

func main() {
	valkeyURL := os.Getenv("VALKEY_URL")
	if valkeyURL == "" {
		valkeyURL = "redis://valkey-vm.proyecto3.svc.cluster.local:6379/0"
	}
	opt, _ := redis.ParseURL(valkeyURL)
	rdb = redis.NewClient(opt)

	http.HandleFunc("/events/war-reports", func(w http.ResponseWriter, r *http.Request) {
		var evt CloudEvent
		if err := json.NewDecoder(r.Body).Decode(&evt); err != nil {
			http.Error(w, "bad request", 400)
			return
		}
		
		ctx := context.Background()
		msg := evt.Data
		raw, _ := json.Marshal(msg)
		
		pipe := rdb.TxPipeline()
		pipe.LPush(ctx, "reports", string(raw))
		pipe.LPush(ctx, "reports:country:"+msg.Country, string(raw))
		pipe.Incr(ctx, "stats:total_reports")
		
		//  PUNTOS EXTRA: Expiración de 24 horas (TTL) 
		pipe.Expire(ctx, "reports", 24*time.Hour)
		pipe.Expire(ctx, "reports:country:"+msg.Country, 24*time.Hour)
		
		pipe.Exec(ctx)
		
		log.Printf("Evento Dapr guardado: %s", msg.Country)
		w.WriteHeader(http.StatusOK)
	})

	log.Println("Subscriber Dapr escuchando en :8083")
	log.Fatal(http.ListenAndServe(":8083", nil))
}
