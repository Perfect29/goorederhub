package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"strconv"

	"goorderhub/internal/model"
	"goorderhub/internal/service"
)

func CreateOrderHandler(orderService *service.OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateOrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		order := model.Order{
			Product:  req.Product,
			Quantity: req.Quantity,
			Status:   "Created",
		}

		id := orderService.CreateOrder(order)

		resp := CreateOrderResponse{
			ID:     id,
			Status: "Created",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}


func GetOrderHandler(orderService *service.OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) != 4 {
			http.Error(w, "Invalid URL", http.StatusBadRequest)
			return
		}
		idStr := parts[3]
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid Order ID", http.StatusBadRequest)
			return 
		}

		order, found := orderService.GetOrder(id)

		if !found {
			http.Error(w, "Order not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(order)
	}
}

func CancelOrderHandler(orderService *service.OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		parts := strings.Split(r.URL.Path, "/")

		if len(parts) != 4 || parts[2] != "cancel" { 
			http.Error(w, "Invalid URL", http.StatusBadRequest)
			return
		}

		idStr := parts[3]
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid Order ID", http.StatusBadRequest)
			return 
		}

		found := orderService.CancelOrder(id)

		if !found {
			http.Error(w, "Order not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "Canceled"})
	}
}

