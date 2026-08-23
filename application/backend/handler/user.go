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

type UserHandler struct {
	gw *fabric.Gateway
}

func NewUserHandler(gw *fabric.Gateway) *UserHandler {
	return &UserHandler{gw: gw}
}

// GetMe returns the currently authenticated user's profile from the ledger.
func (h *UserHandler) GetMe(c *gin.Context) {
	userID := middleware.GetUserID(c)

	result, err := h.gw.Contract.EvaluateTransaction("GetUser", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "failed to get user: " + fabric.ErrorDetail(err)})
		return
	}

	var user model.User
	if err := json.Unmarshal(result, &user); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "failed to parse user data"})
		return
	}
	c.JSON(http.StatusOK, user)
}

// keepValue is the chaincode's "leave this field alone" sentinel: UpdateUser
// only assigns available/balance when the argument is >= 0.
const keepValue = "-1.00"

// UpdateMe updates the authenticated user's ledger record.
//
// Both fields are admin-only. Balance must come from settled trades, and
// available energy must come from GenerateEnergy — letting a user set either
// one directly is a mint: energy set by hand can be sold for real balance, so
// gating balance alone would close only half the hole.
func (h *UserHandler) UpdateMe(c *gin.Context) {
	userID := middleware.GetUserID(c)
	role := middleware.GetUserRole(c)

	var req model.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "invalid request: " + err.Error()})
		return
	}

	if role != "admin" {
		if req.Balance != nil {
			c.JSON(http.StatusForbidden, model.ErrorResponse{Error: "balance can only be changed through trades"})
			return
		}
		if req.Available != nil {
			c.JSON(http.StatusForbidden, model.ErrorResponse{Error: "available energy can only be changed by generating energy"})
			return
		}
	}
	if req.Available != nil && *req.Available < 0 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "available must not be negative"})
		return
	}
	if req.Balance != nil && *req.Balance < 0 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "balance must not be negative"})
		return
	}

	// Pass the sentinel for omitted fields instead of reading the record first:
	// a read-modify-write would race the generation scheduler on the same key.
	availStr, balStr := keepValue, keepValue
	if req.Available != nil {
		availStr = strconv.FormatFloat(*req.Available, 'f', 2, 64)
	}
	if req.Balance != nil {
		balStr = strconv.FormatFloat(*req.Balance, 'f', 2, 64)
	}

	if _, err := h.gw.Submit("UpdateUser", userID, availStr, balStr); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "failed to update user: " + fabric.ErrorDetail(err)})
		return
	}

	c.JSON(http.StatusOK, model.MessageResponse{Message: "user updated"})
}

// TopUp credits a user's balance. Mounted under the admin-only route group;
// this is the only sanctioned way for a consumer to obtain funds, since BUY
// orders escrow the full cost at creation time.
func (h *UserHandler) TopUp(c *gin.Context) {
	var req model.TopUpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "invalid request: " + err.Error()})
		return
	}
	if req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "amount must be positive"})
		return
	}

	amountStr := strconv.FormatFloat(req.Amount, 'f', 2, 64)
	if _, err := h.gw.Submit("TopUpBalance", req.UserID, amountStr); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "failed to top up: " + fabric.ErrorDetail(err)})
		return
	}

	result, err := h.gw.Contract.EvaluateTransaction("GetUser", req.UserID)
	if err != nil {
		c.JSON(http.StatusOK, model.MessageResponse{Message: "balance topped up"})
		return
	}
	var user model.User
	if err := json.Unmarshal(result, &user); err != nil {
		c.JSON(http.StatusOK, model.MessageResponse{Message: "balance topped up"})
		return
	}
	c.JSON(http.StatusOK, user)
}
