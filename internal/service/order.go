package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"goorderhub/internal/model"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/redis/go-redis/v9"
)

type OrderService struct {
	db       *sql.DB
	producer *kafka.Producer
	topic    string
	cache    *redis.Client
}

var c = context.Background()

func NewOrderService(db *sql.DB, producer *kafka.Producer, topic string, cache *redis.Client) *OrderService {
	return &OrderService{
		db:       db,
		producer: producer,
		topic:    topic,
		cache:    cache,
	}
}

func (x *OrderService) CreateOrder(order model.Order) int {
	var id int

	err := x.db.QueryRow(`
		INSERT INTO orders (product, quantity, status)
		VALUES ($1, $2, $3)
		RETURNING id
	`, order.Product, order.Quantity, order.Status).Scan(&id)

	if err != nil {
		fmt.Printf("Failed to create order: %v\n", err)
		return 0
	}

	message := fmt.Sprintf(`{"order_id": %d}`, id)
	x.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &x.topic, Partition: kafka.PartitionAny},
		Value:          []byte(message),
	}, nil)

	return id
}

func StartKafkaWorker(db *sql.DB, topic, broker string, workerID int) {
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": broker,
		"group.id":          "order-group",
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		log.Fatalf("Failed to create Kafka consumer: %v", err)
	}
	defer consumer.Close()

	if err := consumer.SubscribeTopics([]string{topic}, nil); err != nil {
		log.Fatalf("Failed to subscribe to topic: %v", err)
	}

	for {
		msg, err := consumer.ReadMessage(-1)
		if err != nil {
			fmt.Printf("Consumer error: %v\n", err)
			continue
		}

		var payload struct {
			OrderID int `json:"order_id"`
		}
		if err := json.Unmarshal(msg.Value, &payload); err != nil {
			fmt.Printf("Failed to parse message: %v\n", err)
			continue
		}

		fmt.Printf("[Worker %d] Processing order %d\n", workerID, payload.OrderID)

		_, err = db.Exec(`UPDATE orders SET status = 'Processed' WHERE id = $1`, payload.OrderID)
		if err != nil {
			fmt.Printf("Failed to update order %d: %v\n", payload.OrderID, err)
		}
	}
}

func (x *OrderService) GetOrder(id int) (model.Order, bool) {
	var order model.Order
	
	// checking Redis

	key := fmt.Sprintf("order:%d", id)
	val, err := x.cache.Get(c, key).Result()

	if err == nil {
		if err := json.Unmarshal([]byte(val), &order); err == nil {
			fmt.Println("Cache hit!")
			return order, true
		}
	}

	err = x.db.QueryRow(`
		SELECT * FROM orders 
		WHERE id = $1`, id).Scan(
		&order.ID, &order.Product, &order.Quantity, &order.Status,
	)
	if err != nil {
		fmt.Printf("Failed to get order %d: %v\n", id, err)
		return order, false
	}

	bytes, _ := json.Marshal(order)
	x.cache.Set(c, key, bytes, 5*time.Minute)

	fmt.Println("Cache miss. Loaded from DB and cached")
	return order, true
}

func (x *OrderService) CancelOrder(id int) bool {
	result, err := x.db.Exec(`
		UPDATE orders 
		SET status = 'Canceled'
		WHERE id = $1
	`, id)

	if err != nil {
		fmt.Printf("Failed to cancel order %d: %v\n", id, err)
		return false
	}

	key := fmt.Sprintf("order:%d", id)
	x.cache.Del(c, key)

	rows, err := result.RowsAffected()
	if err != nil {
		fmt.Printf("Unable to read order %d: %v\n", id, err)
		return false
	}
	return rows > 0
}
