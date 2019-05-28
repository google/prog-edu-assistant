// Package queue provides an interface to the work queue for the autograder instance.
package queue

import (
	"github.com/streadway/amqp"
)

// Channel represents a connection to the queue service.
type Channel struct {
	*amqp.Connection
	*amqp.Channel
	queues          map[string]amqp.Queue
	receiveChannels map[string]<-chan amqp.Delivery
}

// Open takes a string spec and opens a connection to the queue.
// Example of connection spec: "amqp://localhost:5672/".
func Open(spec string) (*Channel, error) {
	conn, err := amqp.Dial(spec)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}
	return &Channel{
		Connection: conn,
		Channel:    ch,
		queues:     make(map[string]amqp.Queue),
	}, nil
}

func (ch *Channel) Close() error {
	err1 := ch.Channel.Close()
	err2 := ch.Connection.Close()
	if err1 != nil {
		return err1
	}
	return err2
}

func (ch *Channel) getQueue(queueName string) (amqp.Queue, error) {
	q, ok := ch.queues[queueName]
	if !ok {
		var err error
		q, err = ch.Channel.QueueDeclare(
			queueName,
			false, // durable
			false, // delete when unused
			false, // exclusive
			false, // no-wait
			nil,   // extra arguments
		)
		if err != nil {
			return amqp.Queue{}, err
		}
		ch.queues[queueName] = q
	}
	return q, nil
}

func (ch *Channel) Post(queueName string, content []byte) error {
	q, err := ch.getQueue(queueName)
	if err != nil {
		return err
	}
	return ch.Channel.Publish(
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "application/octet-stream",
			Body:        content,
		})
}

// Receive returns a channel that will deliver received messages.
func (ch *Channel) Receive(queueName string) (<-chan []byte, error) {
	q, err := ch.getQueue(queueName)
	if err != nil {
		return nil, err
	}
	deliveries, err := ch.Channel.Consume(
		q.Name,
		"",    // consumer
		true,  // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // extra args
	)
	if err != nil {
		return nil, err
	}
	outputCh := make(chan []byte)
	go func() {
		for d := range deliveries {
			outputCh <- d.Body
		}
		close(outputCh)
	}()
	return outputCh, nil
}
