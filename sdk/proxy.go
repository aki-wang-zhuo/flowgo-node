package sdk

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/flowgo/flowgo/api/types"
)

var proxySeq atomic.Uint64

// ProxyNode 引擎侧代理：把 Init/OnMsg/Destroy 转发到插件进程。
type ProxyNode struct {
	client     *Client
	typeName   string
	instanceID string
	created    bool
}

// NewProxyFactory 生成可注册到 Registry 的工厂。
func NewProxyFactory(client *Client, typeName string) types.NodeFactory {
	return func() types.Node {
		return &ProxyNode{
			client:   client,
			typeName: typeName,
		}
	}
}

// Type 实现 types.Node。
func (p *ProxyNode) Type() string { return p.typeName }

// Init 在插件侧创建并初始化实例。
func (p *ProxyNode) Init(config map[string]interface{}) error {
	if p.client == nil {
		return fmt.Errorf("plugin client is nil")
	}
	if !p.created {
		p.instanceID = fmt.Sprintf("%s-%d", p.typeName, proxySeq.Add(1))
		if err := p.client.Create(p.typeName, p.instanceID); err != nil {
			return err
		}
		p.created = true
	}
	return p.client.Init(p.instanceID, config)
}

// OnMsg 转发到插件。
func (p *ProxyNode) OnMsg(ctx context.Context, msg types.Msg) (types.Msg, string, error) {
	if !p.created {
		return types.Msg{}, "", fmt.Errorf("plugin instance not initialized")
	}
	return p.client.OnMsg(ctx, p.instanceID, msg)
}

// Destroy 销毁插件侧实例。
func (p *ProxyNode) Destroy() {
	if p.created && p.client != nil {
		_ = p.client.Destroy(p.instanceID)
		p.created = false
	}
}
