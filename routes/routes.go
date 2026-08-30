package routes

import (
	"github.com/gin-gonic/gin"
	"FinalProjectBE/controllers"
	"FinalProjectBE/middleware"
	"FinalProjectBE/ws"
)

func SetupRouter(
	authCtrl *controllers.AuthController,
	orderCtrl *controllers.OrderController,
	hub *ws.Hub,
	jwtSecret string,
) *gin.Engine {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	r.GET("/ws/orders", ws.ServeWS(hub))

	api := r.Group("/api")
	{
		api.POST("/auth/login", authCtrl.Login)

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