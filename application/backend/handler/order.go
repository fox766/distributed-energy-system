package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"backend/fabric"
	"backend/middleware"
	"backend/model"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type OrderHandler struct {
	gw *fabric.Gateway
}

func NewOrderHandler(gw *fabric.Gateway) *OrderHandler {
	return &OrderHandler{gw: gw}
}

// CreateOrder creates a new energy trade order on the ledger.
// If price is 0, the current time-of-use price is used automatically.
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req model.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "invalid request: " + err.Error()})
		return
	}
	if req.Direction == "" {
		req.Direction = "SELL"
	}

	// If price is not specified (0), use TOU price
	if req.Price <= 0 {
		touPrice := h.getTOUPrice()
		req.Price = touPrice
	}

	orderID := "energy_order_" + uuid.New().String()[:12]

	amountStr := strconv.FormatFloat(req.Amount, 'f', 2, 64)
	priceStr := strconv.FormatFloat(req.Price, 'f', 2, 64)
	feeStr := "0.01"

	_, err := h.gw.Contract.SubmitTransaction("CreateOrder",
		orderID, userID, req.Direction, req.EnergySource,
		req.DeliveryStart, req.DeliveryEnd,
		amountStr, priceStr, feeStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "failed to create order: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"orderid": orderID, "message": "order created"})
}

// GetOrder returns a single order by ID.
func (h *OrderHandler) GetOrder(c *gin.Context) {
	orderID := c.Param("id")

	result, err := h.gw.Contract.EvaluateTransaction("GetOrder", orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Error: "order not found: " + err.Error()})
		return
	}

	var order model.Order
	if err := json.Unmarshal(result, &order); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "failed to parse order"})
		return
	}
	c.JSON(http.StatusOK, order)
}

// ListOrders returns orders, optionally filtered by status.
func (h *OrderHandler) ListOrders(c *gin.Context) {
	status := c.DefaultQuery("status", "ALL")

	result, err := h.gw.Contract.EvaluateTransaction("GetAllOrders", status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "failed to list orders: " + err.Error()})
		return
	}

	// Handle empty result from chaincode
	if len(result) == 0 || string(result) == "null" {
		c.JSON(http.StatusOK, []model.Order{})
		return
	}

	var orders []model.Order
	if err := json.Unmarshal(result, &orders); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "failed to parse orders"})
		return
	}
	if orders == nil {
		orders = []model.Order{}
	}
	c.JSON(http.StatusOK, orders)
}

// ListMyOrders returns the authenticated user's orders.
func (h *OrderHandler) ListMyOrders(c *gin.Context) {
	userID := middleware.GetUserID(c)

	result, err := h.gw.Contract.EvaluateTransaction("GetUserOrders", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "failed to list orders: " + err.Error()})
		return
	}

	if len(result) == 0 || string(result) == "null" {
		c.JSON(http.StatusOK, []model.Order{})
		return
	}

	var orders []model.Order
	if err := json.Unmarshal(result, &orders); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "failed to parse orders"})
		return
	}
	if orders == nil {
		orders = []model.Order{}
	}
	c.JSON(http.StatusOK, orders)
}

// MatchOrder matches a CREATED order with the authenticated user as buyer.
func (h *OrderHandler) MatchOrder(c *gin.Context) {
	orderID := c.Param("id")
	userID := middleware.GetUserID(c)

	_, err := h.gw.Contract.SubmitTransaction("MatchOrder", orderID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "failed to match order: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.MessageResponse{Message: "order matched"})
}

// SettleOrder completes a MATCHED order. Only parties to the order can settle.
func (h *OrderHandler) SettleOrder(c *gin.Context) {
	orderID := c.Param("id")
	userID := middleware.GetUserID(c)

	// Verify ownership: only partyA or partyB can settle
	order, err := h.getOrderByID(orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Error: "order not found: " + err.Error()})
		return
	}
	if order.PartyA != userID && order.PartyB != userID {
		c.JSON(http.StatusForbidden, model.ErrorResponse{Error: "only parties to the order can settle it"})
		return
	}

	_, err = h.gw.Contract.SubmitTransaction("SettleOrder", orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "failed to settle order: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.MessageResponse{Message: "order settled"})
}

// CancelOrder cancels a CREATED order. Only the creator (partyA) can cancel.
func (h *OrderHandler) CancelOrder(c *gin.Context) {
	orderID := c.Param("id")
	userID := middleware.GetUserID(c)

	// Verify ownership: only partyA (creator) can cancel
	order, err := h.getOrderByID(orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Error: "order not found: " + err.Error()})
		return
	}
	if order.PartyA != userID {
		c.JSON(http.StatusForbidden, model.ErrorResponse{Error: "only the order creator can cancel it"})
		return
	}

	_, err = h.gw.Contract.SubmitTransaction("CancelOrder", orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "failed to cancel order: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.MessageResponse{Message: "order cancelled"})
}

// getOrderByID is a helper to read an order from chaincode.
func (h *OrderHandler) getOrderByID(orderID string) (*model.Order, error) {
	result, err := h.gw.Contract.EvaluateTransaction("GetOrder", orderID)
	if err != nil {
		return nil, err
	}
	var order model.Order
	if err := json.Unmarshal(result, &order); err != nil {
		return nil, err
	}
	return &order, nil
}

// AutoMatchOrder triggers automatic matching for a newly created order (P2).
func (h *OrderHandler) AutoMatchOrder(c *gin.Context) {
	orderID := c.Param("id")

	result, err := h.gw.Contract.SubmitTransaction("AutoMatchOrder", orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "auto-match failed: " + err.Error()})
		return
	}

	var matchResp model.AutoMatchResponse
	if err := json.Unmarshal(result, &matchResp); err != nil {
		c.JSON(http.StatusOK, model.AutoMatchResponse{Matched: false, Message: "failed to parse match result"})
		return
	}
	c.JSON(http.StatusOK, matchResp)
}

// RunAutoMatch triggers batch auto-matching for all CREATED orders.
func (h *OrderHandler) RunAutoMatch(c *gin.Context) {
	result, err := h.gw.Contract.SubmitTransaction("RunAutoMatch")
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "batch auto-match failed: " + err.Error()})
		return
	}

	var matchResp model.BatchMatchResponse
	if err := json.Unmarshal(result, &matchResp); err != nil {
		c.JSON(http.StatusOK, model.BatchMatchResponse{Matched: 0})
		return
	}
	c.JSON(http.StatusOK, matchResp)
}

// getTOUPrice fetches the current time-of-use electricity price from chaincode.
func (h *OrderHandler) getTOUPrice() float64 {
	result, err := h.gw.Contract.EvaluateTransaction("GetTimeOfUsePrice")
	if err != nil {
		return 1.0 // fallback default
	}
	var tou struct {
		Price  float64 `json:"price"`
		Period string  `json:"period"`
	}
	if json.Unmarshal(result, &tou) != nil {
		return 1.0
	}
	if tou.Price <= 0 {
		return 1.0
	}
	return tou.Price
}
