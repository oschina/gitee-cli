package tui

import (
	"fmt"
	"io"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

type imageViewerCmd struct {
	images   []ImageRef
	startIdx int
	renderer RendererType
	width    int
	height   int
}

func newImageViewerCmd(images []ImageRef, startIdx int, width, height int) tea.ExecCommand {
	return &imageViewerCmd{
		images:   images,
		startIdx: startIdx,
		renderer: DetectRenderer(),
		width:    width,
		height:   height,
	}
}

func (c *imageViewerCmd) SetStdin(_ io.Reader)  {}
func (c *imageViewerCmd) SetStdout(_ io.Writer) {}
func (c *imageViewerCmd) SetStderr(_ io.Writer) {}

func (c *imageViewerCmd) Run() error {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("enter raw mode: %w", err)
	}
	defer term.Restore(fd, oldState) //nolint:errcheck

	idx := c.startIdx
	for {
		c.render(idx)

		b := make([]byte, 4)
		n, err := os.Stdin.Read(b)
		if err != nil || n == 0 {
			break
		}
		key := string(b[:n])

		switch {
		case key == "q" || key == "\x1b" || key == "\x03":
			c.clearScreen()
			return nil
		case key == "n" || key == "l" || key == "\x1b[C":
			if idx < len(c.images)-1 {
				idx++
			}
		case key == "p" || key == "h" || key == "\x1b[D":
			if idx > 0 {
				idx--
			}
		}
	}
	c.clearScreen()
	return nil
}

func (c *imageViewerCmd) clearScreen() {
	if c.renderer == RendererKitty {
		fmt.Fprint(os.Stdout, "\x1b_Ga=d\x1b\\")
	}
	fmt.Fprint(os.Stdout, "\x1b[2J\x1b[H")
	if c.width > 0 && c.height > 0 {
		blank := strings.Repeat(" ", c.width)
		for row := 0; row < c.height; row++ {
			fmt.Fprintf(os.Stdout, "\x1b[%d;1H%s", row+1, blank)
		}
	}
	fmt.Fprint(os.Stdout, "\x1b[H")
}

func (c *imageViewerCmd) render(idx int) {
	ref := c.images[idx]

	fmt.Fprint(os.Stdout, "\x1b[2J\x1b[H")

	content, usedRenderer := RenderImage(ref, c.renderer, c.width)

	fmt.Fprint(os.Stdout, content)

	if usedRenderer == RendererKitty || (usedRenderer == RendererITerm2 && isInsideTmux()) {
		fmt.Fprintf(os.Stdout, "\x1b[%d;1H", c.height-2)
	} else {
		fmt.Fprintln(os.Stdout)
	}

	c.printFooter(idx, ref, usedRenderer)
}

func (c *imageViewerCmd) printFooter(idx int, ref ImageRef, usedRenderer RendererType) {
	alt := ref.Alt
	if alt == "" {
		alt = ref.URL
	}
	if len(alt) > 50 {
		alt = alt[:47] + "..."
	}

	counter := fmt.Sprintf("[%d/%d]", idx+1, len(c.images))
	nav := "n/→ next · p/← prev · q/esc back"
	if len(c.images) == 1 {
		nav = "q/esc back"
	}

	var rendererLabel string
	switch usedRenderer {
	case RendererKitty:
		rendererLabel = "kitty"
	case RendererITerm2:
		rendererLabel = "iterm2"
	case RendererChafa:
		rendererLabel = "chafa"
	default:
		rendererLabel = "link"
	}

	line := strings.Join([]string{counter, alt, rendererLabel, nav}, "  ·  ")
	if c.width > 0 && len(line) > c.width {
		line = line[:c.width]
	}

	fmt.Fprintf(os.Stdout, "\r\n\x1b[7m %s \x1b[0m\r\n", line)
}
