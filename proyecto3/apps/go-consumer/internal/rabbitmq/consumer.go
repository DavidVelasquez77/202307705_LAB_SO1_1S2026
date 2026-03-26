package rabbitmq

import (
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
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
}

func NewConsumer(url string) (*Consumer, error) {
	conn, err := amqp.Dial(url)
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

	return &Consumer{
		conn:    conn,
		channel: ch,
		queue:   q,
	}, nil
}

func (c *Consumer) Close() {
	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
}

func (c *Consumer) StartConsuming() error {
	msgs, err := c.channel.Consume(
		c.queue.Name,
		"",
		false, // auto-ack desactivado
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	log.Printf("Consumidor listo. Esperando mensajes en la cola '%s'...", c.queue.Name)

	forever := make(chan struct{})

	go func() {
		for d := range msgs {
			var msg WarReportMessage

			if err := json.Unmarshal(d.Body, &msg); err != nil {
				log.Printf("Error al parsear mensaje: %v | body=%s", err, string(d.Body))
				_ = d.Nack(false, false)
				continue
			}

			log.Printf(
				"Mensaje recibido de RabbitMQ -> country=%s, warplanes=%d, warships=%d, timestamp=%s",
				msg.Country,
				msg.WarplanesInAir,
				msg.WarshipsInWater,
				msg.Timestamp,
			)

			// Más adelante aquí guardaremos en Valkey

			if err := d.Ack(false); err != nil {
				log.Printf("Error haciendo ACK del mensaje: %v", err)
			}
		}
	}()

	<-forever
	return nil
}