package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/dmytrosurovtsev/eckwmsgo/internal/models"
	"github.com/gorilla/mux"
)

// listOrders returns all orders (RMA + repairs)
func (r *Router) listOrders(w http.ResponseWriter, req *http.Request) {
	// Filter by order type if specified
	orderType := req.URL.Query().Get("type")

	var orders []models.Order
	query := r.db.DB

	if orderType != "" {
		query = query.Where("order_type = ?", orderType)
	}

	if err := query.Find(&orders).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch orders")
		return
	}

	respondJSON(w, http.StatusOK, orders)
}

// getOrder returns a single order by ID
func (r *Router) getOrder(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid order ID")
		return
	}

	var order models.Order
	if err := r.db.First(&order, id).Error; err != nil {
		respondError(w, http.StatusNotFound, "Order not found")
		return
	}

	respondJSON(w, http.StatusOK, order)
}

// createOrder creates a new order (RMA or repair)
func (r *Router) createOrder(w http.ResponseWriter, req *http.Request) {
	var order models.Order
	if err := json.NewDecoder(req.Body).Decode(&order); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Set order type if not specified
	if order.OrderType == "" {
		order.OrderType = models.OrderTypeRMA
	}

	if err := r.db.Create(&order).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create order")
		return
	}

	respondJSON(w, http.StatusCreated, order)
}

// updateOrder updates an existing order
func (r *Router) updateOrder(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid order ID")
		return
	}

	var order models.Order
	if err := r.db.First(&order, id).Error; err != nil {
		respondError(w, http.StatusNotFound, "Order not found")
		return
	}

	if err := json.NewDecoder(req.Body).Decode(&order); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := r.db.Save(&order).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update order")
		return
	}

	respondJSON(w, http.StatusOK, order)
}

// deleteOrder deletes an order
func (r *Router) deleteOrder(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid order ID")
		return
	}

	if err := r.db.Delete(&models.Order{}, id).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete order")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Order deleted successfully",
	})
}

// Legacy RMA endpoints (redirect to orders)
func (r *Router) listRMAs(w http.ResponseWriter, req *http.Request) {
	req.URL.Query().Set("type", "rma")
	r.listOrders(w, req)
}

func (r *Router) getRMA(w http.ResponseWriter, req *http.Request) {
	r.getOrder(w, req)
}

func (r *Router) createRMA(w http.ResponseWriter, req *http.Request) {
	var input map[string]interface{}
	if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	input["order_type"] = "rma"
	r.createOrder(w, req)
}

func (r *Router) updateRMA(w http.ResponseWriter, req *http.Request) {
	r.updateOrder(w, req)
}

func (r *Router) deleteRMA(w http.ResponseWriter, req *http.Request) {
	r.deleteOrder(w, req)
}
