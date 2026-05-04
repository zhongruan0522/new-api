package operation_setting

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseHTTPStatusCodeRanges_CommaSeparated(t *testing.T) {
	ranges, err := ParseHTTPStatusCodeRanges("401,403,500-599")
	require.NoError(t, err)
	require.Equal(t, []StatusCodeRange{
		{Start: 401, End: 401},
		{Start: 403, End: 403},
		{Start: 500, End: 599},
	}, ranges)
}

func TestParseHTTPStatusCodeRanges_MergeAndNormalize(t *testing.T) {
	ranges, err := ParseHTTPStatusCodeRanges("500-505,504,401,403,402")
	require.NoError(t, err)
	require.Equal(t, []StatusCodeRange{
		{Start: 401, End: 403},
		{Start: 500, End: 505},
	}, ranges)
}

func TestParseHTTPStatusCodeRanges_Invalid(t *testing.T) {
	_, err := ParseHTTPStatusCodeRanges("99,10000,foo,500-400,500-")
	require.Error(t, err)
}

func TestParseHTTPStatusCodeRanges_NoComma_IsInvalid(t *testing.T) {
	_, err := ParseHTTPStatusCodeRanges("401 403")
	require.Error(t, err)
}

func TestShouldDisableByStatusCode(t *testing.T) {
	orig := AutomaticDisableStatusCodeRanges
	t.Cleanup(func() { AutomaticDisableStatusCodeRanges = orig })

	AutomaticDisableStatusCodeRanges = []StatusCodeRange{
		{Start: 401, End: 403},
		{Start: 500, End: 599},
	}

	require.True(t, ShouldDisableByStatusCode(401))
	require.True(t, ShouldDisableByStatusCode(403))
	require.False(t, ShouldDisableByStatusCode(404))
	require.True(t, ShouldDisableByStatusCode(500))
	require.False(t, ShouldDisableByStatusCode(200))
}

func TestShouldRetryByStatusCode(t *testing.T) {
	orig := AutomaticRetryStatusCodeRanges
	t.Cleanup(func() { AutomaticRetryStatusCodeRanges = orig })

	AutomaticRetryStatusCodeRanges = []StatusCodeRange{
		{Start: 429, End: 429},
		{Start: 500, End: 599},
	}

	require.True(t, ShouldRetryByStatusCode(429))
	require.True(t, ShouldRetryByStatusCode(500))
	require.False(t, ShouldRetryByStatusCode(400))
	require.False(t, ShouldRetryByStatusCode(200))
}

func TestShouldRetryByStatusCode_DefaultMatchesLegacyBehavior(t *testing.T) {
	require.False(t, ShouldRetryByStatusCode(200))
	require.False(t, ShouldRetryByStatusCode(400))
	require.True(t, ShouldRetryByStatusCode(401))
	require.False(t, ShouldRetryByStatusCode(408))
	require.True(t, ShouldRetryByStatusCode(429))
	require.True(t, ShouldRetryByStatusCode(500))
	require.False(t, ShouldRetryByStatusCode(504))
	require.False(t, ShouldRetryByStatusCode(524))
	require.True(t, ShouldRetryByStatusCode(599))
}

func TestAutomaticRetryStatusCodesFromString_EmptyRestoresDefault(t *testing.T) {
	defaultRanges := append([]StatusCodeRange(nil), AutomaticRetryStatusCodeRanges...)
	orig := AutomaticRetryStatusCodeRanges
	t.Cleanup(func() { AutomaticRetryStatusCodeRanges = orig })

	AutomaticRetryStatusCodeRanges = []StatusCodeRange{{Start: 429, End: 429}}

	err := AutomaticRetryStatusCodesFromString("")
	require.NoError(t, err)
	require.Equal(t, defaultRanges, AutomaticRetryStatusCodeRanges)
	require.True(t, ShouldRetryByStatusCode(429))
	require.True(t, ShouldRetryByStatusCode(500))
	require.False(t, ShouldRetryByStatusCode(504))
}

func TestParseHTTPStatusCodeRanges_BusinessErrorCodes(t *testing.T) {
	// Zhipu AI business error codes (1000-1313)
	ranges, err := ParseHTTPStatusCodeRanges("1000-1313")
	require.NoError(t, err)
	require.Equal(t, []StatusCodeRange{
		{Start: 1000, End: 1313},
	}, ranges)
}

func TestShouldRetryByStatusCode_BusinessErrorCodes(t *testing.T) {
	orig := AutomaticRetryStatusCodeRanges
	t.Cleanup(func() { AutomaticRetryStatusCodeRanges = orig })

	AutomaticRetryStatusCodeRanges = []StatusCodeRange{
		{Start: 500, End: 599},
		{Start: 1300, End: 1313},
	}

	// Zhipu AI specific business error codes
	require.True(t, ShouldRetryByStatusCode(1302))   // concurrent limit
	require.True(t, ShouldRetryByStatusCode(1303))   // rate limit
	require.True(t, ShouldRetryByStatusCode(1312))   // model overloaded
	require.False(t, ShouldRetryByStatusCode(1000))  // auth failure - should not retry
	require.False(t, ShouldRetryByStatusCode(1210))  // param error - should not retry
	require.False(t, ShouldRetryByStatusCode(100))   // below min
	require.False(t, ShouldRetryByStatusCode(10000)) // above max
}

func TestShouldRetryByStatusCode_OutOfBounds(t *testing.T) {
	require.False(t, ShouldRetryByStatusCode(99))
	require.False(t, ShouldRetryByStatusCode(10000))
}
