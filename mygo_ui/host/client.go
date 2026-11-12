package host

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
)

// ErrClosed is returned after Shutdown or when the host process exits.
var ErrClosed = errors.New("host closed")

// Image is a display-encoded RGBA8 bitmap from the host (top row first).
type Image struct {
	Width, Height int
	Data          []byte // tightly packed RGBA8, len == width*height*4
	ColorSpace    string
}

// Hello is the hello reply (subset used by this UI).
type Hello struct {
	Protocol    int             `json:"protocol"`
	HostVersion string          `json:"host_version"`
	BuildInfo   string          `json:"build_info"`
	Methods     []string        `json:"methods"`
	Backend     json.RawMessage `json:"backend"`
}

// OpenResult is the open reply (subset).
type OpenResult struct {
	Session string `json:"session"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	Kind    string `json:"kind"`
}

// Client owns one spektralab-host (or mock) child process.
type Client struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser

	mu      sync.Mutex
	nextID  uint32
	pending map[uint32]chan reply
	closed  atomic.Bool
	reader  sync.Once
}

type reply struct {
	ok      bool
	result  json.RawMessage
	payload []byte
	errCode string
	errMsg  string
	err     error
}

// Start launches hostBin with --resources resourcesDir. hostBin may be
// "spektralab-host" on PATH, an absolute path, or "node" with mock args via
// SPEKTRALAB_MOCK_HOST (see README).
func Start(ctx context.Context, hostBin, resourcesDir string, extraArgs ...string) (*Client, error) {
	args := append([]string{"--resources", resourcesDir}, extraArgs...)
	cmd := exec.CommandContext(ctx, hostBin, args...)
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		return nil, err
	}
	c := &Client{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		pending: make(map[uint32]chan reply),
	}
	c.reader.Do(func() { go c.readLoop() })
	return c, nil
}

// StartMock launches the TypeScript mock host (desktop/mock-host/main.ts)
// with Node, for UI work without a Vulkan device.
func StartMock(ctx context.Context, repoRoot string) (*Client, error) {
	main := filepath.Join(repoRoot, "desktop", "mock-host", "main.ts")
	cmd := exec.CommandContext(ctx, "node", main)
	cmd.Dir = filepath.Join(repoRoot, "desktop")
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		return nil, err
	}
	c := &Client{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		pending: make(map[uint32]chan reply),
	}
	c.reader.Do(func() { go c.readLoop() })
	return c, nil
}

func (c *Client) readLoop() {
	r := bufio.NewReaderSize(c.stdout, 1<<20)
	for {
		header, payload, err := readFrame(r)
		if err != nil {
			c.failAll(err)
			return
		}
		var env struct {
			ID     *uint32         `json:"id"`
			Event  string          `json:"event"`
			OK     *bool           `json:"ok"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(header, &env); err != nil {
			continue
		}
		if env.ID == nil {
			continue // unsolicited event; ignore for this MVP
		}
		c.mu.Lock()
		ch := c.pending[*env.ID]
		delete(c.pending, *env.ID)
		c.mu.Unlock()
		if ch == nil {
			continue
		}
		rep := reply{payload: payload, result: env.Result}
		if env.OK != nil && *env.OK {
			rep.ok = true
		} else if env.Error != nil {
			rep.errCode = env.Error.Code
			rep.errMsg = env.Error.Message
		} else {
			rep.errMsg = "host reply missing ok/error"
		}
		ch <- rep
	}
}

func (c *Client) failAll(err error) {
	c.closed.Store(true)
	c.mu.Lock()
	defer c.mu.Unlock()
	for id, ch := range c.pending {
		ch <- reply{err: err}
		delete(c.pending, id)
	}
}

func (c *Client) call(method string, params any) (json.RawMessage, []byte, error) {
	if c.closed.Load() {
		return nil, nil, ErrClosed
	}
	id := atomic.AddUint32(&c.nextID, 1)
	ch := make(chan reply, 1)
	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()

	req := map[string]any{"id": id, "method": method, "params": params}
	if params == nil {
		req["params"] = map[string]any{}
	}
	if err := writeFrame(c.stdin, req, nil); err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, nil, err
	}
	rep := <-ch
	if rep.err != nil {
		return nil, nil, rep.err
	}
	if !rep.ok {
		return nil, nil, fmt.Errorf("%s: %s (%s)", method, rep.errMsg, rep.errCode)
	}
	return rep.result, rep.payload, nil
}

