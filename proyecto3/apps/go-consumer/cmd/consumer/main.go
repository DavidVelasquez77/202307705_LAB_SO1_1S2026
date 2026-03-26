package main

import (
	"log"
	"os"
	"time"

	"github.com/DavidVelasquez77/202307705_LAB_SO1_1S2026/proyecto3/apps/go-consumer/internal/rabbitmq"
)

func main() {
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@localhost:5672/"
	}

	log.Println("Conectando go-consumer a RabbitMQ...")

	var cons *rabbitmq.Consumer
	var err error

	for i := 1; i <= 5; i++ {
		cons, err = rabbitmq.NewConsumer(rabbitURL)
		if err == nil {
			break
		}

		log.Printf("Intento %d fallido al conectar. Reintentando en 3s... Error: %v", i, err)
		time.Sleep(3 * time.Second)
	}

	if err != nil {
		log.Fatalf("No se pudo conectar a RabbitMQ después de 5 intentos: %v", err)
	}
	defer cons.Close()

	log.Println("go-consumer conectado a RabbitMQ exitosamente")

	if err := cons.StartConsuming(); err != nil {
		log.Fatalf("Error al consumir mensajes: %v", err)
	}
}