package handler

import (
	"net/http"
	"time"

	"github.com/Wintercom/c-cube/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type KnowledgeHandler struct {
}

func NewKnowledgeHandler() *KnowledgeHandler {
	return &KnowledgeHandler{}
}

func (h *KnowledgeHandler) CreateKnowledgeFromPassage(c *gin.Context) {
	kbID := c.Param("id")
	if kbID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "knowledge base ID is required",
		})
		return
	}

	var req model.CreatePassageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
			"details": err.Error(),
		})
		return
	}

	if len(req.Passages) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "passages array cannot be empty",
		})
		return
	}

	knowledgeID := uuid.New().String()
	createdAt := time.Now()

	response := model.CreatePassageResponse{
		ID:        knowledgeID,
		CreatedAt: createdAt.Format(time.RFC3339),
		Message:   "Knowledge passage created successfully",
	}

	c.JSON(http.StatusCreated, response)
}
