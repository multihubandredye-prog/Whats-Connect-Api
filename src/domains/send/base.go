package send

type BaseRequest struct {
	Phone       string `json:"phone" form:"phone" validate:"required"`
	Duration    *int   `json:"duration,omitempty" form:"duration"`
	IsForwarded bool   `json:"is_forwarded,omitempty" form:"is_forwarded"`
}
