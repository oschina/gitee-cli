package tui

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

const maxImageBytes = 10 << 20

type RendererType int

const (
	RendererNone RendererType = iota
	RendererChafa
	RendererKitty
	RendererITerm2
)

type ImageRef struct {
	Alt string
	URL string
}

var mdImageRe = regexp.MustCompile(`!\[([^\]]*)\]\(([^)\s]+)(?:\s+"[^"]*")?\)`)

// ExtractImages returns all ![alt](url) references found in raw markdown.
func ExtractImages(markdown string) []ImageRef {
	matches := mdImageRe.FindAllStringSubmatch(markdown, -1)
	refs := make([]ImageRef, 0, len(matches))
	for _, m := range matches {
		refs = append(refs, ImageRef{Alt: m[1], URL: m[2]})
	}
	return refs
}

// DetectRenderer probes the environment for image rendering capability.
// Priority: Kitty Graphics Protocol > iTerm2 Inline Images > chafa > none (OSC 8 hyperlink fallback).
func DetectRenderer() RendererType {
	if isKittySupported() {
		return RendererKitty
	}
	if isITerm2Supported() {
		return RendererITerm2
	}
	if _, err := exec.LookPath("chafa"); err == nil {
		return RendererChafa
	}
	return RendererNone
}

// isInsideTmux reports whether the process is running inside a tmux session.
func isInsideTmux() bool {
	return os.Getenv("TMUX") != ""
}

// detectOuterTerminal returns a string identifying the outer terminal even when
// running inside tmux, where TERM_PROGRAM is overwritten to "tmux".
// It checks environment variables that tmux passes through unchanged.
func detectOuterTerminal() string {
	// These vars survive tmux session inheritance
	if os.Getenv("ITERM_SESSION_ID") != "" {
		return "iterm2"
	}
	if os.Getenv("KITTY_PID") != "" || os.Getenv("KITTY_WINDOW_ID") != "" {
		return "kitty"
	}
	if os.Getenv("GHOSTTY_RESOURCES_DIR") != "" {
		return "ghostty"
	}
	if os.Getenv("WEZTERM_EXECUTABLE") != "" || os.Getenv("WEZTERM_PANE") != "" {
		return "wezterm"
	}
	// Fallback: TERM_PROGRAM is still reliable when not in tmux
	switch os.Getenv("TERM_PROGRAM") {
	case "iTerm.app":
		return "iterm2"
	case "WezTerm":
		return "wezterm"
	case "ghostty":
		return "ghostty"
	}
	return ""
}

func isKittySupported() bool {
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return true
	}
	if strings.Contains(os.Getenv("TERM"), "kitty") {
		return true
	}
	tp := os.Getenv("TERM_PROGRAM")
	if tp == "WezTerm" || tp == "ghostty" {
		return true
	}
	// Inside tmux: check the outer terminal
	if isInsideTmux() {
		outer := detectOuterTerminal()
		return outer == "kitty" || outer == "ghostty"
	}
	return false
}

func isITerm2Supported() bool {
	if os.Getenv("TERM_PROGRAM") == "iTerm.app" {
		return true
	}
	// Inside tmux: TERM_PROGRAM is overwritten; use ITERM_SESSION_ID instead
	if isInsideTmux() && os.Getenv("ITERM_SESSION_ID") != "" {
		return true
	}
	return false
}

func fetchImageBytes(rawURL string) ([]byte, error) {
	return fetchImageBytesWithClient(newImageHTTPClient(), rawURL)
}

func fetchImageBytesWithClient(client *http.Client, rawURL string) ([]byte, error) {
	u, err := validateImageURL(rawURL)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create image request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download image: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("download image: HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength > maxImageBytes {
		return nil, fmt.Errorf("image exceeds %d byte limit", maxImageBytes)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxImageBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read image body: %w", err)
	}
	if len(data) > maxImageBytes {
		return nil, fmt.Errorf("image exceeds %d byte limit", maxImageBytes)
	}
	if contentType := http.DetectContentType(data); !strings.HasPrefix(contentType, "image/") {
		return nil, fmt.Errorf("unsupported image content type %q", contentType)
	}
	return data, nil
}

func newImageHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = safeImageDialContext
	return &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("too many image redirects")
			}
			_, err := validateImageURL(req.URL.String())
			return err
		},
	}
}

func validateImageURL(rawURL string) (*url.URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid image URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("image URL must use http or https")
	}
	if u.Hostname() == "" || u.User != nil {
		return nil, fmt.Errorf("invalid image URL")
	}
	if ip := net.ParseIP(strings.Trim(u.Hostname(), "[]")); ip != nil && isBlockedImageIP(ip) {
		return nil, fmt.Errorf("image URL resolves to a non-public address")
	}
	return u, nil
}

func safeImageDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("invalid image address: %w", err)
	}
	addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("resolve image host: %w", err)
	}
	if len(addresses) == 0 {
		return nil, fmt.Errorf("image host has no addresses")
	}
	for _, address := range addresses {
		if isBlockedImageIP(address.IP) {
			return nil, fmt.Errorf("image host resolves to a non-public address")
		}
	}

	dialer := &net.Dialer{Timeout: 10 * time.Second}
	var lastErr error
	for _, address := range addresses {
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(address.IP.String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("connect to image host: %w", lastErr)
}

func isBlockedImageIP(ip net.IP) bool {
	return ip == nil || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}

// wrapTmuxPassthrough wraps an escape sequence in a DCS tmux passthrough so
// that tmux relays it verbatim to the outer terminal instead of discarding it.
// Rules: every ESC (0x1b) inside the payload must be doubled; the wrapper uses
// ST (ESC \) as the outer terminator because tmux does not accept BEL here.
// Requires `set -g allow-passthrough on` in ~/.tmux.conf (tmux ≥ 3.3).
func wrapTmuxPassthrough(seq string) string {
	escaped := strings.ReplaceAll(seq, "\x1b", "\x1b\x1b")
	return "\x1bPtmux;" + escaped + "\x1b\\"
}

// RenderKitty encodes the image at url as a Kitty Graphics Protocol APC
// sequence (chunked base64, f=100 lets kitty auto-detect format from magic bytes).
// Spec: https://sw.kovidgoyal.net/kitty/graphics-protocol/
func RenderKitty(url string, _ int) (string, error) {
	data, err := fetchImageBytes(url)
	if err != nil {
		return "", err
	}

	encoded := base64.StdEncoding.EncodeToString(data)

	// Kitty requires payloads split into ≤4096-byte base64 chunks.
	const chunkSize = 4096
	var sb strings.Builder
	for i := 0; i < len(encoded); i += chunkSize {
		end := i + chunkSize
		if end > len(encoded) {
			end = len(encoded)
		}
		chunk := encoded[i:end]
		more := 1
		if end >= len(encoded) {
			more = 0
		}
		if i == 0 {
			fmt.Fprintf(&sb, "\x1b_Ga=T,f=100,m=%d;%s\x1b\\", more, chunk)
		} else {
			fmt.Fprintf(&sb, "\x1b_Gm=%d;%s\x1b\\", more, chunk)
		}
	}
	return sb.String(), nil
}

// RenderChafa runs chafa to convert the image at url into unicode art sized
// to termWidth columns.
func RenderChafa(url string, termWidth int) (string, error) {
	width := termWidth
	if width <= 0 || width > 200 {
		width = 80
	}
	sizeArg := fmt.Sprintf("%dx", width)

	data, err := fetchImageBytes(url)
	if err != nil {
		return "", err
	}

	f, err := os.CreateTemp("", "gitee-img-*.png")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(f.Name())

	if _, err := f.Write(data); err != nil {
		f.Close()
		return "", fmt.Errorf("write temp file: %w", err)
	}
	f.Close()

	out, err := exec.Command("chafa", "--size="+sizeArg, "--animate=off", "--symbols=half", "--colors=full", "-f", "symbols", f.Name()).Output()
	if err != nil {
		out, err = exec.Command("chafa", "--size="+sizeArg, "--symbols=half", "--colors=full", "-f", "symbols", f.Name()).Output()
		if err != nil {
			return "", fmt.Errorf("chafa: %w", err)
		}
	}
	return strings.ReplaceAll(string(out), "\n", "\r\n"), nil
}

// RenderITerm2 encodes the image as an iTerm2 Inline Images Protocol sequence.
// Spec: https://iterm2.com/documentation-images.html
func RenderITerm2(url string, termWidth int) (string, error) {
	data, err := fetchImageBytes(url)
	if err != nil {
		return "", err
	}

	width := termWidth
	if width <= 0 {
		width = 80
	}

	encoded := base64.StdEncoding.EncodeToString(data)

	if !isInsideTmux() {
		seq := fmt.Sprintf("\x1b]1337;File=inline=1;width=%dpx;preserveAspectRatio=1:%s\a\r\n", width*8, encoded)
		return seq, nil
	}

	const partSize = 200
	var sb strings.Builder

	header := fmt.Sprintf("\x1b]1337;MultipartFile=inline=1;width=%dpx;size=%d;preserveAspectRatio=1\a", width*8, len(data))
	sb.WriteString(wrapTmuxPassthrough(header))

	for i := 0; i < len(encoded); i += partSize {
		end := i + partSize
		if end > len(encoded) {
			end = len(encoded)
		}
		sb.WriteString(wrapTmuxPassthrough(fmt.Sprintf("\x1b]1337;FilePart=%s\a", encoded[i:end])))
	}

	sb.WriteString(wrapTmuxPassthrough("\x1b]1337;FileEnd\a"))
	sb.WriteString("\r\n")
	return sb.String(), nil
}

// RenderOSC8 returns an OSC 8 hyperlink so the user can click the image URL
// in terminals that support it (kitty, WezTerm, iTerm2, etc.).
func RenderOSC8(alt, rawURL string) string {
	label := sanitizeTerminalText(alt)
	if label == "" {
		label = sanitizeTerminalText(rawURL)
	}
	u, err := validateImageURL(rawURL)
	if err != nil {
		return label
	}
	return fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\", sanitizeTerminalText(u.String()), label)
}

func sanitizeTerminalText(value string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
			return -1
		}
		return r
	}, value)
}

// RenderImage tries each renderer in priority order and returns the rendered
// content along with the renderer that succeeded.
func RenderImage(ref ImageRef, renderer RendererType, termWidth int) (string, RendererType) {
	switch renderer {
	case RendererKitty:
		if s, err := RenderKitty(ref.URL, termWidth); err == nil {
			return s, RendererKitty
		}
		fallthrough
	case RendererITerm2:
		if s, err := RenderITerm2(ref.URL, termWidth); err == nil {
			return s, RendererITerm2
		}
		fallthrough
	case RendererChafa:
		if s, err := RenderChafa(ref.URL, termWidth); err == nil {
			return s, RendererChafa
		}
		fallthrough
	default:
		return RenderOSC8(ref.Alt, ref.URL), RendererNone
	}
}
