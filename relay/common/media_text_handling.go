package common

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/dto"
)

type MediaTextResolver func(kind string, resolvedURL string) (text string, err error)

type mediaTextCounters struct {
	image int
	video int
}

func ApplyMediaAutoConvertToText(req *dto.GeneralOpenAIRequest, resolveURL MediaURLResolver, resolveText MediaTextResolver) (changed bool, err error) {
	if req == nil || len(req.Messages) == 0 {
		return false, nil
	}
	if resolveText == nil {
		return false, errors.New("resolveText is nil")
	}

	urlResolver := resolveURL
	if urlResolver == nil {
		urlResolver = func(rawURL string, _ string) (string, error) { return rawURL, nil }
	}

	counters := mediaTextCounters{}
	for i := range req.Messages {
		if strings.ToLower(req.Messages[i].Role) != "user" {
			continue
		}

		contents := req.Messages[i].ParseContent()
		if len(contents) == 0 {
			continue
		}

		base := strings.TrimRight(extractTextFromContents(contents), " \t\r\n")
		mediaURLs, collectErr := collectResolvedMediaURLs(contents, urlResolver)
		if collectErr != nil {
			return changed, collectErr
		}
		mediaURLs = dedupMediaURLs(mediaURLs)
		if len(mediaURLs) == 0 {
			continue
		}

		if strings.TrimSpace(base) == "" {
			base = "[media]"
		}
		appendix, nextCounters, buildErr := buildMediaTextAppendix(mediaURLs, resolveText, counters)
		if buildErr != nil {
			return changed, buildErr
		}
		counters = nextCounters
		if strings.TrimSpace(appendix) == "" {
			continue
		}

		req.Messages[i].SetStringContent(strings.TrimRight(base+"\n\n"+appendix, "\n"))
		changed = true
	}

	return changed, nil
}

func extractTextFromContents(contents []dto.MediaContent) string {
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

func collectResolvedMediaURLs(contents []dto.MediaContent, resolveURL MediaURLResolver) ([]mediaURL, error) {
	if len(contents) == 0 {
		return nil, nil
	}
	out := make([]mediaURL, 0)
	for _, part := range contents {
		switch part.Type {
		case dto.ContentTypeImageURL:
			image := part.GetImageMedia()
			if image == nil {
				continue
			}
			resolved, err := resolveURL(strings.TrimSpace(image.Url), dto.ContentTypeImageURL)
			if err != nil {
				return out, err
			}
			resolved = strings.TrimSpace(resolved)
			if resolved == "" {
				continue
			}
			out = append(out, mediaURL{Kind: "image", URL: resolved})
		case dto.ContentTypeVideoUrl:
			video := part.GetVideoUrl()
			if video == nil {
				continue
			}
			resolved, err := resolveURL(strings.TrimSpace(video.Url), dto.ContentTypeVideoUrl)
			if err != nil {
				return out, err
			}
			resolved = strings.TrimSpace(resolved)
			if resolved == "" {
				continue
			}
			out = append(out, mediaURL{Kind: "video", URL: resolved})
		}
	}
	return out, nil
}

func buildMediaTextAppendix(items []mediaURL, resolveText MediaTextResolver, counters mediaTextCounters) (string, mediaTextCounters, error) {
	if len(items) == 0 {
		return "", counters, nil
	}

	var b strings.Builder
	for _, item := range items {
		kind := strings.TrimSpace(item.Kind)
		u := strings.TrimSpace(item.URL)
		if u == "" {
			continue
		}

		text, err := resolveText(kind, u)
		if err != nil {
			return "", counters, err
		}
		text = strings.TrimSpace(text)
		if text == "" {
			return "", counters, fmt.Errorf("empty third-party model output for %s: %s", kind, u)
		}

		if b.Len() > 0 {
			b.WriteString("\n")
		}
		switch kind {
		case "image":
			counters.image++
			b.WriteString("图片")
			b.WriteString(strconv.Itoa(counters.image))
			b.WriteString("：")
			b.WriteString(text)
		case "video":
			counters.video++
			b.WriteString("视频")
			b.WriteString(strconv.Itoa(counters.video))
			b.WriteString("：")
			b.WriteString(text)
		default:
			b.WriteString("媒体：")
			b.WriteString(text)
		}
	}

	return strings.TrimRight(b.String(), "\n"), counters, nil
}

func dedupMediaURLs(items []mediaURL) []mediaURL {
	if len(items) == 0 {
		return nil
	}

	out := make([]mediaURL, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
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
		out = append(out, mediaURL{Kind: kind, URL: u})
	}
	return out
}
