package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"

	"github.com/jmoiron/sqlx"
	amqp "github.com/rabbitmq/amqp091-go"
	payment "tx-api"
	"tx-api/service"
)

const (
	defaultRabbitMQURL = "amqp://guest:guest@localhost/"
	defaultPostgresURL = "postgres://postgres:postgres@localhost?sslmode=disable"
)

type Worker struct {
	s          PaymentService
	rabbitConn *amqp.Connection
}

func NewWorker(s PaymentService, rabbitConn *amqp.Connection) *Worker {
	return &Worker{
		s:          s,
		rabbitConn: rabbitConn,
	}
}

type PaymentService interface {
	MakePayment(ctx context.Context, newPayment *payment.Payment) error
}

func (w *Worker) ConsumePayments(ctx context.Context) error {
	ch, err := w.rabbitConn.Channel()
	if err != nil {
		return err
	}

	q, err := ch.QueueDeclare(
		"payments",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		return err
	}

	go func() {
		log.Println("Start consuming payments")
		for m := range msgs {
			var p payment.Payment
			if err := json.Unmarshal(m.Body, &p); err != nil {
				log.Printf("Failed to unmarshal payment with id %s error %s\n", p.ID, err)

				// we use manually ack to maintain payments messages safety
				_ = m.Ack(false)
				continue
			}

			if err := w.s.MakePayment(ctx, &p); err != nil {
				log.Printf("Failed to make payment with id %s error %s\n", p.ID, err)
			} else {
				log.Printf("successful make payment %s\n", p.ID)
			}
			_ = m.Ack(false)
		}
	}()

	return nil
}

func main() {
	rabbitMQURL := os.Getenv("RABBIT_MQ_URL")
	if rabbitMQURL == "" {
		rabbitMQURL = defaultRabbitMQURL
	}

	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		postgresURL = defaultPostgresURL
	}

	db, err := sqlx.Connect("postgres", postgresURL)
	if err != nil {
		log.Fatal("Failed to connect to db with error", err)
	}

	// apply migrations through the code
	_, err = db.Exec(`create table if not exists users
(
    id      varchar,
    balance int
);

`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`create table if not exists transactions
(
    id      varchar,
    user_id varchar,
    amount int,
    tx_type varchar
);

`)
	if err != nil {
		log.Fatal(err)
	}

	conn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		log.Fatal("Failed to connect to rabbit mq error", err)
	}

	s := service.NewService(db)

	consumer := NewWorker(s, conn)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := consumer.ConsumePayments(ctx); err != nil {
		log.Fatal(err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
}
