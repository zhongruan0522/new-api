package common

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/dto"
)

type MediaURLResolver func(rawURL string, mediaContentType string) (resolvedURL string, err error)

type mediaURL struct {
	Kind string
	URL  string
}

// ApplyImageAutoConvertToURL converts multimodal media blocks (e.g. "image_url", "video_url")
// into plain-text URLs and appends them to the last user message.
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

	mediaURLs := make([]mediaURL, 0)
	lastUserIdx := -1

	for i := range req.Messages {
		if strings.ToLower(req.Messages[i].Role) != "user" {
			continue
		}
		lastUserIdx = i

		contents := req.Messages[i].ParseContent()
		if len(contents) == 0 {
			continue
		}

		hasMedia := false
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
				hasMedia = true
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
				hasMedia = true
			}
		}

		if !hasMedia {
			continue
		}

		// Strip non-text parts from this user message to keep the upstream request text-only.
		text := strings.TrimSpace(extractText(contents))
		if text == "" {
			text = "[media]"
		}
		req.Messages[i].SetStringContent(text)
		changed = true
	}

	if lastUserIdx < 0 || len(mediaURLs) == 0 {
		return changed, nil
	}

	// Deduplicate while preserving order.
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
		return changed, nil
	}

	lastUser := &req.Messages[lastUserIdx]
	base := strings.TrimRight(extractText(lastUser.ParseContent()), " \t\r\n")
	// Make sure the last user message is text-only before appending URLs.
	lastUser.SetStringContent(base)

	var b strings.Builder
	if base != "" {
		b.WriteString(base)
		b.WriteString("\n\n")
	}
	b.WriteString("Media URLs:\n")
	for idx, item := range dedup {
		label := strings.TrimSpace(item.Kind)
		u := strings.TrimSpace(item.URL)
		if label != "" {
			b.WriteString(fmt.Sprintf("%d. [%s] %s\n", idx+1, label, u))
		} else {
			b.WriteString(fmt.Sprintf("%d. %s\n", idx+1, u))
		}
	}

	lastUser.SetStringContent(strings.TrimRight(b.String(), "\n"))
	return true, nil
}
