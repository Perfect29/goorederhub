# GoorederHub

Microservice in Go for simple order management.  
Uses PostgreSQL, Redis and Kafka.

## Run

Before starting the server, set your DB connection string:

```bash
export DB_CONN="postgres://postgres:yourpass@localhost:5432/goorderhub?sslmode=disable"

# Create order
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"product": "Laptop", "quantity": 2}'

# Get order
curl http://localhost:8080/orders/get/1

# Cancel order
curl -X POST http://localhost:8080/orders/cancel/1
