package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"backend/fabric"
	"backend/middleware"
	"backend/model"

	"github.com/gin-gonic/gin"
)

type MarketHandler struct {
	gw *fabric.Gateway
}

func NewMarketHandler(gw *fabric.Gateway) *MarketHandler {
	return &MarketHandler{gw: gw}
}

// GetStatus returns system status including energy price and user/order counts.
func (h *MarketHandler) GetStatus(c *gin.Context) {
	energyResult, err := h.gw.Contract.EvaluateTransaction("GetEnergyStatus")
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "failed to get energy status: " + fabric.ErrorDetail(err)})
		return
	}

	var energy model.EnergyStatus
	if err := json.Unmarshal(energyResult, &energy); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "failed to parse energy status"})
		return
	}

	// Get counts from chaincode
	userCount, orderCount := 0, 0
	userResult, err := h.gw.Contract.EvaluateTransaction("GetSystemCounts")
	if err == nil {
		var counts struct {
			UserCount  int `json:"userCount"`
			OrderCount int `json:"orderCount"`
		}
		if json.Unmarshal(userResult, &counts) == nil {
			userCount = counts.UserCount
			orderCount = counts.OrderCount
		}
	}

	c.JSON(http.StatusOK, model.SystemStatus{
		EnergyPrice: energy.EnergyPrice,
		Fee:         energy.Fee,
		UserNum:     userCount,
		OrderNum:    orderCount,
	})
}

// GetPriceHistory returns recent energy price records.
func (h *MarketHandler) GetPriceHistory(c *gin.Context) {
	result, err := h.gw.Contract.EvaluateTransaction("GetPriceHistory", "7")
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "failed to get price history: " + fabric.ErrorDetail(err)})
		return
	}
	if len(result) == 0 || string(result) == "null" {
		c.JSON(http.StatusOK, []model.PriceRecord{})
		return
	}

	var records []model.PriceRecord
	if err := json.Unmarshal(result, &records); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "failed to parse price history"})
		return
	}
	if records == nil {
		records = []model.PriceRecord{}
	}
	c.JSON(http.StatusOK, records)
}

// GetCarbonStats returns total carbon savings from the ledger.
func (h *MarketHandler) GetCarbonStats(c *gin.Context) {
	result, err := h.gw.Contract.EvaluateTransaction("GetCarbonStats")
	if err != nil || len(result) == 0 {
		c.JSON(http.StatusOK, gin.H{"totalCarbonSaved": 0, "totalGreenTrades": 0})
		return
	}
	var stats map[string]interface{}
	if json.Unmarshal(result, &stats) != nil {
		c.JSON(http.StatusOK, gin.H{"totalCarbonSaved": 0, "totalGreenTrades": 0})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// GetAuditLog returns recent audit entries from the ledger.
func (h *MarketHandler) GetAuditLog(c *gin.Context) {
	result, err := h.gw.Contract.EvaluateTransaction("GetAuditLog", "50")
	if err != nil || len(result) == 0 {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}
	var entries []interface{}
	if json.Unmarshal(result, &entries) != nil {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}
	c.JSON(http.StatusOK, entries)
}

// UpdateEnergyPrice updates the system energy price (admin only via middleware).
func (h *MarketHandler) UpdateEnergyPrice(c *gin.Context) {
	var req model.UpdateEnergyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "invalid request: " + err.Error()})
		return
	}

	priceStr := strconv.FormatFloat(req.Price, 'f', 2, 64)
	feeStr := strconv.FormatFloat(req.Fee, 'f', 2, 64)

	_, err := h.gw.Submit("UpdateEnergyPrice", priceStr, feeStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "failed to update energy price: " + fabric.ErrorDetail(err)})
		return
	}

	c.JSON(http.StatusOK, model.MessageResponse{Message: "energy price updated"})
}

// GetTOUPrice returns the current time-of-use electricity price.
func (h *MarketHandler) GetTOUPrice(c *gin.Context) {
	result, err := h.gw.Contract.EvaluateTransaction("GetTimeOfUsePrice")
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "failed to get TOU price: " + fabric.ErrorDetail(err)})
		return
	}

	var touPrice model.TOUPriceResponse
	if err := json.Unmarshal(result, &touPrice); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "failed to parse TOU price"})
		return
	}
	c.JSON(http.StatusOK, touPrice)
}

