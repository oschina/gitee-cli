package tui

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestValidateImageURLRejectsUnsafeTargets(t *testing.T) {
	for _, rawURL := range []string{
		"file:///etc/passwd",
		"http://127.0.0.1/image.png",
		"http://[::1]/image.png",
		"http://169.254.169.254/latest/meta-data",
		"https://user:pass@example.com/image.png",
	} {
		if _, err := validateImageURL(rawURL); err == nil {
			t.Errorf("expected %q to be rejected", rawURL)
		}
	}
	if _, err := validateImageURL("https://cdn.example.com/image.png"); err != nil {
		t.Fatalf("expected public HTTPS URL to pass syntax validation: %v", err)
	}
}

func TestBlockedImageIP(t *testing.T) {
	for _, rawIP := range []string{"127.0.0.1", "10.0.0.1", "169.254.169.254", "::1", "fd00::1"} {
		if !isBlockedImageIP(net.ParseIP(rawIP)) {
			t.Errorf("expected %s to be blocked", rawIP)
		}
	}
	if isBlockedImageIP(net.ParseIP("8.8.8.8")) {
		t.Fatal("expected public IP to be allowed")
	}
}

func TestFetchImageBytesValidatesContentAndSize(t *testing.T) {
	png := append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, 16)...)
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode:    http.StatusOK,
			Body:          io.NopCloser(bytes.NewReader(png)),
			ContentLength: int64(len(png)),
			Header:        make(http.Header),
		}, nil
	})}
	got, err := fetchImageBytesWithClient(client, "https://cdn.example.com/image.png")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, png) {
		t.Fatal("downloaded image bytes changed")
	}

	client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("not an image")),
			Header:     make(http.Header),
		}, nil
	})
	if _, err := fetchImageBytesWithClient(client, "https://cdn.example.com/image.png"); err == nil {
		t.Fatal("expected non-image response to be rejected")
	}

	client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode:    http.StatusOK,
			Body:          io.NopCloser(strings.NewReader("ignored")),
			ContentLength: maxImageBytes + 1,
			Header:        make(http.Header),
		}, nil
	})
	if _, err := fetchImageBytesWithClient(client, "https://cdn.example.com/image.png"); err == nil {
		t.Fatal("expected oversized image to be rejected")
	}
}

func TestRenderOSC8StripsTerminalControls(t *testing.T) {
	got := RenderOSC8("safe\x1b]8;;https://evil.example\a", "https://example.com/a.png\x1b")
	if strings.ContainsAny(got, "\x1b\a") {
		t.Fatalf("unexpected OSC 8 output: %q", got)
	}
}
