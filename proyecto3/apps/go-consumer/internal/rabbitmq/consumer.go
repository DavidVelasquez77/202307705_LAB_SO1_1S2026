package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

type WarReportMessage struct {
	Country         string `json:"country"`
	WarplanesInAir  int32  `json:"warplanes_in_air"`
	WarshipsInWater int32  `json:"warships_in_water"`
	Timestamp       string `json:"timestamp"`
}

type Consumer struct {
	conn            *amqp.Connection
	channel         *amqp.Channel
	queue           amqp.Queue
	rdb             *redis.Client
	assignedCountry string
	processDelay    time.Duration
}

func NewConsumer(rabbitURL string, valkeyURL string) (*Consumer, error) {
	assignedCountry := os.Getenv("ASSIGNED_COUNTRY")
	if assignedCountry == "" {
		assignedCountry = "CHN"
	}

	processDelay := parseProcessDelay(os.Getenv("CONSUMER_PROCESS_DELAY_MS"))

	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	q, err := ch.QueueDeclare(
		"war_reports",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}

	opt, err := redis.ParseURL(valkeyURL)
	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}

	rdb := redis.NewClient(opt)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		_ = rdb.Close()
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}

	if err := ch.Qos(1, 0, false); err != nil {
		_ = rdb.Close()
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}

	return &Consumer{
		conn:            conn,
		channel:         ch,
		queue:           q,
		rdb:             rdb,
		assignedCountry: assignedCountry,
		processDelay:    processDelay,
	}, nil
}

func (c *Consumer) Close() {
	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
	if c.rdb != nil {
		_ = c.rdb.Close()
	}
}

func (c *Consumer) StartConsuming() error {
	msgs, err := c.channel.Consume(
		c.queue.Name,
		"",
		false, // ACK manual
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	log.Printf(
		"Consumidor listo. Esperando mensajes en '%s'. País asignado para dashboard: %s. Delay por mensaje: %s",
		c.queue.Name,
		c.assignedCountry,
		c.processDelay,
	)

	forever := make(chan struct{})

	go func() {
		for d := range msgs {
			ctx := context.Background()

			if c.processDelay > 0 {
				time.Sleep(c.processDelay)
			}

			var msg WarReportMessage
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				log.Printf("Error al parsear mensaje: %v | body=%s", err, string(d.Body))
				_ = d.Nack(false, false)
				continue
			}

			if err := c.storeMessage(ctx, msg, string(d.Body)); err != nil {
				log.Printf("Error guardando en Valkey: %v", err)
				_ = d.Nack(false, true)
				continue
			}

			log.Printf(
				"Guardado en Valkey -> country=%s, warplanes=%d, warships=%d, timestamp=%s",
				msg.Country,
				msg.WarplanesInAir,
				msg.WarshipsInWater,
				msg.Timestamp,
			)

			if err := d.Ack(false); err != nil {
				log.Printf("Error haciendo ACK: %v", err)
			}
		}
	}()

	<-forever
	return nil
}

func parseProcessDelay(raw string) time.Duration {
	if raw == "" {
		return 0
	}

	ms, err := strconv.Atoi(raw)
	if err != nil || ms < 0 {
		log.Printf("CONSUMER_PROCESS_DELAY_MS inválido (%q), usando 0ms", raw)
		return 0
	}

	return time.Duration(ms) * time.Millisecond
}

func (c *Consumer) storeMessage(ctx context.Context, msg WarReportMessage, raw string) error {
	pipe := c.rdb.TxPipeline()

	// Guardamos el país asignado como llave simple para Grafana
	pipe.Set(ctx, "dashboard:assigned_country_name", c.assignedCountry, 0)

	// Historial general y por país
	pipe.LPush(ctx, "reports", raw)
	pipe.LPush(ctx, "reports:country:"+msg.Country, raw)

	// Conteos
	pipe.Incr(ctx, "stats:total_reports")
	pipe.HIncrBy(ctx, "stats:reports_by_country", msg.Country, 1)

	// Rankings acumulados
	pipe.ZIncrBy(ctx, "leaderboard:warplanes_by_country", float64(msg.WarplanesInAir), msg.Country)
	pipe.ZIncrBy(ctx, "leaderboard:warships_by_country", float64(msg.WarshipsInWater), msg.Country)

	// Histogramas
	pipe.HIncrBy(ctx, "histogram:warplanes", fmt.Sprintf("%d", msg.WarplanesInAir), 1)
	pipe.HIncrBy(ctx, "histogram:warships", fmt.Sprintf("%d", msg.WarshipsInWater), 1)

	// Timeline por país
	parsedTime, err := time.Parse(time.RFC3339, msg.Timestamp)
	if err == nil {
		unixMs := parsedTime.UnixMilli()

		pipe.XAdd(ctx, &redis.XAddArgs{
			Stream: fmt.Sprintf("stream:timeline:%s", msg.Country),
			ID:     "*",
			Values: map[string]interface{}{
				"timestamp":         unixMs,
				"country":           msg.Country,
				"warplanes_in_air":  msg.WarplanesInAir,
				"warships_in_water": msg.WarshipsInWater,
			},
		})
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}

	if err := c.updateMinMax(ctx, msg); err != nil {
		return err
	}

	if err := c.updateDashboardKeys(ctx); err != nil {
		return err
	}

	return nil
}

