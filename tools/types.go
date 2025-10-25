package main

type TransformedQA struct {
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Passage     string                 `json:"passage"`
	Metadata    map[string]interface{} `json:"metadata"`
}
