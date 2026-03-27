package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type App struct {
	rdb             *redis.Client
	assignedCountry string
}

type APIResponse struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data,omitempty"`
	Error  string      `json:"error,omitempty"`
}

type SummaryResponse struct {
	AssignedCountry       string `json:"assigned_country"`
	TotalReports          int64  `json:"total_reports"`
	AssignedCountryTotal  int64  `json:"assigned_country_total_reports"`
	MaxWarplanes          int64  `json:"max_warplanes"`
	MinWarplanes          int64  `json:"min_warplanes"`
	MaxWarships           int64  `json:"max_warships"`
	MinWarships           int64  `json:"min_warships"`
}

type RankedCountry struct {
	Country string  `json:"country"`
	Score   float64 `json:"score"`
}

type ModeResponse struct {
	Warplanes struct {
		Value int64 `json:"value"`
		Count int64 `json:"count"`
	} `json:"warplanes"`
	Warships struct {
		Value int64 `json:"value"`
		Count int64 `json:"count"`
	} `json:"warships"`
}

type TimelinePoint struct {
	Timestamp        string `json:"timestamp"`
	Unix             int64  `json:"unix"`
	WarplanesInAir   int64  `json:"warplanes_in_air"`
	WarshipsInWater  int64  `json:"warships_in_water"`
	Country          string `json:"country"`
}

type RawReport struct {
	Country         string `json:"country"`
	WarplanesInAir  int64  `json:"warplanes_in_air"`
	WarshipsInWater int64  `json:"warships_in_water"`
	Timestamp       string `json:"timestamp"`
}

func main() {
	valkeyURL := os.Getenv("VALKEY_URL")
	if valkeyURL == "" {
		valkeyURL = "redis://localhost:6379/0"
	}

	statsPort := os.Getenv("STATS_API_PORT")
	if statsPort == "" {
		statsPort = "8082"
	}

	assignedCountry := os.Getenv("ASSIGNED_COUNTRY")
	if assignedCountry == "" {
		assignedCountry = "CHN"
	}

	opt, err := redis.ParseURL(valkeyURL)
	if err != nil {
		log.Fatalf("URL de Valkey inválida: %v", err)
	}

	rdb := redis.NewClient(opt)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("No se pudo conectar a Valkey: %v", err)
	}

	app := &App{
		rdb:             rdb,
		assignedCountry: assignedCountry,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", app.handleHealth)
	mux.HandleFunc("/stats/summary", app.handleSummary)
	mux.HandleFunc("/stats/top-warplanes", app.handleTopWarplanes)
	mux.HandleFunc("/stats/top-warships", app.handleTopWarships)
	mux.HandleFunc("/stats/modes", app.handleModes)
	mux.HandleFunc("/stats/timeline/", app.handleTimelineByCountry)
	mux.HandleFunc("/stats/assigned-country", app.handleAssignedCountry)

	server := &http.Server{
		Addr:              ":" + statsPort,
		Handler:           logMiddleware(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("go-stats-api escuchando en http://localhost:%s", statsPort)
	log.Printf("País asignado configurado: %s", assignedCountry)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("No se pudo iniciar go-stats-api: %v", err)
	}
}

func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, APIResponse{
		Status: "success",
		Data: map[string]string{
			"service": "go-stats-api",
			"status":  "ok",
		},
	})
}

func (a *App) handleAssignedCountry(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	totalByCountry, _ := a.rdb.HGet(ctx, "stats:reports_by_country", a.assignedCountry).Int64()

	writeJSON(w, http.StatusOK, APIResponse{
		Status: "success",
		Data: map[string]interface{}{
			"assigned_country": a.assignedCountry,
			"total_reports":    totalByCountry,
		},
	})
}

