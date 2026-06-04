package model

import (
	"testing"
)

func strPtr(value string) *string {
	return &value
}

func TestSearchChannelsFiltersByExactGroup(t *testing.T) {
	setupChannelCacheTestDB(t)

	createChannelCacheTestChannel(t, Channel{Name: "alpha channel", Group: "alpha", Models: "gpt-test"})
	createChannelCacheTestChannel(t, Channel{Name: "beta channel", Group: "beta", Models: "gpt-test"})
	createChannelCacheTestChannel(t, Channel{Name: "multi channel", Group: "beta,alpha", Models: "gpt-test"})

	channels, err := SearchChannels("", "alpha", "gpt-test", false)
	if err != nil {
		t.Fatalf("SearchChannels error = %v", err)
	}

	if len(channels) != 2 {
		t.Fatalf("channel count = %d, want 2", len(channels))
	}
	for _, channel := range channels {
		if channel.Group != "alpha" && channel.Group != "beta,alpha" {
			t.Fatalf("unexpected channel group %q in filtered results", channel.Group)
		}
	}
}

func TestSearchChannelsEscapesGroupWildcards(t *testing.T) {
	setupChannelCacheTestDB(t)

	createChannelCacheTestChannel(t, Channel{Name: "literal underscore", Group: "alpha_1", Models: "gpt-test"})
	createChannelCacheTestChannel(t, Channel{Name: "wildcard lookalike", Group: "alphaX1", Models: "gpt-test"})

	channels, err := SearchChannels("", "alpha_1", "gpt-test", false)
	if err != nil {
		t.Fatalf("SearchChannels error = %v", err)
	}

	if len(channels) != 1 {
		t.Fatalf("channel count = %d, want 1", len(channels))
	}
	if channels[0].Group != "alpha_1" {
		t.Fatalf("channel group = %q, want alpha_1", channels[0].Group)
	}
}

func TestChannelTagQueriesApplyGroupFilter(t *testing.T) {
	setupChannelCacheTestDB(t)

	createChannelCacheTestChannel(t, Channel{Name: "alpha tagged", Group: "alpha", Models: "gpt-test", Tag: strPtr("shared")})
	createChannelCacheTestChannel(t, Channel{Name: "beta tagged", Group: "beta", Models: "gpt-test", Tag: strPtr("shared")})
	createChannelCacheTestChannel(t, Channel{Name: "beta only", Group: "beta", Models: "gpt-test", Tag: strPtr("beta-only")})

	query := ApplyChannelGroupFilter(DB.Model(&Channel{}), "alpha")
	total, err := CountChannelTags(query)
	if err != nil {
		t.Fatalf("CountChannelTags error = %v", err)
	}
	if total != 1 {
		t.Fatalf("tag total = %d, want 1", total)
	}

	tags, err := GetPaginatedChannelTags(ApplyChannelGroupFilter(DB.Model(&Channel{}), "alpha"), 0, 20)
	if err != nil {
		t.Fatalf("GetPaginatedChannelTags error = %v", err)
	}
	if len(tags) != 1 || tags[0] == nil || *tags[0] != "shared" {
		t.Fatalf("tags = %#v, want only shared", tags)
	}

	channels, err := GetChannelsByTagWithGroup("shared", "alpha", false, false)
	if err != nil {
		t.Fatalf("GetChannelsByTagWithGroup error = %v", err)
	}
	if len(channels) != 1 || channels[0].Group != "alpha" {
		t.Fatalf("channels = %#v, want only alpha shared channel", channels)
	}
	if channels[0].Key != "" {
		t.Fatal("expected non-selectAll tag query to omit channel key")
	}
}

func TestNormalizeChannelGroupFilter(t *testing.T) {
	for _, raw := range []string{"", "all", "ALL", "null", " NULL "} {
		if got := NormalizeChannelGroupFilter(raw); got != "" {
			t.Fatalf("NormalizeChannelGroupFilter(%q) = %q, want empty", raw, got)
		}
	}
	if got := NormalizeChannelGroupFilter(" default "); got != "default" {
		t.Fatalf("NormalizeChannelGroupFilter kept %q, want default", got)
	}
}