// GenerateEnergy triggers energy generation for a producer's device.
func (h *MarketHandler) GenerateEnergy(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req model.GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "invalid request: " + err.Error()})
		return
	}

	if !allowedDeviceTypes[req.DeviceType] {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: "deviceType must be one of SOLAR_PANEL, WIND_TURBINE, BATTERY_STORAGE"})
		return
	}

	_, err := h.gw.Submit("GenerateEnergy", userID, req.DeviceType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "failed to generate energy: " + fabric.ErrorDetail(err)})
		return
	}

	c.JSON(http.StatusOK, model.MessageResponse{Message: "energy generated successfully"})
}

// GetGenerationHistory returns generation records.
func (h *MarketHandler) GetGenerationHistory(c *gin.Context) {
	userID := c.DefaultQuery("userid", "")
	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)

	result, err := h.gw.Contract.EvaluateTransaction("GetGenerationHistory", userID, strconv.Itoa(limit))
	if err != nil {
		c.JSON(http.StatusOK, []model.GenerationRecord{})
		return
	}
	if len(result) == 0 || string(result) == "null" {
		c.JSON(http.StatusOK, []model.GenerationRecord{})
		return
	}

	var records []model.GenerationRecord
	if err := json.Unmarshal(result, &records); err != nil {
		c.JSON(http.StatusOK, []model.GenerationRecord{})
		return
	}
	if records == nil {
		records = []model.GenerationRecord{}
	}
	c.JSON(http.StatusOK, records)
}

// GetTransactionHistory returns finished orders for a user.
func (h *MarketHandler) GetTransactionHistory(c *gin.Context) {
	userID := c.DefaultQuery("userid", "")
	month := c.DefaultQuery("month", "")
	limitStr := c.DefaultQuery("limit", "100")
	limit, _ := strconv.Atoi(limitStr)

	result, err := h.gw.Contract.EvaluateTransaction("GetTransactionHistory", userID, month, strconv.Itoa(limit))
	if err != nil {
		c.JSON(http.StatusOK, []model.Order{})
		return
	}
	if len(result) == 0 || string(result) == "null" {
		c.JSON(http.StatusOK, []model.Order{})
		return
	}

	var orders []model.Order
	if err := json.Unmarshal(result, &orders); err != nil {
		c.JSON(http.StatusOK, []model.Order{})
		return
	}
	if orders == nil {
		orders = []model.Order{}
	}
	c.JSON(http.StatusOK, orders)
}

// GetMonthlySummary returns a monthly statement for a user.
func (h *MarketHandler) GetMonthlySummary(c *gin.Context) {
	userID := c.DefaultQuery("userid", "")
	month := c.DefaultQuery("month", "")

	result, err := h.gw.Contract.EvaluateTransaction("GetMonthlySummary", userID, month)
	if err != nil {
		c.JSON(http.StatusOK, model.MonthlySummary{UserID: userID, Month: month})
		return
	}

	var summary model.MonthlySummary
	if err := json.Unmarshal(result, &summary); err != nil {
		c.JSON(http.StatusOK, model.MonthlySummary{UserID: userID, Month: month})
		return
	}
	c.JSON(http.StatusOK, summary)
}

// GetUserStatement returns all monthly summaries for a user.
func (h *MarketHandler) GetUserStatement(c *gin.Context) {
	userID := c.DefaultQuery("userid", "")

	result, err := h.gw.Contract.EvaluateTransaction("GetUserStatement", userID)
	if err != nil || len(result) == 0 || string(result) == "null" {
		c.JSON(http.StatusOK, []model.MonthlySummary{})
		return
	}

	var summaries []model.MonthlySummary
	if err := json.Unmarshal(result, &summaries); err != nil {
		c.JSON(http.StatusOK, []model.MonthlySummary{})
		return
	}
	if summaries == nil {
		summaries = []model.MonthlySummary{}
	}
	c.JSON(http.StatusOK, summaries)
}