// Hello sends hello.
func (c *Client) Hello() (Hello, error) {
	raw, _, err := c.call("hello", map[string]any{})
	var h Hello
	if err != nil {
		return h, err
	}
	err = json.Unmarshal(raw, &h)
	return h, err
}

// Open opens a frame at path with the given params delta JSON object.
func (c *Client) Open(path string, params map[string]any) (OpenResult, error) {
	raw, _, err := c.call("open", map[string]any{"path": path, "params": params})
	var o OpenResult
	if err != nil {
		return o, err
	}
	err = json.Unmarshal(raw, &o)
	return o, err
}

// SetParams applies a params delta.
func (c *Client) SetParams(session string, delta map[string]any) error {
	_, _, err := c.call("set_params", map[string]any{"session": session, "delta": delta})
	return err
}

// RenderRGBA8 runs a legacy render (or render_view fallback) to sRGB RGBA8.
func (c *Client) RenderRGBA8(session, tier string, longEdge int) (Image, error) {
	params := map[string]any{
		"session": session,
		"tier":    tier,
		"format":  "rgba8",
		"display": "srgb",
	}
	if longEdge > 0 {
		// Prefer render_view when the host has it; fall back to render.
		viewParams := map[string]any{
			"session":     session,
			"tier":        tier,
			"format":      "rgba8",
			"display":     "srgb",
			"long_edge":   longEdge,
			"generation":  1,
			"adjustments": map[string]any{},
			"geometry":    map[string]any{},
		}
		raw, payload, err := c.call("render_view", viewParams)
		if err == nil {
			return decodeImage(raw, payload)
		}
		// fall through to render
	}
	raw, payload, err := c.call("render", params)
	if err != nil {
		return Image{}, err
	}
	return decodeImage(raw, payload)
}

// ExportPNG writes the current session to path as PNG.
func (c *Client) ExportPNG(session, path string) error {
	_, _, err := c.call("export_image", map[string]any{
		"session":     session,
		"path":        path,
		"format":      "png",
		"color_space": "sRGB",
		"overwrite":   true,
	})
	return err
}

// CloseSession releases one session.
func (c *Client) CloseSession(session string) error {
	_, _, err := c.call("close", map[string]any{"session": session})
	return err
}

// Shutdown asks the host to exit and waits.
func (c *Client) Shutdown() error {
	if c.closed.Load() {
		return nil
	}
	_, _, _ = c.call("shutdown", map[string]any{})
	c.closed.Store(true)
	_ = c.stdin.Close()
	return c.cmd.Wait()
}

func decodeImage(raw json.RawMessage, payload []byte) (Image, error) {
	var wrap struct {
		Image struct {
			Width      int    `json:"width"`
			Height     int    `json:"height"`
			Format     string `json:"format"`
			ColorSpace string `json:"color_space"`
			RowBytes   int    `json:"row_bytes"`
		} `json:"image"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return Image{}, err
	}
	info := wrap.Image
	if info.Format != "rgba8" && info.Format != "" {
		return Image{}, fmt.Errorf("unsupported format %q", info.Format)
	}
	if info.Width <= 0 || info.Height <= 0 {
		return Image{}, fmt.Errorf("bad image size %dx%d", info.Width, info.Height)
	}
	tight := info.Width * 4
	stride := info.RowBytes
	if stride == 0 {
		stride = tight
	}
	if stride < tight || len(payload) < stride*info.Height {
		return Image{}, fmt.Errorf("payload short: have %d need %d", len(payload), stride*info.Height)
	}
	out := make([]byte, tight*info.Height)
	for y := 0; y < info.Height; y++ {
		copy(out[y*tight:(y+1)*tight], payload[y*stride:y*stride+tight])
	}
	return Image{Width: info.Width, Height: info.Height, Data: out, ColorSpace: info.ColorSpace}, nil
}
