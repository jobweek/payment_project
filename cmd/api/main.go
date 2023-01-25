package main

import (
	"context"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	amqp "github.com/rabbitmq/amqp091-go"
	"tx-api/service"
)

const (
	serverPort         = ":8088"
	defaultRabbitMQURL = "amqp://guest:guest@localhost/"
	defaultPostgresURL = "postgres://postgres:postgres@localhost?sslmode=disable"
)

func main() {
	addr := os.Getenv("SERVER_PORT")
	if addr == "" {
		addr = serverPort
	}

	rabbitMQURL := os.Getenv("RABBIT_MQ_URL")
	if rabbitMQURL == "" {
		rabbitMQURL = defaultRabbitMQURL
	}

	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		postgresURL = defaultPostgresURL
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	s := service.NewService(db)

	conn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		log.Fatal("Failed to connect to rabbit mq error", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
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
		log.Fatal(err)
	}

	server := NewServer(s, ch, q.Name)
	<-server.Run(ctx, addr)
}
