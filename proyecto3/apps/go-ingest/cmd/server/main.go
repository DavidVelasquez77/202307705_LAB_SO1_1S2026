package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	wartweets "github.com/DavidVelasquez77/202307705_LAB_SO1_1S2026/proyecto3/proto/gen/go/wartweets"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type WarReportPayload struct {
	Country         string `json:"country"`
	WarplanesInAir  int32  `json:"warplanes_in_air"`
	WarshipsInWater int32  `json:"warships_in_water"`
	Timestamp       string `json:"timestamp"`
}

type APIResponse struct {
	Status       string `json:"status"`
	Message      string `json:"message"`
	Service      string `json:"service"`
	GRPCTarget   string `json:"grpc_target,omitempty"`
	GRPCResponse string `json:"grpc_response,omitempty"`
}

func main() {
	httpPort := os.Getenv("GO_INGEST_PORT")
	if httpPort == "" {
		httpPort = "8081"
	}

	grpcTarget := os.Getenv("GRPC_WRITER_ADDR")
	if grpcTarget == "" {
		grpcTarget = "localhost:50051"
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, APIResponse{
			Status:  "success",
			Message: "go-ingest funcionando correctamente",
			Service: "go-ingest",
		})
	})

	mux.HandleFunc("/internal/report", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, APIResponse{
				Status:  "error",
				Message: "Método no permitido",
				Service: "go-ingest",
			})
			return
		}

		var payload WarReportPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeJSON(w, http.StatusBadRequest, APIResponse{
				Status:  "error",
				Message: "JSON inválido: " + err.Error(),
				Service: "go-ingest",
			})
			return
		}

		payload.Country = strings.ToUpper(strings.TrimSpace(payload.Country))
		payload.Timestamp = strings.TrimSpace(payload.Timestamp)

		if err := validatePayload(payload); err != nil {
			writeJSON(w, http.StatusBadRequest, APIResponse{
				Status:  "error",
				Message: err.Error(),
				Service: "go-ingest",
			})
			return
		}

		countryEnum, err := mapCountryToEnum(payload.Country)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, APIResponse{
				Status:  "error",
				Message: err.Error(),
				Service: "go-ingest",
			})
			return
		}

		grpcResp, err := sendToGRPC(grpcTarget, &wartweets.WarReportRequest{
			Country:          countryEnum,
			WarplanesInAir:   payload.WarplanesInAir,
			WarshipsInWater:  payload.WarshipsInWater,
			Timestamp:        payload.Timestamp,
		})
		if err != nil {
			writeJSON(w, http.StatusBadGateway, APIResponse{
				Status:     "error",
				Message:    "Error enviando reporte por gRPC: " + err.Error(),
				Service:    "go-ingest",
				GRPCTarget: grpcTarget,
			})
			return
		}

		writeJSON(w, http.StatusOK, APIResponse{
			Status:       "success",
			Message:      "Reporte recibido por HTTP y enviado por gRPC",
			Service:      "go-ingest",
			GRPCTarget:   grpcTarget,
			GRPCResponse: grpcResp.GetStatus(),
		})
	})

	server := &http.Server{
		Addr:              ":" + httpPort,
		Handler:           logMiddleware(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("go-ingest escuchando en http://localhost:%s", httpPort)
	log.Printf("go-ingest enviará reportes a gRPC target: %s", grpcTarget)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("No se pudo iniciar go-ingest: %v", err)
	}
}

func validatePayload(p WarReportPayload) error {
	validCountries := map[string]bool{
		"USA": true,
		"RUS": true,
		"CHN": true,
		"ESP": true,
		"GTM": true,
	}

	if !validCountries[p.Country] {
		return &appError{"country inválido. Permitidos: USA, RUS, CHN, ESP, GTM"}
	}
	if p.WarplanesInAir < 0 || p.WarplanesInAir > 50 {
		return &appError{"warplanes_in_air debe estar entre 0 y 50"}
	}
	if p.WarshipsInWater < 0 || p.WarshipsInWater > 30 {
		return &appError{"warships_in_water debe estar entre 0 y 30"}
	}
	if p.Timestamp == "" {
		return &appError{"timestamp es obligatorio"}
	}

	return nil
}

func mapCountryToEnum(country string) (wartweets.Countries, error) {
	switch country {
	case "USA":
		return wartweets.Countries_usa, nil
	case "RUS":
		return wartweets.Countries_rus, nil
	case "CHN":
		return wartweets.Countries_chn, nil
	case "ESP":
		return wartweets.Countries_esp, nil
	case "GTM":
		return wartweets.Countries_gtm, nil
	default:
		return wartweets.Countries_countries_unknown, &appError{"no se pudo mapear el country al enum gRPC"}
	}
}

func sendToGRPC(target string, req *wartweets.WarReportRequest) (*wartweets.WarReportResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := wartweets.NewWarReportServiceClient(conn)

	return client.SendReport(ctx, req)
}

func writeJSON(w http.ResponseWriter, statusCode int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")

	if err := enc.Encode(body); err != nil {
		http.Error(w, `{"status":"error","message":"no se pudo serializar la respuesta"}`, http.StatusInternalServerError)
		return
	}

	_, _ = w.Write(buf.Bytes())
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

type appError struct {
	msg string
}

func (e *appError) Error() string {
	return e.msg
}