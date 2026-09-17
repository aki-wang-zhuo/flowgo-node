/**
 * 示例插件节点：将消息 data 前加上 prefix（配置项），走 Success。
 * 构建：见同目录 build.ps1 / build.sh
 */
package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/flowgo/flowgo-node/sdk"
	"github.com/flowgo/flowgo/api/types"
)

const Type = "pluginEcho"

// Def 面板元数据。
var Def = types.ComponentDef{
	Type:           Type,
	Label:          "插件回显",
	Labels:         map[string]string{types.LocaleEnUS: "Plugin Echo"},
	Category:       "测试",
	CategoryLabel:  "转换",
	CategoryLabels: map[string]string{types.LocaleEnUS: "Transform"},
	Order:          90,
	Icon:           "E",
	RelationTypes:  []string{types.RelationSuccess},
	Source:         types.ComponentSourcePlugin,
	Description:    "示例插件：在消息内容前追加 prefix。",
	Descriptions: map[string]string{
		types.LocaleEnUS: "Sample plugin: prepend prefix to message data.",
	},
	Usage: `configuration.prefix 为要追加到 data 前面的字符串，默认 "[echo] "。`,
	ConfigFields: []types.ConfigField{
		{
			Name: "prefix", Type: "string", Default: "[echo] ", Widget: types.WidgetText,
			Description: "前缀",
			Descriptions: map[string]string{types.LocaleEnUS: "Prefix"},
		},
	},
	Actions: types.NodeActions{Edit: true, Delete: true, Run: true, RunOnly: true},
}

// EchoNode 回显节点。
type EchoNode struct {
	prefix string
}

func New() types.Node { return &EchoNode{} }

func (n *EchoNode) Type() string { return Type }

func (n *EchoNode) Init(config map[string]interface{}) error {
	n.prefix = "[echo] "
	if config != nil {
		if v, ok := config["prefix"].(string); ok {
			n.prefix = v
		}
	}
	return nil
}

func (n *EchoNode) OnMsg(_ context.Context, msg types.Msg) (types.Msg, string, error) {
	msg.Data = n.prefix + strings.TrimPrefix(msg.Data, n.prefix)
	return msg, types.RelationSuccess, nil
}

func (n *EchoNode) Destroy() {}

func main() {
	if err := sdk.Serve(sdk.Registration{Def: Def, Factory: New}); err != nil {
		panic(fmt.Sprintf("plugin serve: %v", err))
	}
}
