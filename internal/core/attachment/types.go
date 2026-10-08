package attachment

import (
	"net/http"

	"github.com/taoworklabs/mmerp/internal/platform"
)

// maxBytes bounds one file.
const maxBytes = 20 << 20

const (
	docx = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	xlsx = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
)

// accepted maps each file type taken to the name extensions a file picker offers for it.
var accepted = map[string][]string{
	"application/pdf": {".pdf"},
	"image/jpeg":      {".jpg", ".jpeg"},
	"image/png":       {".png"},
	docx:              {".docx"},
	xlsx:              {".xlsx"},
}

// actionDelete is the per-file allowed action.
const actionDelete = "delete"

// AttachmentList is the attachments of a record, with the limits a client checks before
// uploading; the server checks them again.
type AttachmentList struct {
	// The caller sees the record but its files need a right they lack (e.g. salary view).
	Hidden bool `json:"hidden"`
	// attach when the actor may add a file.
	AllowedActions []string     `json:"allowed_actions" nullable:"false"`
	Items          []Attachment `json:"items" nullable:"false"`
	MaxMB          int          `json:"max_mb"`
	Accept         []string     `json:"accept" nullable:"false" doc:"Name extensions of the accepted types, e.g. .pdf"`
}

type Attachment struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	Size           int64  `json:"size"`
	ContentType    string `json:"content_type"`
	UploadedByName string `json:"uploaded_by_name"`
	UploadedAt     string `json:"uploaded_at" format:"date-time"`
	// delete, for its uploader or whoever holds attach, while the record takes changes.
	AllowedActions []string `json:"allowed_actions" nullable:"false"`
}

var (
	ErrFileTooLarge = &platform.Error{Status: http.StatusRequestEntityTooLarge, Code: "file_too_large", Params: map[string]any{"max_mb": maxBytes >> 20}}
	ErrFileType     = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "file_type_not_allowed"}
	errFileName     = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_request", Params: map[string]any{"fields": []string{"file"}}}
)
