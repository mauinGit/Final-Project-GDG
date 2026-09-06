package routes

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"FinalProjectBE/controllers"
	"FinalProjectBE/middleware"
	"FinalProjectBE/ws"
	"golang.org/x/time/rate"
)

func SetupRouter(
	authCtrl *controllers.AuthController,
	orderCtrl *controllers.OrderController,
	healthCtrl *controllers.HealthController,
	hub *ws.Hub,
	jwtSecret string,
	log *slog.Logger,
) *gin.Engine {
	r := gin.New()

	r.Use(middleware.RequestID())
	r.Use(middleware.Logger(log))
	r.Use(middleware.Recovery(log))

	globalLimiter := middleware.NewRateLimiter(rate.Limit(20), 40)
	r.Use(globalLimiter.Middleware())

	r.StaticFile("/", "./frontend/index.html")
	r.Static("/frontend", "./frontend")

	r.GET("/healthz", healthCtrl.Healthz)
	r.GET("/readyz", healthCtrl.Readyz)

	r.GET("/ws/orders", ws.ServeWS(hub))

	api := r.Group("/api")
	{
		loginLimiter := middleware.NewRateLimiter(rate.Limit(0.2), 5)
		api.POST("/auth/login", loginLimiter.Middleware(), authCtrl.Login)

		protected := api.Group("")
		protected.Use(middleware.AuthRequired(jwtSecret))
		{
			protected.GET("/orders", orderCtrl.List)
			protected.GET("/orders/:id", orderCtrl.GetByID)

			protected.POST("/orders", middleware.RequireRole("kasir"), orderCtrl.Create)
			protected.DELETE("/orders/:id", middleware.RequireRole("kasir"), orderCtrl.Cancel)

			protected.PATCH("/orders/:id/status", middleware.RequireRole("koki"), orderCtrl.UpdateStatus)
		}
	}

	return r
}