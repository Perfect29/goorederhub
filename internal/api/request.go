package api

type CreateOrderRequest struct {
	Product 	string `json:"product"`
	Quantity    int    `json:"quantity"`
}

type CreateOrderResponse struct {
	ID 		int 	`json:"id"`
	Status  string  `json:"status"`
}