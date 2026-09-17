package sdk

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"

	"github.com/flowgo/flowgo/api/types"
)

// Client 托管侧：与插件子进程通信。
type Client struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	reader *bufio.Reader
	enc    *json.Encoder
	mu     sync.Mutex
	nextID atomic.Uint64
	defs   []WireDef
	types  []string
}

// Start 启动插件可执行文件并完成握手。
func Start(binPath string) (*Client, error) {
	cmd := exec.Command(binPath)
	cmd.Env = append(os.Environ(), "FLOWGO_PLUGIN=1")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, err
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		return nil, err
	}
	c := &Client{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		reader: bufio.NewReader(stdout),
		enc:    json.NewEncoder(stdin),
	}
	c.enc.SetEscapeHTML(false)
	hello, err := c.roundTrip(Envelope{Op: OpHello, Protocol: ProtocolVersion})
	if err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("plugin hello: %w", err)
	}
	if hello.Op != OpHelloOK {
		_ = c.Close()
		return nil, fmt.Errorf("plugin hello failed: %s", hello.Error)
	}
	if hello.Protocol != 0 && hello.Protocol != ProtocolVersion {
		_ = c.Close()
		return nil, fmt.Errorf("unsupported plugin protocol %d", hello.Protocol)
	}
	c.types = append([]string(nil), hello.Types...)
	list, err := c.roundTrip(Envelope{Op: OpListDefs})
	if err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("plugin list_defs: %w", err)
	}
	if list.Op != OpListDefsOK {
		_ = c.Close()
		return nil, fmt.Errorf("plugin list_defs failed: %s", list.Error)
	}
	c.defs = append([]WireDef(nil), list.Defs...)
	return c, nil
}

// Defs 返回插件声明的组件元数据。
func (c *Client) Defs() []WireDef {
	return append([]WireDef(nil), c.defs...)
}

// Types 返回插件声明的节点类型列表。
func (c *Client) Types() []string {
	return append([]string(nil), c.types...)
}

// Create 在插件进程中创建节点实例。
func (c *Client) Create(typeName, instanceID string) error {
	resp, err := c.roundTrip(Envelope{Op: OpCreate, Type: typeName, InstanceID: instanceID})
	if err != nil {
		return err
	}
	if resp.Op != OpCreateOK {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}

// Init 初始化插件侧实例。
func (c *Client) Init(instanceID string, config map[string]interface{}) error {
	resp, err := c.roundTrip(Envelope{Op: OpInit, InstanceID: instanceID, Config: config})
	if err != nil {
		return err
	}
	if resp.Op != OpInitOK {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}

// OnMsg 转发消息到插件侧实例。
func (c *Client) OnMsg(ctx context.Context, instanceID string, msg types.Msg) (types.Msg, string, error) {
	_ = ctx
	msgCopy := msg
	resp, err := c.roundTrip(Envelope{Op: OpOnMsg, InstanceID: instanceID, Msg: &msgCopy})
	if err != nil {
		return types.Msg{}, "", err
	}
	if resp.Op == OpOnMsgErr || resp.Op == OpError {
		return types.Msg{}, "", fmt.Errorf("%s", resp.Error)
	}
	if resp.Op != OpOnMsgOK || resp.Msg == nil {
		return types.Msg{}, "", fmt.Errorf("unexpected on_msg response: %s", resp.Op)
	}
	return *resp.Msg, resp.Relation, nil
}

// Destroy 销毁插件侧实例。
func (c *Client) Destroy(instanceID string) error {
	resp, err := c.roundTrip(Envelope{Op: OpDestroy, InstanceID: instanceID})
	if err != nil {
		return err
	}
	if resp.Op != OpDestroyOK && resp.Op != OpError {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}

// Close 结束子进程。
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.stdin.Close()
	_ = c.stdout.Close()
	if c.cmd != nil && c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
		_, _ = c.cmd.Process.Wait()
	}
	return nil
}

func (c *Client) roundTrip(req Envelope) (Envelope, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if req.ID == "" {
		req.ID = fmt.Sprintf("%d", c.nextID.Add(1))
	}
	if err := c.enc.Encode(req); err != nil {
		return Envelope{}, err
	}
	line, err := c.reader.ReadBytes('\n')
	if err != nil {
		return Envelope{}, err
	}
	var resp Envelope
	if err := json.Unmarshal(line, &resp); err != nil {
		return Envelope{}, err
	}
	return resp, nil
}
