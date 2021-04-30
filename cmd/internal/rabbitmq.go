package internal

import (
	"fmt"

	"github.com/streadway/amqp"

	"github.com/MarioCarrion/todo-api/internal/envvar"
)

// RabbitMQ ...
type RabbitMQ struct {
	Connection *amqp.Connection
	Channel    *amqp.Channel
}

// NewRabbitMQ instantiates the RabbitMQ instances using configuration defined in environment variables.
func NewRabbitMQ(conf *envvar.Configuration) (*RabbitMQ, error) {
	// XXX: Instead of using `RABBITMQ_URL` perhaps it makes sense to define
	// concrete `RABBIT_XYZ` variables similar to what we do for PostgreSQL to
	// clearly separate each field, like hostname, username, password, etc.
	url, err := conf.Get("RABBITMQ_URL")
	if err != nil {
		return nil, fmt.Errorf("conf.Get %w", err)
	}

	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("amqp.Dial %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("conn.Channel %w", err)
	}

	err = ch.ExchangeDeclare(
		"tasks", // name
		"topic", // type
		true,    // durable
		false,   // auto-deleted
		false,   // internal
		false,   // no-wait
		nil,     // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("ch.ExchangeDeclare %w", err)
	}

	if err := ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	); err != nil {
		return nil, fmt.Errorf("ch.Qos %w", err)
	}

	// XXX: Dead Letter Exchange will be implemented in future episodes

	return &RabbitMQ{
		Connection: conn,
		Channel:    ch,
	}, nil
}

// Close ...
func (r *RabbitMQ) Close() {
	r.Connection.Close()
}
