package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   amqp.Queue
	rdb     *redis.Client
}

func NewConsumer(rabbitURL string, valkeyURL string) (*Consumer, error) {
	// Conexión a RabbitMQ
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

	// Conexión a Valkey
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

	return &Consumer{
		conn:    conn,
		channel: ch,
		queue:   q,
		rdb:     rdb,
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
	log.Println("AGGREGATES_V2_ACTIVO")
	log.Printf("Consumidor y Valkey listos. Esperando mensajes en la cola '%s'...", c.queue.Name)

	forever := make(chan struct{})

	go func() {
		for d := range msgs {
			ctx := context.Background()

			var msg WarReportMessage
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				log.Printf("Error al parsear mensaje: %v | body=%s", err, string(d.Body))
				_ = d.Nack(false, false)
				continue
			}

			err := c.storeMessage(ctx, msg, string(d.Body))
			if err != nil {
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

func (c *Consumer) storeMessage(ctx context.Context, msg WarReportMessage, raw string) error {
	pipe := c.rdb.TxPipeline()

	// Historial general y por país
	pipe.LPush(ctx, "reports", raw)
	pipe.LPush(ctx, "reports:country:"+msg.Country, raw)

	// Conteos
	pipe.Incr(ctx, "stats:total_reports")
	pipe.HIncrBy(ctx, "stats:reports_by_country", msg.Country, 1)

	// Ranking por acumulado de aviones y barcos
	pipe.ZIncrBy(ctx, "leaderboard:warplanes_by_country", float64(msg.WarplanesInAir), msg.Country)
	pipe.ZIncrBy(ctx, "leaderboard:warships_by_country", float64(msg.WarshipsInWater), msg.Country)

	// Histogramas para moda
	pipe.HIncrBy(ctx, "histogram:warplanes", fmt.Sprintf("%d", msg.WarplanesInAir), 1)
	pipe.HIncrBy(ctx, "histogram:warships", fmt.Sprintf("%d", msg.WarshipsInWater), 1)

	// Serie temporal por país
	parsedTime, err := time.Parse(time.RFC3339, msg.Timestamp)
	if err == nil {
		pipe.ZAdd(ctx, "timeline:"+msg.Country, redis.Z{
			Score:  float64(parsedTime.Unix()),
			Member: raw,
		})
	}

	_, execErr := pipe.Exec(ctx)
	if execErr != nil {
		return execErr
	}

	// Máximos y mínimos globales
	if err := c.updateMinMax(ctx, msg); err != nil {
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