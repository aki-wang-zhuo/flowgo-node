package sdk

import (
	"strings"

	"github.com/flowgo/flowgo/api/types"
)

// ProtocolVersion 当前 JSON-RPC 行协议版本。
const ProtocolVersion = 1

// 操作名（请求与响应共用 op 字段；响应用 *_ok / *_err）。
const (
	OpHello     = "hello"
	OpHelloOK   = "hello_ok"
	OpListDefs  = "list_defs"
	OpListDefsOK = "list_defs_ok"
	OpCreate    = "create"
	OpCreateOK  = "create_ok"
	OpInit      = "init"
	OpInitOK    = "init_ok"
	OpOnMsg     = "on_msg"
	OpOnMsgOK   = "on_msg_ok"
	OpOnMsgErr  = "on_msg_err"
	OpDestroy   = "destroy"
	OpDestroyOK = "destroy_ok"
	OpError     = "error"
)

// Envelope 一行 JSON 请求/响应。
type Envelope struct {
	ID         string                 `json:"id"`
	Op         string                 `json:"op"`
	Protocol   int                    `json:"protocol,omitempty"`
	Type       string                 `json:"type,omitempty"`
	InstanceID string                 `json:"instanceId,omitempty"`
	Config     map[string]interface{} `json:"config,omitempty"`
	Msg        *types.Msg             `json:"msg,omitempty"`
	Relation   string                 `json:"relation,omitempty"`
	Error      string                 `json:"error,omitempty"`
	Defs       []WireDef              `json:"defs,omitempty"`
	Types      []string               `json:"types,omitempty"`
}

// WireDef 跨进程传输的组件元数据（含多语言表，可 JSON 序列化）。
type WireDef struct {
	Type           string               `json:"type"`
	Label          string               `json:"label"`
	Labels         map[string]string    `json:"labels,omitempty"`
	Category       string               `json:"category"`
	CategoryLabel  string               `json:"categoryLabel"`
	CategoryLabels map[string]string    `json:"categoryLabels,omitempty"`
	Order          int                  `json:"order"`
	Color          string               `json:"color,omitempty"`
	Icon           string               `json:"icon,omitempty"`
	DefaultScript  string               `json:"defaultScript,omitempty"`
	RelationTypes  []string             `json:"relationTypes,omitempty"`
	Description    string               `json:"description,omitempty"`
	Descriptions   map[string]string    `json:"descriptions,omitempty"`
	Usage          string               `json:"usage,omitempty"`
	// Doc / Docs：二进制内嵌的编辑器 Markdown；磁盘 MD 优先于本字段。
	Doc            string               `json:"doc,omitempty"`
	Docs           map[string]string    `json:"docs,omitempty"`
	ConfigFields   []types.ConfigField  `json:"configFields,omitempty"`
	Actions        types.NodeActions    `json:"actions,omitempty"`
}

// ToWire 将引擎 ComponentDef 转为可序列化 WireDef。
func ToWire(d types.ComponentDef) WireDef {
	return WireDef{
		Type:           d.Type,
		Label:          d.Label,
		Labels:         d.Labels,
		Category:       d.Category,
		CategoryLabel:  d.CategoryLabel,
		CategoryLabels: d.CategoryLabels,
		Order:          d.Order,
		Color:          d.Color,
		Icon:           d.Icon,
		DefaultScript:  d.DefaultScript,
		RelationTypes:  d.RelationTypes,
		Description:    d.Description,
		Descriptions:   d.Descriptions,
		Usage:          d.Usage,
		Doc:            d.Doc,
		Docs:           d.Docs,
		ConfigFields:   d.ConfigFields,
		Actions:        d.Actions,
	}
}

// FromWire 还原为引擎 ComponentDef，并标记 Source=plugin；非标准分类归入 other。
func FromWire(w WireDef) types.ComponentDef {
	cat := strings.TrimSpace(w.Category)
	// 延迟到 host 侧 Normalize，避免 sdk 依赖 components；此处仅空串占位
	if cat == "" {
		cat = "other"
	}
	return types.ComponentDef{
		Type:           w.Type,
		Label:          w.Label,
		Labels:         w.Labels,
		Category:       cat,
		CategoryLabel:  w.CategoryLabel,
		CategoryLabels: w.CategoryLabels,
		Order:          w.Order,
		Color:          w.Color,
		Icon:           w.Icon,
		DefaultScript:  w.DefaultScript,
		RelationTypes:  w.RelationTypes,
		Description:    w.Description,
		Descriptions:   w.Descriptions,
		Usage:          w.Usage,
		Doc:            w.Doc,
		Docs:           w.Docs,
		Source:         types.ComponentSourcePlugin,
		ConfigFields:   w.ConfigFields,
		Actions:        w.Actions,
	}
}

// Registration 插件侧注册的一项节点。
type Registration struct {
	Def     types.ComponentDef
	Factory types.NodeFactory
}
