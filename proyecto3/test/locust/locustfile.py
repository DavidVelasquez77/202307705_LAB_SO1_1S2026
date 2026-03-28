import random
import json
from datetime import datetime, timezone
from locust import HttpUser, task, between

class MilitaryReportUser(HttpUser):
    
    wait_time = between(0.1, 0.5)

    @task
    def send_report(self):
        countries = ["USA", "RUS", "CHN", "ESP", "GTM"]
        
        payload = {
            "country": random.choice(countries),
            "warplanes_in_air": random.randint(0, 50),
            "warships_in_water": random.randint(0, 30),
            "timestamp": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
        }

        headers = {'Content-Type': 'application/json'}
        
        with self.client.post("/grpc-202307705", data=json.dumps(payload), headers=headers, catch_response=True) as response:
            if response.status_code == 200:
                response.success()
            else:
                response.failure(f"Fallo con código: {response.status_code}")