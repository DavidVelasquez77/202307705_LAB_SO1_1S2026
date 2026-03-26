package main

import (
	"context"
	"log"
	"net"
	"os"
	"time"

	"github.com/DavidVelasquez77/202307705_LAB_SO1_1S2026/proyecto3/apps/go-writer/internal/rabbitmq"
	wartweets "github.com/DavidVelasquez77/202307705_LAB_SO1_1S2026/proyecto3/proto/gen/go/wartweets"
	"google.golang.org/grpc"
)

type server struct {
	wartweets.UnimplementedWarReportServiceServer
	pub *rabbitmq.Publisher
}

func (s *server) SendReport(ctx context.Context, req *wartweets.WarReportRequest) (*wartweets.WarReportResponse, error) {
	err := s.pub.Publish(req)
	if err != nil {
		log.Printf(" Error al publicar en RabbitMQ: %v", err)
		return &wartweets.WarReportResponse{
			Status: "error - no se pudo encolar",
		}, err
	}

	return &wartweets.WarReportResponse{
		Status: "ok - reporte encolado en RabbitMQ",
	}, nil
}

func main() {
	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "50051"
	}

	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@localhost:5672/"
	}

	log.Println("Conectando a RabbitMQ...")
	var pub *rabbitmq.Publisher
	var err error

	// Lógica de reintentos (5 intentos, espera 3 segundos entre cada uno)
	for i := 1; i <= 5; i++ {
		pub, err = rabbitmq.NewPublisher(rabbitURL)
		if err == nil {
			break
		}
		log.Printf("Intento %d fallido al conectar a RabbitMQ. Reintentando en 3s... Error: %v", i, err)
		time.Sleep(3 * time.Second)
	}

	if err != nil {
		log.Fatalf("No se pudo conectar a RabbitMQ después de 5 intentos: %v", err)
	}
	defer pub.Close()
	log.Println(" Conectado a RabbitMQ exitosamente")

	listener, err := net.Listen("tcp", "0.0.0.0:"+port)
	if err != nil {
		log.Fatalf("No se pudo abrir el puerto %s: %v", port, err)
	}

	grpcServer := grpc.NewServer()
	wartweets.RegisterWarReportServiceServer(grpcServer, &server{pub: pub})

	log.Printf("go-writer escuchando en grpc://0.0.0.0:%s", port)

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Error al levantar go-writer: %v", err)
	}
}