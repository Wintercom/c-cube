package model

type CreatePassageRequest struct {
	Passages    []string               `json:"passages" binding:"required"`
	Title       string                 `json:"title,omitempty"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type CreatePassageResponse struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	Message   string `json:"message"`
}
