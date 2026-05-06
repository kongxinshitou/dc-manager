package main

import (
	"dcmanager/config"
	"dcmanager/database"
	"dcmanager/handlers"
	"dcmanager/mcp"
	"dcmanager/middleware"
	"dcmanager/services"
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
	services.StartInspectionEscalationWorker(database.DB)

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"*"},
		AllowCredentials: false,
	}))

	// 静态文件服务 - 上传的图片
	r.Static("/uploads", config.UploadDir)

	api := r.Group("/api")
	{
		// 公开路由 - 登录
		api.POST("/auth/login", handlers.Login)

		// 需要认证的路由
		auth := api.Group("")
		auth.Use(middleware.AuthRequired())
		{
			// 认证相关
			auth.POST("/auth/change-password", handlers.ChangePassword)

			// 大屏数据
			auth.GET("/dashboard", handlers.GetDashboard)

			// 设备台账 - 读取
			auth.GET("/devices", handlers.GetDevices)
			auth.GET("/devices/options", handlers.GetDeviceOptions)
			auth.GET("/devices/export", handlers.ExportDevices)
			auth.GET("/devices/by-location", handlers.GetDeviceByLocation)
			auth.GET("/devices/cabinets", handlers.GetCabinets)
			auth.GET("/devices/:id", handlers.GetDevice)

			// 设备台账 - 写入
			auth.POST("/devices", middleware.PermissionRequired("device:write"), handlers.CreateDevice)
			auth.PUT("/devices/:id", middleware.PermissionRequired("device:write"), handlers.UpdateDevice)
			auth.DELETE("/devices/:id", middleware.PermissionRequired("device:delete"), handlers.DeleteDevice)
			auth.DELETE("/devices/batch", middleware.PermissionRequired("device:delete"), handlers.BatchDeleteDevices)
			auth.POST("/devices/import", middleware.PermissionRequired("device:import"), handlers.ImportDevices)

			// 设备工作流操作
			auth.POST("/devices/:id/operate", middleware.PermissionRequired("device:write"), handlers.OperateDevice)
			auth.GET("/devices/:id/operations", handlers.GetDeviceOperations)
			auth.PUT("/devices/batch-custodian", middleware.PermissionRequired("device:write"), handlers.BatchUpdateCustodian)

			// 系统配置
			auth.GET("/config/:key", middleware.PermissionRequired("config:manage"), handlers.GetConfig)
			auth.PUT("/config/:key", middleware.PermissionRequired("config:manage"), handlers.UpdateConfig)

			// 审批管理
			auth.POST("/approvals", middleware.PermissionRequired("approval:submit"), handlers.SubmitApproval)
			auth.GET("/approvals", middleware.PermissionRequired("approval:view"), handlers.GetApprovals)
			auth.GET("/approvals/:id", middleware.PermissionRequired("approval:view"), handlers.GetApproval)
			auth.PUT("/approvals/:id/approve", middleware.PermissionRequired("approval:approve"), handlers.ApproveApproval)
			auth.PUT("/approvals/:id/reject", middleware.PermissionRequired("approval:approve"), handlers.RejectApproval)
			auth.PUT("/approvals/:id/execute", middleware.PermissionRequired("approval:execute"), handlers.ExecuteApproval)
			auth.PUT("/approvals/:id/cancel", middleware.PermissionRequired("approval:submit"), handlers.CancelApproval)

			// 机房管理
			auth.GET("/datacenters", handlers.GetDatacenters)
			auth.POST("/datacenters", middleware.PermissionRequired("datacenter:manage"), handlers.CreateDatacenter)
			auth.PUT("/datacenters/:id", middleware.PermissionRequired("datacenter:manage"), handlers.UpdateDatacenter)
			auth.DELETE("/datacenters/:id", middleware.PermissionRequired("datacenter:manage"), handlers.DeleteDatacenter)
			auth.GET("/datacenters/:id/columns", handlers.GetDatacenterColumns)
			auth.POST("/datacenters/:id/columns", middleware.PermissionRequired("datacenter:manage"), handlers.SetDatacenterColumns)
			auth.GET("/datacenters/:id/rows", handlers.GetDatacenterRows)
			auth.POST("/datacenters/:id/rows", middleware.PermissionRequired("datacenter:manage"), handlers.SetDatacenterRows)
			auth.GET("/datacenters/:id/cabinets", handlers.GetDatacenterCabinets)
			auth.POST("/datacenters/:id/cabinets/generate", middleware.PermissionRequired("datacenter:manage"), handlers.GenerateCabinets)
			auth.GET("/datacenters/:id/layout", handlers.GetDatacenterLayout)
			auth.PUT("/cabinets/:id", middleware.PermissionRequired("datacenter:manage"), handlers.UpdateCabinet)
			auth.GET("/cabinets/:id/devices", handlers.GetCabinetDevices)

			// 巡检记录 - 读取
			auth.GET("/inspections", handlers.GetInspections)
			auth.GET("/inspections/:id", handlers.GetInspection)
			auth.GET("/inspections/:id/events", handlers.GetInspectionEvents)

			// 巡检记录 - 写入
			auth.POST("/inspections", middleware.PermissionRequired("inspection:write"), handlers.CreateInspection)
			auth.PUT("/inspections/:id", middleware.PermissionRequired("inspection:write"), handlers.UpdateInspection)
			auth.POST("/inspections/:id/transition", middleware.PermissionRequired("inspection:write"), handlers.TransitionInspection)
			auth.DELETE("/inspections/:id", middleware.PermissionRequired("inspection:delete"), handlers.DeleteInspection)
			auth.DELETE("/inspections/batch", middleware.PermissionRequired("inspection:delete"), handlers.BatchDeleteInspections)
			auth.POST("/inspections/import", middleware.PermissionRequired("inspection:import"), handlers.ImportInspections)

			// 巡检图片
			auth.GET("/inspections/:id/images", handlers.GetInspectionImages)
			auth.POST("/inspections/:id/images", middleware.PermissionRequired("image:upload"), handlers.UploadInspectionImages)
			auth.DELETE("/inspections/:id/images/:imageId", middleware.PermissionRequired("image:delete"), handlers.DeleteInspectionImage)

			// 角色管理
			auth.GET("/roles", middleware.PermissionRequired("role:manage"), handlers.GetRoles)
			auth.POST("/roles", middleware.PermissionRequired("role:manage"), handlers.CreateRole)
			auth.PUT("/roles/:id", middleware.PermissionRequired("role:manage"), handlers.UpdateRole)
			auth.DELETE("/roles/:id", middleware.PermissionRequired("role:manage"), handlers.DeleteRole)
			auth.GET("/permissions", middleware.PermissionRequired("role:manage"), handlers.GetPermissionInfo)

			// 用户管理
			auth.GET("/users/options", handlers.GetUserOptions)
			auth.GET("/users", middleware.PermissionRequired("user:manage"), handlers.GetUsers)
			auth.POST("/users", middleware.PermissionRequired("user:manage"), handlers.CreateUser)
			auth.PUT("/users/:id", middleware.PermissionRequired("user:manage"), handlers.UpdateUser)
			auth.PUT("/users/:id/reset-password", middleware.PermissionRequired("user:manage"), handlers.ResetPassword)
			auth.DELETE("/users/:id", middleware.PermissionRequired("user:manage"), handlers.DeleteUser)
		}
	}

	// MCP endpoints
	r.POST("/mcp", mcp.HandleMCP)
	r.GET("/mcp/sse", mcp.HandleMCPSSE)
	r.POST("/mcp/messages", mcp.HandleMCPMessages)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server running on :%s", port)
	log.Printf("MCP endpoint: http://localhost:%s/mcp", port)
	r.Run(":" + port)
}
