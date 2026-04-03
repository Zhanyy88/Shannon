package attachments

// MaxMultimodalBodyBytes is the maximum request body size for endpoints that
// may carry inline base64 attachments (images, PDFs). 30 MB accommodates
// several high-resolution images plus JSON overhead.
const MaxMultimodalBodyBytes = 30 * 1024 * 1024

// MaxAttachmentThumbnailBytes is the maximum thumbnail data URL size persisted
// in DB metadata. Oversized thumbnails are silently dropped.
const MaxAttachmentThumbnailBytes = 50 * 1024

// MaxDecodedAttachmentBytes is the maximum total decoded size of all
// attachments in a single request (20 MB).
const MaxDecodedAttachmentBytes = 20 * 1024 * 1024
