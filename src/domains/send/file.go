package send

import "mime/multipart"

type FileRequest struct {
	BaseRequest
	File      *multipart.FileHeader `json:"-" form:"file"`
	Caption   string                `json:"caption" form:"caption"`
	FileName  *string               `json:"file_name" form:"file_name"`
	FileURL   *string               `json:"file_url" form:"file_url"`
	FilePath  *string               `json:"file_path"`
}