func (a *App) handleSummary(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	totalReports, _ := a.rdb.Get(ctx, "stats:total_reports").Int64()
	assignedCountryTotal, _ := a.rdb.HGet(ctx, "stats:reports_by_country", a.assignedCountry).Int64()
	maxWarplanes, _ := a.rdb.Get(ctx, "stats:max_warplanes").Int64()
	minWarplanes, _ := a.rdb.Get(ctx, "stats:min_warplanes").Int64()
	maxWarships, _ := a.rdb.Get(ctx, "stats:max_warships").Int64()
	minWarships, _ := a.rdb.Get(ctx, "stats:min_warships").Int64()

	resp := SummaryResponse{
		AssignedCountry:      a.assignedCountry,
		TotalReports:         totalReports,
		AssignedCountryTotal: assignedCountryTotal,
		MaxWarplanes:         maxWarplanes,
		MinWarplanes:         minWarplanes,
		MaxWarships:          maxWarships,
		MinWarships:          minWarships,
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Status: "success",
		Data:   resp,
	})
}

func (a *App) handleTopWarplanes(w http.ResponseWriter, r *http.Request) {
	a.handleTopZSet(w, "leaderboard:warplanes_by_country")
}

func (a *App) handleTopWarships(w http.ResponseWriter, r *http.Request) {
	a.handleTopZSet(w, "leaderboard:warships_by_country")
}

func (a *App) handleTopZSet(w http.ResponseWriter, key string) {
	ctx := context.Background()

	results, err := a.rdb.ZRevRangeWithScores(ctx, key, 0, 4).Result()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{
			Status: "error",
			Error:  err.Error(),
		})
		return
	}

	var items []RankedCountry
	for _, item := range results {
		country, _ := item.Member.(string)
		items = append(items, RankedCountry{
			Country: country,
			Score:   item.Score,
		})
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Status: "success",
		Data:   items,
	})
}

func (a *App) handleModes(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	warplanesHist, err := a.rdb.HGetAll(ctx, "histogram:warplanes").Result()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Status: "error", Error: err.Error()})
		return
	}

	warshipsHist, err := a.rdb.HGetAll(ctx, "histogram:warships").Result()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Status: "error", Error: err.Error()})
		return
	}

	wpValue, wpCount := computeMode(warplanesHist)
	wsValue, wsCount := computeMode(warshipsHist)

	var resp ModeResponse
	resp.Warplanes.Value = wpValue
	resp.Warplanes.Count = wpCount
	resp.Warships.Value = wsValue
	resp.Warships.Count = wsCount

	writeJSON(w, http.StatusOK, APIResponse{
		Status: "success",
		Data:   resp,
	})
}

func computeMode(hist map[string]string) (int64, int64) {
	type pair struct {
		value int64
		count int64
	}

	var pairs []pair
	for k, v := range hist {
		value, err1 := strconv.ParseInt(k, 10, 64)
		count, err2 := strconv.ParseInt(v, 10, 64)
		if err1 == nil && err2 == nil {
			pairs = append(pairs, pair{value: value, count: count})
		}
	}

	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].count == pairs[j].count {
			return pairs[i].value < pairs[j].value
		}
		return pairs[i].count > pairs[j].count
	})

	if len(pairs) == 0 {
		return 0, 0
	}

	return pairs[0].value, pairs[0].count
}

func (a *App) handleTimelineByCountry(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	prefix := "/stats/timeline/"
	country := r.URL.Path[len(prefix):]
	if country == "" {
		country = a.assignedCountry
	}

	results, err := a.rdb.ZRangeWithScores(ctx, "timeline:"+country, 0, 99).Result()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{
			Status: "error",
			Error:  err.Error(),
		})
		return
	}

	points := make([]TimelinePoint, 0, len(results))
	for _, item := range results {
		raw, ok := item.Member.(string)
		if !ok {
			continue
		}

		var report RawReport
		if err := json.Unmarshal([]byte(raw), &report); err != nil {
			continue
		}

		points = append(points, TimelinePoint{
			Timestamp:       report.Timestamp,
			Unix:            int64(item.Score),
			WarplanesInAir:  report.WarplanesInAir,
			WarshipsInWater: report.WarshipsInWater,
			Country:         report.Country,
		})
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Status: "success",
		Data: map[string]interface{}{
			"country": country,
			"points":  points,
		},
	})
}

func writeJSON(w http.ResponseWriter, statusCode int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(body)
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}