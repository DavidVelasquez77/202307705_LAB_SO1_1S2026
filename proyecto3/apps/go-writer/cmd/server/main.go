package main

import (
	"context"
	"log"
	"net"
	"os"

	wartweets "github.com/DavidVelasquez77/202307705_LAB_SO1_1S2026/proyecto3/proto/gen/go/wartweets"
	"google.golang.org/grpc"
)

type server struct {
	wartweets.UnimplementedWarReportServiceServer
}

func (s *server) SendReport(ctx context.Context, req *wartweets.WarReportRequest) (*wartweets.WarReportResponse, error) {
	log.Printf(
		"Reporte recibido -> country=%s, warplanes=%d, warships=%d, timestamp=%s",
		req.GetCountry().String(),
		req.GetWarplanesInAir(),
		req.GetWarshipsInWater(),
		req.GetTimestamp(),
	)

	return &wartweets.WarReportResponse{
		Status: "ok - reporte recibido por go-writer",
	}, nil
}

func main() {
	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "50051"
	}

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("No se pudo abrir el puerto %s: %v", port, err)
	}

	grpcServer := grpc.NewServer()
	wartweets.RegisterWarReportServiceServer(grpcServer, &server{})

	log.Printf("go-writer escuchando en grpc://localhost:%s", port)

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Error al levantar go-writer: %v", err)
	}
}