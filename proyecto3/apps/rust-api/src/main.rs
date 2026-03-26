use axum::{
    extract::State,
    http::StatusCode,
    response::IntoResponse,
    routing::{get, post},
    Json, Router,
};
use reqwest::Client;
use serde::{Deserialize, Serialize};
use std::{env, sync::Arc};
use tokio::net::TcpListener;

#[derive(Clone)]
struct AppState {
    client: Client,
    go_ingest_url: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
struct WarReportPayload {
    country: String,
    warplanes_in_air: i32,
    warships_in_water: i32,
    timestamp: String,
}

#[derive(Debug, Serialize)]
struct ApiResponse {
    status: String,
    message: String,
    service: String,
    forwarded_to: Option<String>,
    downstream_status: Option<u16>,
    downstream_body: Option<String>,
}

#[tokio::main]
async fn main() {
    let port = env::var("RUST_API_PORT").unwrap_or_else(|_| "8080".to_string());
    let go_ingest_url = env::var("GO_INGEST_URL")
        .unwrap_or_else(|_| "http://localhost:8081/internal/report".to_string());

    let state = Arc::new(AppState {
        client: Client::new(),
        go_ingest_url,
    });

    let app = Router::new()
        .route("/health", get(health))
        .route("/ingest", post(ingest_report))
        .with_state(state);

    let address = format!("0.0.0.0:{port}");
    let listener = TcpListener::bind(&address)
        .await
        .expect("No se pudo iniciar el servidor Rust");

    println!("Rust API escuchando en http://{address}");
    axum::serve(listener, app)
        .await
        .expect("Error al ejecutar el servidor");
}

async fn health() -> impl IntoResponse {
    let response = ApiResponse {
        status: "success".to_string(),
        message: "Rust API funcionando correctamente".to_string(),
        service: "rust-api".to_string(),
        forwarded_to: None,
        downstream_status: None,
        downstream_body: None,
    };

    (StatusCode::OK, Json(response))
}

async fn ingest_report(
    State(state): State<Arc<AppState>>,
    Json(mut payload): Json<WarReportPayload>,
) -> impl IntoResponse {
    payload.country = payload.country.trim().to_uppercase();
    payload.timestamp = payload.timestamp.trim().to_string();

    if let Err(error_message) = validar_payload(&payload) {
        let response = ApiResponse {
            status: "error".to_string(),
            message: error_message,
            service: "rust-api".to_string(),
            forwarded_to: None,
            downstream_status: None,
            downstream_body: None,
        };

        return (StatusCode::BAD_REQUEST, Json(response)).into_response();
    }

    let result = state
        .client
        .post(&state.go_ingest_url)
        .json(&payload)
        .send()
        .await;

    match result {
        Ok(resp) => {
            let status = resp.status();
            let status_code = status.as_u16();
            let body = resp
                .text()
                .await
                .unwrap_or_else(|_| "No se pudo leer la respuesta del servicio Go".to_string());

            if !status.is_success() {
                let response = ApiResponse {
                    status: "error".to_string(),
                    message: "El servicio go-ingest respondió con error".to_string(),
                    service: "rust-api".to_string(),
                    forwarded_to: Some(state.go_ingest_url.clone()),
                    downstream_status: Some(status_code),
                    downstream_body: Some(body),
                };

                return (StatusCode::BAD_GATEWAY, Json(response)).into_response();
            }

            let response = ApiResponse {
                status: "success".to_string(),
                message: "Reporte recibido y reenviado correctamente a go-ingest".to_string(),
                service: "rust-api".to_string(),
                forwarded_to: Some(state.go_ingest_url.clone()),
                downstream_status: Some(status_code),
                downstream_body: Some(body),
            };

            (StatusCode::OK, Json(response)).into_response()
        }
        Err(error) => {
            let response = ApiResponse {
                status: "error".to_string(),
                message: format!("No se pudo conectar con go-ingest: {error}"),
                service: "rust-api".to_string(),
                forwarded_to: Some(state.go_ingest_url.clone()),
                downstream_status: None,
                downstream_body: None,
            };

            (StatusCode::BAD_GATEWAY, Json(response)).into_response()
        }
    }
}

fn validar_payload(payload: &WarReportPayload) -> Result<(), String> {
    let paises_validos = ["USA", "RUS", "CHN", "ESP", "GTM"];

    if !paises_validos.contains(&payload.country.as_str()) {
        return Err(format!(
            "El país '{}' no es válido. Valores permitidos: USA, RUS, CHN, ESP, GTM",
            payload.country
        ));
    }

    if payload.warplanes_in_air < 0 {
        return Err("warplanes_in_air no puede ser negativo".to_string());
    }

    if payload.warships_in_water < 0 {
        return Err("warships_in_water no puede ser negativo".to_string());
    }

    if payload.timestamp.is_empty() {
        return Err("timestamp es obligatorio".to_string());
    }

    Ok(())
}