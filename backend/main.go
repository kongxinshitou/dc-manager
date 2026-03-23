package main

import (
	"dcmanager/database"
	"dcmanager/handlers"
	"dcmanager/mcp"
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "dc_manager.db"
	}
	database.Init(dbPath)

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"*"},
		AllowCredentials: false,
	}))

	api := r.Group("/api")
	{
		// 设备台账
		api.GET("/devices", handlers.GetDevices)
		api.GET("/devices/options", handlers.GetDeviceOptions)
		api.GET("/devices/:id", handlers.GetDevice)
		api.POST("/devices", handlers.CreateDevice)
		api.PUT("/devices/:id", handlers.UpdateDevice)
		api.DELETE("/devices/:id", handlers.DeleteDevice)

		// 巡检记录
		api.GET("/inspections", handlers.GetInspections)
		api.GET("/inspections/:id", handlers.GetInspection)
		api.POST("/inspections", handlers.CreateInspection)
		api.PUT("/inspections/:id", handlers.UpdateInspection)
		api.DELETE("/inspections/:id", handlers.DeleteInspection)

		// 大屏数据
		api.GET("/dashboard", handlers.GetDashboard)
	}

	// MCP endpoint
	r.POST("/mcp", mcp.HandleMCP)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server running on :%s", port)
	log.Printf("MCP endpoint: http://localhost:%s/mcp", port)
	r.Run(":" + port)
}
