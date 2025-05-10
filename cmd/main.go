package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/redis/go-redis/v9"
	"goorderhub/internal/api"
	"goorderhub/internal/database"
	"goorderhub/internal/service"
)

func main() {
	// DB init
	connStr := os.Getenv("DB_CONN")
	if connStr == "" {
		log.Fatal("DB_CONN is not set")
	}

	db, err := database.ConnectDB(connStr)
	if err != nil {
		log.Fatal("Failed to connect to DB:", err)
	}
	defer db.Close()

	// Kafka producer
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
	})
	if err != nil {
		log.Fatalf("Kafka producer error: %s", err)
	}
	defer producer.Close()

	// Kafka delivery handler
	go func() {
		for e := range producer.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					log.Printf("Delivery failed: %v\n", ev.TopicPartition.Error)
				} else {
					log.Printf("Delivered message to %v\n", ev.TopicPartition)
				}
			}
		}
	}()

	// Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	if _, err := rdb.Ping(context.Background()).Result(); err != nil {
		log.Fatalf("Redis connection failed: %v", err)
	}
	defer rdb.Close()
	fmt.Println("Connected to Redis")

	// Service
	orderService := service.NewOrderService(db, producer, "orders", rdb)

	// Kafka consumers
	for i := 1; i <= 3; i++ {
		go service.StartKafkaWorker(db, "orders", "localhost:9092", i)
	}

	// API
	mux := http.NewServeMux()
	mux.HandleFunc("/orders", api.CreateOrderHandler(orderService))
	mux.HandleFunc("/orders/get/", api.GetOrderHandler(orderService))
	mux.HandleFunc("/orders/cancel/", api.CancelOrderHandler(orderService))

	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// Shutdown handling
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Run server
	go func() {
		fmt.Println("Listening on http://localhost:8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	log.Println("Shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	log.Println("Server exited cleanly")
}
