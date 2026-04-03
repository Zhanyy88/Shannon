package attachments

import "strings"

// IsSupportedMediaType checks if a MIME type is supported for attachments.
// Supported: all image/* and text/* subtypes, plus application/pdf,
// application/json, application/xml, application/x-yaml,
// application/javascript, application/typescript.
func IsSupportedMediaType(mediaType string) bool {
	if strings.HasPrefix(mediaType, "image/") || strings.HasPrefix(mediaType, "text/") {
		return true
	}
	switch mediaType {
	case "application/pdf", "application/json", "application/xml",
		"application/x-yaml", "application/javascript", "application/typescript":
		return true
	}
	return false
}
