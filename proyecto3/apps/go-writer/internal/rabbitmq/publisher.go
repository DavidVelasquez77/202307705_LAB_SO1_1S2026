package rabbitmq

import (
	"context"
	"encoding/json"
	"log"
	"time"

	wartweets "github.com/DavidVelasquez77/202307705_LAB_SO1_1S2026/proyecto3/proto/gen/go/wartweets"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   amqp.Queue
}

func NewPublisher(url string) (*Publisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	q, err := ch.QueueDeclare(
		"war_reports", true, false, false, false, nil,
	)
	if err != nil {
		return nil, err
	}

	return &Publisher{
		conn:    conn,
		channel: ch,
		queue:   q,
	}, nil
}

func (p *Publisher) Close() {
	p.channel.Close()
	p.conn.Close()
}

// Función auxiliar para normalizar el país a MAYÚSCULAS
func countryToCode(c wartweets.Countries) string {
	switch c {
	case wartweets.Countries_usa:
		return "USA"
	case wartweets.Countries_rus:
		return "RUS"
	case wartweets.Countries_chn:
		return "CHN"
	case wartweets.Countries_esp:
		return "ESP"
	case wartweets.Countries_gtm:
		return "GTM"
	default:
		return "UNKNOWN"
	}
}

func (p *Publisher) Publish(req *wartweets.WarReportRequest) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	body, err := json.Marshal(map[string]interface{}{
		"country":           countryToCode(req.GetCountry()), // <--- Usamos la normalización
		"warplanes_in_air":  req.GetWarplanesInAir(),
		"warships_in_water": req.GetWarshipsInWater(),
		"timestamp":         req.GetTimestamp(),
	})
	if err != nil {
		return err
	}

	err = p.channel.PublishWithContext(ctx,
		"", p.queue.Name, false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		})

	if err == nil {
		log.Printf(" Mensaje publicado en RabbitMQ: %s", string(body))
	}
	return err
}