PROTO_DIR := proto
GEN_GO_DIR := $(PROTO_DIR)/gen/go/wartweets

.PHONY: proto proto-check clean-proto

proto:
	@echo "Generando código gRPC para Go..."
	@mkdir -p $(GEN_GO_DIR)
	@cd $(PROTO_DIR) && protoc --proto_path=. \
		--go_out=gen/go/wartweets --go_opt=paths=source_relative \
		--go-grpc_out=gen/go/wartweets --go-grpc_opt=paths=source_relative \
		war_report.proto
	@echo "¡Código Go generado exitosamente!"

proto-check:
	@echo "Archivos generados:"
	@find $(PROTO_DIR) -type f | sort

clean-proto:
	@echo "Eliminando archivos generados..."
	@rm -f $(GEN_GO_DIR)/war_report.pb.go
	@rm -f $(GEN_GO_DIR)/war_report_grpc.pb.go
	@echo "Archivos eliminados."