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
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "failed to get user: " + err.Error()})
		return
	}

	var user model.User
	if err := json.Unmarshal(result, &user); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "failed to parse user data"})
		return
	}
	c.JSON(http.StatusOK, user)
}

// UpdateMe updates the authenticated user's available energy only (balance locked).
// Users cannot modify their own balance — balance only changes via settled trades.
func (h *UserHandler) UpdateMe(c *gin.Context) {
	userID := middleware.GetUserID(c)
	role := middleware.GetUserRole(c)

	var req model.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "invalid request: " + err.Error()})
		return
	}

	// Non-admin users cannot modify their own balance
	if role != "admin" && req.Balance != 0 {
		c.JSON(http.StatusForbidden, model.ErrorResponse{Error: "balance can only be changed through trades"})
		return
	}

	// Fetch current user to preserve existing values
	current, err := h.gw.Contract.EvaluateTransaction("GetUser", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "failed to get user: " + err.Error()})
		return
	}
	var user model.User
	if err := json.Unmarshal(current, &user); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "failed to parse user data"})
		return
	}

	// Only update available energy (for producers adding generation); balance stays
	avail := user.Available
	if req.Available >= 0 {
		avail = req.Available
	}
	// Balance: use existing unless admin explicitly sets
	bal := user.Balance
	if role == "admin" && req.Balance >= 0 {
		bal = req.Balance
	}

	availStr := strconv.FormatFloat(avail, 'f', 2, 64)
	balStr := strconv.FormatFloat(bal, 'f', 2, 64)

	_, err = h.gw.Contract.SubmitTransaction("UpdateUser", userID, availStr, balStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "failed to update user: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, model.MessageResponse{Message: "user updated"})
}
