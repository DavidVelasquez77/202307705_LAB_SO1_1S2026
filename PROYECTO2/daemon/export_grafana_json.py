import json
from pathlib import Path

base = Path("/home/vela/Documentos/SOPES1/PROYECTO2/daemon")
log_file = base / "monitor_logs.jsonl"

ram_timeseries = []
deleted = []
top_rss_latest = []
top_cpu_latest = []
summary_latest = {}

if log_file.exists():
    lines = [line.strip() for line in log_file.read_text().splitlines() if line.strip()]
    records = [json.loads(line) for line in lines]

    for r in records:
        ram_timeseries.append({
            "timestamp": r.get("timestamp"),
            "ram_total_kb": r.get("ram_total_kb"),
            "ram_free_kb": r.get("ram_free_kb"),
            "ram_used_kb": r.get("ram_used_kb"),
        })

        if r.get("action_taken") == "removed":
            deleted.append({
                "timestamp": r.get("timestamp"),
                "deleted_container": r.get("deleted_container"),
                "deleted_reason": r.get("deleted_reason"),
                "count": 1
            })

    if records:
        latest = records[-1]
        top_rss_latest = latest.get("top_rss", [])
        top_cpu_latest = latest.get("top_cpu_delta", [])
        summary_latest = [{
            "timestamp": latest.get("timestamp"),
            "ram_total_kb": latest.get("ram_total_kb"),
            "ram_free_kb": latest.get("ram_free_kb"),
            "ram_used_kb": latest.get("ram_used_kb"),
            "deleted_container": latest.get("deleted_container"),
            "deleted_reason": latest.get("deleted_reason"),
            "action_taken": latest.get("action_taken"),
        }]

(base / "ram_timeseries.json").write_text(json.dumps(ram_timeseries, indent=2))
(base / "deleted_containers.json").write_text(json.dumps(deleted, indent=2))
(base / "top_rss_latest.json").write_text(json.dumps(top_rss_latest, indent=2))
(base / "top_cpu_latest.json").write_text(json.dumps(top_cpu_latest, indent=2))
(base / "summary_latest.json").write_text(json.dumps(summary_latest, indent=2))

print("Archivos JSON para Grafana generados correctamente.")