package rabbitmq

import (
	"context"
	"encoding/json"
	"log"

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
	// 1. Conectar a RabbitMQ
	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	q, err := ch.QueueDeclare("war_reports", true, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}

	// 2. Conectar a Valkey
	opt, err := redis.ParseURL(valkeyURL)
	if err != nil {
		return nil, err
	}
	rdb := redis.NewClient(opt)

	// Comprobar que Valkey responde
	if err := rdb.Ping(context.Background()).Err(); err != nil {
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
	msgs, err := c.channel.Consume(c.queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	log.Printf(" Consumidor y Valkey listos. Esperando mensajes...")

	forever := make(chan struct{})

	go func() {
		ctx := context.Background()
		for d := range msgs {
			var msg WarReportMessage

			if err := json.Unmarshal(d.Body, &msg); err != nil {
				log.Printf(" Error al parsear mensaje: %v", err)
				_ = d.Nack(false, false)
				continue
			}

			// GUARDAR EN VALKEY (Usamos una Lista llamada "reports")
			err := c.rdb.LPush(ctx, "reports", string(d.Body)).Err()
			if err != nil {
				log.Printf(" Error guardando en Valkey: %v", err)
				// Si falla Valkey, devolvemos el mensaje a la cola para no perderlo
				_ = d.Nack(false, true)
				continue
			}

			log.Printf(" Guardado en Valkey: %s", msg.Country)

			// Si todo salió perfecto, le confirmamos a RabbitMQ
			if err := d.Ack(false); err != nil {
				log.Printf(" Error haciendo ACK: %v", err)
			}
		}
	}()

	<-forever
	return nil
}