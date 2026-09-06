package routes

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"FinalProjectBE/controllers"
	"FinalProjectBE/middleware"
	"FinalProjectBE/ws"
)

func SetupRouter(
	authCtrl *controllers.AuthController,
	orderCtrl *controllers.OrderController,
	healthCtrl *controllers.HealthController,
	menuCtrl *controllers.MenuController,
	reportCtrl *controllers.ReportController,
	hub *ws.Hub,
	jwtSecret string,
	log *slog.Logger,
	allowedOrigins []string,
) *gin.Engine {
	r := gin.New()

	r.Use(middleware.RequestID())
	r.Use(middleware.Logger(log))
	r.Use(middleware.Recovery(log))

	globalLimiter := middleware.NewRateLimiter(rate.Limit(20), 40)
	r.Use(globalLimiter.Middleware())

	r.Use(middleware.CORS(allowedOrigins))
	r.Use(middleware.SecurityHeaders())

	r.StaticFile("/", "./frontend/index.html")
	r.Static("/frontend", "./frontend")

	r.GET("/healthz", healthCtrl.Healthz)
	r.GET("/readyz", healthCtrl.Readyz)

	r.GET("/ws/orders", ws.ServeWS(hub))

	api := r.Group("/api")
	{
		loginLimiter := middleware.NewRateLimiter(rate.Limit(0.2), 5)
		api.POST("/auth/login", loginLimiter.Middleware(), authCtrl.Login)
		api.POST("/auth/refresh", loginLimiter.Middleware(), authCtrl.Refresh)

		protected := api.Group("")
		protected.Use(middleware.AuthRequired(jwtSecret))
		{
			protected.GET("/auth/me", authCtrl.Me)
			protected.POST("/auth/logout", authCtrl.Logout)
			protected.GET("/orders", orderCtrl.List)
			protected.GET("/orders/:id", orderCtrl.GetByID)

			protected.POST("/orders", middleware.RequireRole("kasir"), orderCtrl.Create)
			protected.DELETE("/orders/:id", middleware.RequireRole("kasir"), orderCtrl.Cancel)

			protected.PATCH("/orders/:id/status", middleware.RequireRole("koki"), orderCtrl.UpdateStatus)

			protected.GET("/menu", menuCtrl.List)
			protected.GET("/menu/:id", menuCtrl.GetByID)

			protected.POST("/menu", middleware.RequireRole("kasir"), menuCtrl.Create)
			protected.PATCH("/menu/:id", middleware.RequireRole("kasir"), menuCtrl.Update)
			protected.DELETE("/menu/:id", middleware.RequireRole("kasir"), menuCtrl.Delete)

			protected.GET("/reports/daily", middleware.RequireRole("kasir"), reportCtrl.Daily)
		}
	}

	return r
}