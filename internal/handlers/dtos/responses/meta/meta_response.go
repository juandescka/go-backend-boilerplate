package dtos

type PaginationResponse struct {
	Page        int    `json:"page"`
	Limit       int    `json:"limit"`
	HasNext     bool   `json:"has_next"`
	HasPrevious bool   `json:"has_previous"`
	NextURL     string `json:"next_url"`
	PrevURL     string `json:"prev_url"`
	Total       *int64 `json:"total,omitempty"`
}

type FilterResponse struct {
	Role     string `json:"role,omitempty"`
	IsActive *bool  `json:"isActive,omitempty"`
}

type SortingResponse struct {
	OrderBy   string `json:"order_by,omitempty"`
	Direction string `json:"direction,omitempty"`
}
