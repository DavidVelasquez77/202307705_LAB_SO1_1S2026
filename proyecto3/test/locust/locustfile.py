import random
import json
from datetime import datetime, timezone
from locust import HttpUser, task, between

class MilitaryReportUser(HttpUser):
    # Simula un tiempo de espera entre 1 y 2 segundos entre cada envío por usuario
    wait_time = between(1, 2)

    @task
    def send_report(self):
        # Lista de países permitidos
        countries = ["USA", "RUS", "CHN", "ESP", "GTM"]
        
        # Generar datos aleatorios según los rangos del PDF
        payload = {
            "country": random.choice(countries),
            "warplanes_in_air": random.randint(0, 50),
            "warships_in_water": random.randint(0, 30),
            # Formato de fecha exacto requerido: "2026-03-12T20:15:30Z"
            "timestamp": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
        }

        # Enviar la petición POST a la ruta principal usando tu carnet
        headers = {'Content-Type': 'application/json'}
        
        # Hacemos el POST a la ruta del Gateway API
        with self.client.post("/grpc-202307705", data=json.dumps(payload), headers=headers, catch_response=True) as response:
            if response.status_code in [200, 201]:
                response.success()
            else:
                response.failure(f"Fallo al enviar reporte. Código: {response.status_code}")