package gemini

var ModelList = []string{
	"embedding-001", "gemini-1.5-flash", "gemini-1.5-flash-8b",
	"gemini-1.5-flash-latest", "gemini-1.5-pro", "gemini-1.5-pro-latest",
	"gemini-2.0-flash", "gemini-2.0-flash-exp", "gemini-2.0-flash-lite-preview",
	"gemini-2.0-flash-thinking-exp", "gemini-2.0-pro-exp", "gemini-2.5-flash",
	"gemini-2.5-flash-image", "gemini-2.5-flash-lite", "gemini-2.5-pro",
	"gemini-2.5-pro-exp-03-25", "gemini-2.5-pro-preview", "gemini-2.5-pro-preview-03-25",
	"gemini-2.5-pro-preview-05-06", "gemini-3-flash-preview", "gemini-3-pro-image",
	"gemini-3-pro-image-preview", "gemini-3-pro-preview", "gemini-3.1-flash-image",
	"gemini-3.1-flash-image-preview", "gemini-3.1-flash-lite", "gemini-3.1-flash-lite-image",
	"gemini-3.1-flash-lite-preview", "gemini-3.1-pro-preview", "gemini-3.1-pro-preview-customtools",
	"gemini-3.5-flash", "gemini-embedding-exp-03-07", "gemini-exp-1206",
	"gemma-2-27b-it", "gemma-3-12b-it", "gemma-3-27b-it",
	"gemma-3-4b-it", "gemma-3n-e4b-it", "gemma-4-26b-a4b-it",
	"gemma-4-31b-it", "imagen-3.0-generate-002", "lyria-3-clip-preview",
	"lyria-3-pro-preview", "text-embedding-004",
}

var SafetySettingList = []string{
	"HARM_CATEGORY_HARASSMENT",
	"HARM_CATEGORY_HATE_SPEECH",
	"HARM_CATEGORY_SEXUALLY_EXPLICIT",
	"HARM_CATEGORY_DANGEROUS_CONTENT",
	//"HARM_CATEGORY_CIVIC_INTEGRITY", This item is deprecated!
}

var ChannelName = "google gemini"
