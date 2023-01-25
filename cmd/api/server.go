package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	amqp "github.com/rabbitmq/amqp091-go"
	payment "tx-api"
)

type paymentRequest struct {
	UserID string `json:"user_id"`
	Amount int    `json:"amount"`
	TxType string `json:"tx_type"`
}

type Server struct {
	e          *echo.Echo
	service    Service
	paymentsCh *amqp.Channel
	queueName  string
}

func NewServer(s Service, ch *amqp.Channel, queueName string) *Server {
	server := &Server{
		e:          echo.New(),
		service:    s,
		paymentsCh: ch,
		queueName:  queueName,
	}

	server.registerEndpoint()
	return server
}

func (s *Server) registerEndpoint() {
	s.e.GET("/v1.0/user", s.User)
	s.e.GET("/v1.0/payment", s.Payment)
	s.e.POST("/v1.0/payment", s.MakePayment)
}

func (s *Server) Run(ctx context.Context, addr string) <-chan struct{} {
	go func() {
		if err := s.e.Start(addr); err != nil {
			log.Fatal(err)
		}
	}()

	finish := make(chan struct{}, 1)
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt)
		<-quit
		ctx, cancel := context.WithTimeout(ctx, time.Second*30)
		defer cancel()
		if err := s.e.Shutdown(ctx); err != nil {
			log.Fatal(err)
		}
		finish <- struct{}{}
	}()

	return finish
}

func (s *Server) User(c echo.Context) error {
	ctx := c.Request().Context()
	userID := c.QueryParam("id")
	if userID == "" {
		c.Response().Writer.WriteHeader(http.StatusBadRequest)
		return nil
	}

	u, err := s.service.User(ctx, userID)
	if err != nil {
		return respondWithJSON(c, http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
	}

	return respondWithJSON(c, http.StatusOK, u)
}

func (s *Server) Payment(c echo.Context) error {
	ctx := c.Request().Context()
	paymentID := c.QueryParam("id")
	if paymentID == "" {
		c.Response().Writer.WriteHeader(http.StatusBadRequest)
		return nil
	}

	p, err := s.service.Payment(ctx, paymentID)
	if err != nil {
		return respondWithJSON(c, http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
	}

	return respondWithJSON(c, http.StatusOK, p)
}

func (s *Server) MakePayment(c echo.Context) error {
	ctx := c.Request().Context()
	var req paymentRequest
	if err := c.Bind(&req); err != nil {
		return respondWithJSON(c, http.StatusBadGateway, map[string]interface{}{"error": err.Error()})
	}

	if req.Amount == 0 {
		return respondWithJSON(c, http.StatusBadGateway, map[string]interface{}{"error": "amount must be greater than 0"})
	}

	if req.TxType != payment.Deposit && req.TxType != payment.WithdrawTx {
		return respondWithJSON(c, http.StatusBadGateway, map[string]interface{}{"error": "unsupported tx type"})
	}

	newPayment := payment.Payment{
		ID:     uuid.New().String(),
		UserID: req.UserID,
		Amount: req.Amount,
		TxType: req.TxType,
	}

	bytes, err := json.Marshal(&newPayment)
	if err != nil {
		return err
	}

	err = s.paymentsCh.PublishWithContext(ctx,
		"",          // exchange
		s.queueName, // routing key
		false,       // mandatory
		false,       // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        bytes,
		})
	if err != nil {
		return respondWithJSON(c, http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
	}

	return respondWithJSON(c, http.StatusCreated, map[string]interface{}{"payment_id": newPayment.ID})
}

func respondWithJSON(c echo.Context, code int, v interface{}) error {
	bytes, err := json.Marshal(v)
	if err != nil {
		return err
	}

	c.Response().WriteHeader(code)
	_, err = c.Response().Write(bytes)
	return err
}
