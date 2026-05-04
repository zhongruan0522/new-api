package constant

var StreamingTimeout int
var MaxFileDownloadMB int
var MaxImageUploadMB int
var MaxVideoUploadMB int
var StreamScannerMaxBufferMB int
var ForceStreamOption bool
var CountToken bool
var GetMediaToken bool
var GetMediaTokenNotStream bool
var MaxRequestBodyMB int
var AzureDefaultAPIVersion string
var NotifyLimitCount int
var NotificationLimitDurationMinute int
var GenerateDefaultToken bool
var ErrorLogEnabled bool

// Stored media pool limits (in MB). When exceeded, the oldest stored assets are deleted automatically.
var StoredImagePoolMB int
var StoredVideoPoolMB int

// TrustedRedirectDomains is a list of trusted domains for redirect URL validation.
// Domains support subdomain matching (e.g., "example.com" matches "sub.example.com").
var TrustedRedirectDomains []string
