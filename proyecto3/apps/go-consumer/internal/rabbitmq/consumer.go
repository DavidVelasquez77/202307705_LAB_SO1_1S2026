package rabbitmq

import (
	"context"
	"encoding/json"
	"log"
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
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

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

	pipe.LPush(ctx, "reports", raw)
	pipe.LPush(ctx, "reports:country:"+msg.Country, raw)
	pipe.Incr(ctx, "stats:total_reports")
	pipe.HIncrBy(ctx, "stats:reports_by_country", msg.Country, 1)

	parsedTime, err := time.Parse(time.RFC3339, msg.Timestamp)
	if err == nil {
		pipe.ZAdd(ctx, "timeline:"+msg.Country, redis.Z{
			Score:  float64(parsedTime.Unix()),
			Member: raw,
		})
	}

	_, execErr := pipe.Exec(ctx)
	return execErr
}