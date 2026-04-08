package mq

import (
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/jibiao-ai/deliverydesk/internal/config"
	"github.com/jibiao-ai/deliverydesk/pkg/logger"
)

const (
	QueueAgentTask = "deliverydesk.agent.task"
)

type TaskMessage struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	AgentID uint   `json:"agent_id"`
	UserID  uint   `json:"user_id"`
	Payload string `json:"payload"`
}

type RabbitMQ struct {
	cfg  config.RabbitMQConfig
	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewRabbitMQ(cfg config.RabbitMQConfig) *RabbitMQ {
	return &RabbitMQ{cfg: cfg}
}

func (r *RabbitMQ) Connect() error {
	url := fmt.Sprintf("amqp://%s:%s@%s:%d/%s",
		r.cfg.User, r.cfg.Password, r.cfg.Host, r.cfg.Port, r.cfg.VHost)

	var err error
	for i := 0; i < 10; i++ {
		r.conn, err = amqp.Dial(url)
		if err == nil {
			break
		}
		logger.Log.Warnf("Failed to connect to RabbitMQ (attempt %d/10): %v", i+1, err)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	r.ch, err = r.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}

	_, err = r.ch.QueueDeclare(QueueAgentTask, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	logger.Log.Info("RabbitMQ connected")
	return nil
}

func (r *RabbitMQ) Close() {
	if r.ch != nil {
		r.ch.Close()
	}
	if r.conn != nil {
		r.conn.Close()
	}
}

func (r *RabbitMQ) Publish(queue string, msg TaskMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return r.ch.Publish("", queue, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}

func (r *RabbitMQ) Consume(queue string, handler func(TaskMessage) error) {
	msgs, err := r.ch.Consume(queue, "", true, false, false, false, nil)
	if err != nil {
		logger.Log.Errorf("Failed to consume queue %s: %v", queue, err)
		return
	}

	go func() {
		for d := range msgs {
			var msg TaskMessage
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				logger.Log.Warnf("Invalid task message: %v", err)
				continue
			}
			if err := handler(msg); err != nil {
				logger.Log.Errorf("Task handler error: %v", err)
			}
		}
	}()

	logger.Log.Infof("Consuming queue: %s", queue)
}
