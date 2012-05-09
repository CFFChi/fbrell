package og

import (
	"net/url"
	"testing"
)

// Order insensitive pairs matching. This isn't fully accurate as OG
// is order sensitive. But since query parameters are not, we use this
// to ignore order.
func assertSubset(t *testing.T, expected, actual *Object) {
Outer:
	for _, pair := range expected.Pairs {
		for _, value := range actual.GetAll(pair.Key) {
			if pair.Value == value {
				continue Outer
			}
		}
		t.Fatalf(`Did not find expected pair "%s" = "%s"`, pair.Key, pair.Value)
	}
}

func TestParseBase64(t *testing.T) {
	t.Parallel()
	const song1 = "W1sib2c6dGl0bGUiLCJzb25nMSJdLFsib2c6dHlwZSIsInNvbmciXV0"
	expected := &Object{Pairs: []Pair{
		{"og:title", "song1"},
		{"og:type", "song"},
		{"og:url", "http://www.fbrell.com/rog/" + song1},
		{"og:image", "http://www.fbrell.com/" + stockImages[8]},
		{"og:description", stockDescriptions[0]},
	}}

	object, err := NewFromBase64(song1)
	if err != nil {
		t.Fatal(err)
	}
	assertSubset(t, expected, object)
}

func TestParseValues(t *testing.T) {
	t.Parallel()
	const ogType = "article"
	const ogTitle = "foo"
	values := url.Values{}
	values.Set("og:type", ogType)
	values.Set("og:title", ogTitle)
	expected := &Object{Pairs: []Pair{
		{"og:type", ogType},
		{"og:title", ogTitle},
		{"og:url", "http://www.fbrell.com/og/" + ogType + "/" + ogTitle},
		{"og:image", "http://www.fbrell.com/" + stockImages[8]},
		{"og:description", stockDescriptions[6]},
	}}

	object, err := NewFromValues(values)
	if err != nil {
		t.Fatal(err)
	}
	assertSubset(t, expected, object)
}