func (c *Consumer) updateMinMax(ctx context.Context, msg WarReportMessage) error {
	if err := updateMax(ctx, c.rdb, "stats:max_warplanes", msg.WarplanesInAir); err != nil {
		return err
	}
	if err := updateMin(ctx, c.rdb, "stats:min_warplanes", msg.WarplanesInAir); err != nil {
		return err
	}
	if err := updateMax(ctx, c.rdb, "stats:max_warships", msg.WarshipsInWater); err != nil {
		return err
	}
	if err := updateMin(ctx, c.rdb, "stats:min_warships", msg.WarshipsInWater); err != nil {
		return err
	}
	return nil
}

func (c *Consumer) updateDashboardKeys(ctx context.Context) error {
	pipe := c.rdb.TxPipeline()

	// Espejo de totales y min/max en llaves simples
	totalReports, _ := c.rdb.Get(ctx, "stats:total_reports").Int64()
	assignedTotal, _ := c.rdb.HGet(ctx, "stats:reports_by_country", c.assignedCountry).Int64()
	maxWarplanes, _ := c.rdb.Get(ctx, "stats:max_warplanes").Int64()
	minWarplanes, _ := c.rdb.Get(ctx, "stats:min_warplanes").Int64()
	maxWarships, _ := c.rdb.Get(ctx, "stats:max_warships").Int64()
	minWarships, _ := c.rdb.Get(ctx, "stats:min_warships").Int64()

	pipe.Set(ctx, "dashboard:total_reports", totalReports, 0)
	pipe.Set(ctx, "dashboard:assigned_country_total_reports", assignedTotal, 0)
	pipe.Set(ctx, "dashboard:max_warplanes", maxWarplanes, 0)
	pipe.Set(ctx, "dashboard:min_warplanes", minWarplanes, 0)
	pipe.Set(ctx, "dashboard:max_warships", maxWarships, 0)
	pipe.Set(ctx, "dashboard:min_warships", minWarships, 0)

	// Top países por aviones y barcos
	topWarplanes, _ := c.rdb.ZRevRangeWithScores(ctx, "leaderboard:warplanes_by_country", 0, 0).Result()
	if len(topWarplanes) > 0 {
		if country, ok := topWarplanes[0].Member.(string); ok {
			pipe.Set(ctx, "dashboard:top_warplanes_country", country, 0)
			pipe.Set(ctx, "dashboard:top_warplanes_score", int64(topWarplanes[0].Score), 0)
		}
	}

	topWarships, _ := c.rdb.ZRevRangeWithScores(ctx, "leaderboard:warships_by_country", 0, 0).Result()
	if len(topWarships) > 0 {
		if country, ok := topWarships[0].Member.(string); ok {
			pipe.Set(ctx, "dashboard:top_warships_country", country, 0)
			pipe.Set(ctx, "dashboard:top_warships_score", int64(topWarships[0].Score), 0)
		}
	}

	// Moda
	warplanesHist, _ := c.rdb.HGetAll(ctx, "histogram:warplanes").Result()
	warshipsHist, _ := c.rdb.HGetAll(ctx, "histogram:warships").Result()

	wpValue, wpCount := computeMode(warplanesHist)
	wsValue, wsCount := computeMode(warshipsHist)

	pipe.Set(ctx, "dashboard:mode_warplanes_value", wpValue, 0)
	pipe.Set(ctx, "dashboard:mode_warplanes_count", wpCount, 0)
	pipe.Set(ctx, "dashboard:mode_warships_value", wsValue, 0)
	pipe.Set(ctx, "dashboard:mode_warships_count", wsCount, 0)

	_, err := pipe.Exec(ctx)
	return err
}

func computeMode(hist map[string]string) (int64, int64) {
	type pair struct {
		value int64
		count int64
	}

	var best pair
	first := true

	for k, v := range hist {
		value, err1 := strconv.ParseInt(k, 10, 64)
		count, err2 := strconv.ParseInt(v, 10, 64)
		if err1 != nil || err2 != nil {
			continue
		}

		if first || count > best.count || (count == best.count && value < best.value) {
			best = pair{value: value, count: count}
			first = false
		}
	}

	return best.value, best.count
}

func updateMax(ctx context.Context, rdb *redis.Client, key string, value int32) error {
	current, err := rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return rdb.Set(ctx, key, value, 0).Err()
	}
	if err != nil {
		return err
	}

	currentInt, convErr := strconv.Atoi(current)
	if convErr != nil {
		return convErr
	}

	if int(value) > currentInt {
		return rdb.Set(ctx, key, value, 0).Err()
	}
	return nil
}

func updateMin(ctx context.Context, rdb *redis.Client, key string, value int32) error {
	current, err := rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return rdb.Set(ctx, key, value, 0).Err()
	}
	if err != nil {
		return err
	}

	currentInt, convErr := strconv.Atoi(current)
	if convErr != nil {
		return convErr
	}

	if int(value) < currentInt {
		return rdb.Set(ctx, key, value, 0).Err()
	}
	return nil
}
