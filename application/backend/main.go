package main

import (
	"fmt"
	"log"
	"time"

	"backend/config"
	"backend/fabric"
	"backend/handler"
	"backend/middleware"
	"backend/store"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	// Initialize credential store (SQLite)
	credStore, err := store.New("credentials.db")
	if err != nil {
		log.Fatalf("Failed to initialize credential store: %v", err)
	}
	defer credStore.Close()

	// Initialize Fabric gateway
	gw, err := fabric.NewGateway(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to Fabric: %v", err)
	}
	defer gw.Close()

	// Initialize chaincode energy status. This must run BEFORE BootstrapAdmin:
	// Init seeds EnergyStatus with a zero user counter, so registering the
	// admin first would have its increment overwritten and undercount users.
	if _, err := gw.Submit("Init"); err != nil {
		log.Printf("Warning: chaincode init (may already be initialized): %v", err)
	}

	// Initialize TOU schedule
	if _, err := gw.Submit("InitTOUSchedule"); err != nil {
		log.Printf("Warning: TOU schedule init (may already be initialized): %v", err)
	}

	// Bootstrap admin user
	if err := handler.BootstrapAdmin(cfg, credStore, gw); err != nil {
		log.Printf("Warning: admin bootstrap: %v", err)
	}

	// Handlers
	authH := handler.NewAuthHandler(cfg, credStore, gw)
	userH := handler.NewUserHandler(gw)
	orderH := handler.NewOrderHandler(gw)
	marketH := handler.NewMarketHandler(gw)

	// Router with CORS
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost", "http://127.0.0.1"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// --- Public routes ---
	api := r.Group("/api")
	{
		api.POST("/register", authH.Register)
		api.POST("/login", authH.Login)

		api.GET("/market/status", marketH.GetStatus)
		api.GET("/market/price-history", marketH.GetPriceHistory)
		api.GET("/market/carbon-stats", marketH.GetCarbonStats)
		api.GET("/market/audit-log", marketH.GetAuditLog)
		api.GET("/market/tou-price", marketH.GetTOUPrice)
		api.GET("/orders", orderH.ListOrders)
		api.GET("/orders/:id", orderH.GetOrder)

		// Transaction history (public for transparency)
		api.GET("/transactions", marketH.GetTransactionHistory)
		api.GET("/transactions/summary", marketH.GetMonthlySummary)
		api.GET("/transactions/statement", marketH.GetUserStatement)
	}

	// --- Authenticated routes ---
	auth := api.Group("")
	auth.Use(middleware.AuthRequired(cfg))
	{
		auth.POST("/logout", authH.Logout)

		auth.GET("/user/me", userH.GetMe)
		auth.PUT("/user/me", userH.UpdateMe)

		auth.POST("/orders", orderH.CreateOrder)
		auth.GET("/orders/mine", orderH.ListMyOrders)
		auth.POST("/orders/:id/match", orderH.MatchOrder)
		auth.POST("/orders/:id/auto-match", orderH.AutoMatchOrder)
		auth.POST("/orders/:id/settle", orderH.SettleOrder)
		auth.POST("/orders/:id/cancel", orderH.CancelOrder)

		// Energy generation (producer or admin role checked in middleware)
		auth.POST("/generate", middleware.ProducerRequired(), marketH.GenerateEnergy)
		auth.GET("/generation-history", marketH.GetGenerationHistory)
	}

	// --- Admin-only routes ---
	admin := api.Group("/admin")
	admin.Use(middleware.AuthRequired(cfg))
	admin.Use(middleware.AdminRequired())
	{
		admin.POST("/energy-price", marketH.UpdateEnergyPrice)
		admin.POST("/auto-match", orderH.RunAutoMatch)
		// Only sanctioned way to fund a consumer: BUY orders escrow their full
		// cost at creation, so a zero balance makes buying impossible.
		admin.POST("/user/top-up", userH.TopUp)
	}

	// --- Background Scheduler ---
	// Disabled by DISABLE_SCHEDULERS so that end-to-end balance assertions are
	// not perturbed by background generation and matching.
	if cfg.DisableSchedulers {
		log.Printf("Schedulers disabled (DISABLE_SCHEDULERS set)")
	} else {
		// P0: Auto-generate energy for producers every 30 seconds
		go runGenerationScheduler(gw, credStore)
		// P2: Auto-match orders every 15 seconds
		go runAutoMatchScheduler(gw)
	}

	fmt.Printf("Energy Trading API server starting on :%s\n", cfg.Port)
	if err := r.Run("0.0.0.0:" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// runGenerationScheduler periodically generates energy for all PRODUCER users.
func runGenerationScheduler(gw *fabric.Gateway, credStore *store.CredentialStore) {
	// Wait for server to start
	time.Sleep(5 * time.Second)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Generate for producers only, each with the device they registered.
		// Admin is excluded: it is an operator account, not a generation site,
		// and including it made admin accrue energy indefinitely.
		users := credStore.GetAllUsers()
		for username, rec := range users {
			if rec.Role != "PRODUCER" {
				continue
			}
			deviceType := rec.DeviceType
			if deviceType == "" {
				deviceType = store.DefaultDeviceType
			}
			_, err := gw.Submit("GenerateEnergy", rec.UserID, deviceType)
			if err != nil {
				log.Printf("Scheduler: generation failed for %s (%s): %v", username, rec.UserID, fabric.ErrorDetail(err))
			} else {
				log.Printf("Scheduler: generated energy for %s (%s) using %s", username, rec.UserID, deviceType)
			}
		}
	}
}

// runAutoMatchScheduler periodically runs auto-matching for all CREATED orders.
func runAutoMatchScheduler(gw *fabric.Gateway) {
	time.Sleep(10 * time.Second)

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// The chaincode settles one pair per transaction (Fabric has no
		// read-your-writes), so drain the book here rather than in one call.
		settled, err := handler.RunAutoMatchLoop(gw, 50)
		if err != nil {
			log.Printf("Scheduler: auto-match failed after %d pair(s): %v", settled, fabric.ErrorDetail(err))
		} else if settled > 0 {
			log.Printf("Scheduler: auto-match settled %d pair(s)", settled)
		}
	}
}
