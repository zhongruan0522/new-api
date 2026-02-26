package common

import (
	"strings"

	"github.com/QuantumNous/new-api/dto"
)

type MediaURLResolver func(rawURL string, mediaContentType string) (resolvedURL string, err error)

type mediaURL struct {
	Kind string
	URL  string
}

// ApplyImageAutoConvertToURL converts multimodal media blocks (e.g. "image_url", "video_url")
// into plain-text URLs and appends them to the end of the corresponding user message.
//
// This is intended for text-only upstream models: the model can "see" image URLs and call
// an external image understanding tool (e.g. MCP), while the upstream request stays text-only.
//
// Note: despite the legacy name, this function now handles both images and videos.
func ApplyImageAutoConvertToURL(req *dto.GeneralOpenAIRequest, resolve MediaURLResolver) (changed bool, err error) {
	if req == nil || len(req.Messages) == 0 {
		return false, nil
	}
	if resolve == nil {
		resolve = func(rawURL string, _ string) (string, error) { return rawURL, nil }
	}

	extractText := func(contents []dto.MediaContent) string {
		if len(contents) == 0 {
			return ""
		}
		var b strings.Builder
		for _, part := range contents {
			if part.Type != dto.ContentTypeText {
				continue
			}
			if part.Text == "" {
				continue
			}
			b.WriteString(part.Text)
		}
		return b.String()
	}

	for i := range req.Messages {
		if strings.ToLower(req.Messages[i].Role) != "user" {
			continue
		}

		contents := req.Messages[i].ParseContent()
		if len(contents) == 0 {
			continue
		}

		mediaURLs := make([]mediaURL, 0)
		for _, part := range contents {
			switch part.Type {
			case dto.ContentTypeImageURL:
				image := part.GetImageMedia()
				if image == nil {
					continue
				}
				url := strings.TrimSpace(image.Url)
				if url == "" {
					continue
				}
				resolved, rErr := resolve(url, dto.ContentTypeImageURL)
				if rErr != nil {
					return changed, rErr
				}
				resolved = strings.TrimSpace(resolved)
				if resolved == "" {
					continue
				}
				mediaURLs = append(mediaURLs, mediaURL{Kind: "image", URL: resolved})
			case dto.ContentTypeVideoUrl:
				video := part.GetVideoUrl()
				if video == nil {
					continue
				}
				url := strings.TrimSpace(video.Url)
				if url == "" {
					continue
				}
				resolved, rErr := resolve(url, dto.ContentTypeVideoUrl)
				if rErr != nil {
					return changed, rErr
				}
				resolved = strings.TrimSpace(resolved)
				if resolved == "" {
					continue
				}
				mediaURLs = append(mediaURLs, mediaURL{Kind: "video", URL: resolved})
			}
		}

		if len(mediaURLs) == 0 {
			continue
		}

		// Deduplicate while preserving order (per message).
		dedup := make([]mediaURL, 0, len(mediaURLs))
		seen := make(map[string]struct{}, len(mediaURLs))
		for _, item := range mediaURLs {
			kind := strings.TrimSpace(item.Kind)
			u := strings.TrimSpace(item.URL)
			if u == "" {
				continue
			}
			key := kind + "\n" + u
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			dedup = append(dedup, mediaURL{Kind: kind, URL: u})
		}
		if len(dedup) == 0 {
			continue
		}

		// Strip non-text parts from this user message to keep the upstream request text-only,
		// and append URLs to the same message.
		base := strings.TrimRight(extractText(contents), " \t\r\n")
		if strings.TrimSpace(base) == "" {
			base = "[media]"
		}

		var b strings.Builder
		b.WriteString(strings.TrimRight(base, " \t\r\n"))
		b.WriteString("\n\n")
		for idx, item := range dedup {
			if idx > 0 {
				b.WriteString("\n")
			}
			switch strings.TrimSpace(item.Kind) {
			case "image":
				b.WriteString("图片URL：")
				b.WriteString(item.URL)
				b.WriteString("，请使用MCP工具查看")
			case "video":
				b.WriteString("视频URL：")
				b.WriteString(item.URL)
				b.WriteString("，请使用MCP工具查看")
			default:
				// Fallback for unexpected kinds.
				b.WriteString("媒体URL：")
				b.WriteString(item.URL)
				b.WriteString("，请使用MCP工具查看")
			}
		}

		req.Messages[i].SetStringContent(strings.TrimRight(b.String(), "\n"))
		changed = true
	}

	return changed, nil
}
