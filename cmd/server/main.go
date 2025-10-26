package main

import (
	"log"
	"net/http"
	"time"

	"github.com/Wintercom/c-cube/internal/handler"
	"github.com/Wintercom/c-cube/internal/middleware"
	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	router.Use(middleware.CORS())
	router.Use(middleware.Logger())

	v1 := router.Group("/api/v1")
	{
		knowledgeHandler := handler.NewKnowledgeHandler()
		
		kbGroup := v1.Group("/knowledge-bases/:id")
		{
			kbGroup.POST("/knowledge/passage", knowledgeHandler.CreateKnowledgeFromPassage)
		}
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	log.Println("Starting C-Cube API Server on :8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
