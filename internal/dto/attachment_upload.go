package dto

// AttachmentUploadResult preserves every selected file's outcome, including partial success.
// An empty Items slice means the file picker was cancelled.
type AttachmentUploadResult struct {
	Items []AttachmentUploadItem `json:"items"`
}

// Exactly one of Attachment and Error is present for each selected file.
type AttachmentUploadItem struct {
	Filename   string                 `json:"filename"`
	Attachment *Attachment            `json:"attachment,omitempty"`
	Error      *AttachmentUploadError `json:"error,omitempty"`
}

// AttachmentUploadError contains only public diagnostics, never local paths or raw errors.
type AttachmentUploadError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Status    int    `json:"status"`
	RequestID string `json:"requestId,omitempty"`
}
