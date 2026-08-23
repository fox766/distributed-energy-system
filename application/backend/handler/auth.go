package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"backend/config"
	"backend/fabric"
	"backend/middleware"
	"backend/model"
	"backend/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuthHandler struct {
	cfg   *config.Config
	store *store.CredentialStore
	gw    *fabric.Gateway
}

func NewAuthHandler(cfg *config.Config, s *store.CredentialStore, gw *fabric.Gateway) *AuthHandler {
	return &AuthHandler{cfg: cfg, store: s, gw: gw}
}

// allowedRoles are the roles a client may assign itself at registration.
// "admin" is deliberately absent: the JWT role is minted from this stored value
// (see Login), so accepting it here would let anyone issue themselves an admin
// token and walk past every AdminRequired/ProducerRequired gate.
var allowedRoles = map[string]bool{
	"PRODUCER": true,
	"CONSUMER": true,
}

// allowedDeviceTypes mirrors the chaincode DeviceType constants
// (blockchain/chaincode-go/chaincode/module.go).
var allowedDeviceTypes = map[string]bool{
	"SOLAR_PANEL":     true,
	"WIND_TURBINE":    true,
	"BATTERY_STORAGE": true,
}

// Register creates a new user on chaincode and stores credentials locally.
func (h *AuthHandler) Register(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "invalid request: " + err.Error()})
		return
	}

	if !allowedRoles[req.Role] {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "role must be PRODUCER or CONSUMER"})
		return
	}
	if !allowedDeviceTypes[req.EnergyType] {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error: "energyType must be one of SOLAR_PANEL, WIND_TURBINE, BATTERY_STORAGE"})
		return
	}

	if h.store.Exists(req.Username) {
		c.JSON(http.StatusConflict, model.ErrorResponse{Error: "username already exists"})
		return
	}

	userID := "energy_user_" + uuid.New().String()[:12]

	// Register on chaincode. New users are credited an initial balance so that
	// consumers can place BUY orders, which escrow funds at creation time.
	balStr := strconv.FormatFloat(h.cfg.InitialBalance, 'f', 2, 64)
	_, err := h.gw.Submit("RegisterUser",
		userID, req.Username, req.Role, "0.00", balStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "chaincode register failed: " + fabric.ErrorDetail(err)})
		return
	}

	// Store credentials locally with userID
	if err := h.store.CreateUser(req.Username, req.Password, userID, req.Role, req.EnergyType); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "credential store failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"userid": userID, "message": "registration successful"})
}

// Login verifies credentials and returns a JWT token.
func (h *AuthHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "invalid request: " + err.Error()})
		return
	}

	if err := h.store.VerifyPassword(req.Username, req.Password); err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Error: "invalid username or password"})
		return
	}

	rec, ok := h.store.GetUserRecord(req.Username)
	if !ok {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "user record not found"})
		return
	}

	// Get user from chaincode to populate current balance/available
	user := model.User{
		UserID:   rec.UserID,
		UserName: req.Username,
		UserRole: rec.Role,
	}
	result, err := h.gw.Contract.EvaluateTransaction("GetUser", rec.UserID)
	if err == nil {
		var chainUser model.User
		if json.Unmarshal(result, &chainUser) == nil {
			user.Available = chainUser.Available
			user.Balance = chainUser.Balance
		}
	}

	token, err := middleware.GenerateToken(h.cfg, rec.UserID, rec.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "token generation failed"})
		return
	}

	c.JSON(http.StatusOK, model.LoginResponse{Token: token, User: user})
}

// Logout is a no-op. Client discards the token.
func (h *AuthHandler) Logout(c *gin.Context) {
	c.JSON(http.StatusOK, model.MessageResponse{Message: "logged out"})
}

// BootstrapAdmin creates the admin user if it doesn't exist.
func BootstrapAdmin(cfg *config.Config, s *store.CredentialStore, gw *fabric.Gateway) error {
	if s.Exists(cfg.AdminUsername) {
		return nil
	}

	adminID := "energy_user_admin"
	// Admin never generates energy, so the device type is inert; it only has to
	// satisfy the column's NOT NULL constraint.
	if err := s.CreateUser(cfg.AdminUsername, cfg.AdminPassword, adminID, "admin", store.DefaultDeviceType); err != nil {
		return err
	}

	availStr := strconv.FormatFloat(1000000.0, 'f', 2, 64)
	balStr := strconv.FormatFloat(1000000.0, 'f', 2, 64)
	_, err := gw.Submit("RegisterUser",
		adminID, cfg.AdminUsername, "admin", availStr, balStr)
	return err
}
