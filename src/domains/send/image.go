package send

import "mime/multipart"

type ImageRequest struct {
	BaseRequest
	Caption   string                `json:"caption" form:"caption"`
	Image     *multipart.FileHeader `json:"-" form:"image"`
	ImageURL  *string               `json:"image_url" form:"image_url"`
	ImagePath *string               `json:"image_path"`
	ViewOnce  bool                  `json:"view_once" form:"view_once"`
	Compress  bool                  `json:"compress"`
}
