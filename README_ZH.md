# FlowGo Node

[English](./README.md)

**FlowGo Node** 提供 FlowGo 的**进程外插件 SDK** 与示例节点。插件是独立可执行文件，通过 stdin/stdout 上的 **JSON 行协议**与宿主通信。[flowgo-server](https://github.com/aki-wang-zhuo/flowgo-server) 负责托管进程，将代理节点注册到 [flowgo](https://github.com/aki-wang-zhuo/flowgo) 引擎 Registry，并支持从编辑器或 REST API 热加载 / 启用 / 卸载。

> **内置节点应放在 `flowgo`，不要放在本仓库。** 此处仅用于第三方 / 可导入扩展。

| 相关项目 | 定位 |
| --- | --- |
| [flowgo](https://github.com/aki-wang-zhuo/flowgo) | 引擎 + 内置节点 |
| [flowgo-server](https://github.com/aki-wang-zhuo/flowgo-server) | 插件宿主（`pluginhost`） |
| [flowgo-editor](https://github.com/aki-wang-zhuo/flowgo-editor) | 安装 / 启用 / 文档 UI |

---

## 特性

- **进程隔离** — 每个插件为操作系统可执行文件（**不是** Go `plugin` 的 `.so`/`.dll`）
- **共用 SDK** — 插件侧（`Serve`）与宿主侧（`Client` + `ProxyNode`）同一模块
- **组件元数据** — 中英标签、描述、配置字段、文档（与内置节点同一套 `ComponentDef`）
- **生命周期 RPC** — hello → list_defs → create → init → on_msg → destroy
- **安装格式** — 裸二进制或 zip（二进制 + `*_zh.md` / `*_en.md` 文档）
- **示例** — `flowgo-nodes/echo` 中的 `pluginEcho`

---

## 环境要求

- Go **1.22+**
- 开发时本地存在 [flowgo](https://github.com/aki-wang-zhuo/flowgo) 以满足 `replace`
- 插件二进制的 **OS/架构须与服务端主机一致**（Windows PE / Linux ELF）

建议 monorepo 布局：

```text
RuleGo/
├── flowgo/
├── flowgo-server/
├── flowgo-node/     ← 本仓库
└── flowgo-editor/
```

---

## 目录结构

```text
flowgo-node/
├── sdk/                         # github.com/flowgo/flowgo-node/sdk
│   ├── protocol.go              # Envelope、WireDef、操作、版本
│   ├── serve.go                 # 插件主循环（stdin/stdout）
│   ├── client.go                # 宿主：启动子进程 + RPC
│   ├── proxy.go                 # ProxyNode / NewProxyFactory
│   └── client_test.go
└── flowgo-nodes/
    └── echo/                    # 示例插件（独立 go.mod）
        ├── main.go
        ├── build.ps1
        ├── pluginEcho_zh.md
        ├── pluginEcho_en.md
        └── dist/                # 构建产物（gitignore）
```

**没有**根级 `go.mod`。SDK 与每个插件各自为独立 module。

---

## 快速开始 — 编写插件

### 1. 实现 `types.Node`

```go
type EchoNode struct {
    prefix string
}

func (n *EchoNode) Type() string { return "pluginEcho" }

func (n *EchoNode) Init(config map[string]interface{}) error {
    // 读取 config["prefix"] 等
    return nil
}

func (n *EchoNode) OnMsg(ctx context.Context, msg types.Msg) (types.Msg, string, error) {
    msg.Data = n.prefix + msg.Data
    return msg, types.RelationSuccess, nil
}

func (n *EchoNode) Destroy() {}
```

### 2. 声明双语 `ComponentDef`

提供中文默认值，并在 `Labels` / `Descriptions` / `Docs`（及字段描述）中补齐英文。打包时将 `pluginEcho_zh.md` / `pluginEcho_en.md` 与二进制放在一起。

### 3. 在 `main` 中调用 `sdk.Serve`

```go
func main() {
    sdk.Serve(sdk.Registration{
        Def:     Def,
        Factory: func() types.Node { return &EchoNode{} },
    })
}
```

一个进程可通过多个 `Registration` 注册多种类型。

### 4. 构建与打包

```powershell
cd flowgo-nodes/echo
.\build.ps1
# → dist/echo.exe（或 echo）、文档与 echo.zip
```

### 5. 安装到服务端

- **编辑器**：设置 → 节点/插件管理 → 上传 `.exe` 或 `.zip`
- **API**：`POST /api/components/plugins/load`（multipart `file`，需鉴权）

插件落盘于 `flowgo-server/data/plugins/<id>/`，并生成 `manifest.json`。

---

## 协议概览

| 请求 `op` | 成功响应 | 作用 |
| --- | --- | --- |
| `hello` | `hello_ok` | 握手；返回协议版本与类型列表 |
| `list_defs` | `list_defs_ok` | 返回 `WireDef[]` 元数据 |
| `create` | `create_ok` | 按 type + instanceId 创建实例 |
| `init` | `init_ok` | 应用配置 |
| `on_msg` | `on_msg_ok` / `on_msg_err` | 处理消息；返回 msg + relation |
| `destroy` | `destroy_ok` | 销毁实例 |

- 协议版本：**1**（`ProtocolVersion`）
- 帧格式：stdin/stdout 上**每行一个 JSON 对象**
- 宿主设置环境变量 `FLOWGO_PLUGIN=1`
- **禁止向 stdout 写日志**（stdout 是 RPC 通道）。需要时可写 stderr。

---

## SDK API

### 插件侧

| API | 说明 |
| --- | --- |
| `sdk.Serve(regs ...Registration)` | 主循环 |
| `sdk.Registration{Def, Factory}` | 元数据 + `types.NodeFactory` |
| `sdk.ToWire` / `FromWire` | `ComponentDef` ↔ 可 JSON 的 `WireDef` |

### 宿主侧（flowgo-server 使用）

| API | 说明 |
| --- | --- |
| `sdk.Start(binPath)` | 启动子进程、握手、拉取 defs |
| `(*Client).Create / Init / OnMsg / Destroy / Close` | 实例生命周期 |
| `sdk.NewProxyFactory(client, typeName)` | 供 `engine.Registry.Register` 使用的工厂 |
| `sdk.ProxyNode` | 实现 `types.Node`，转发 RPC |

模块路径：

```text
github.com/flowgo/flowgo-node/sdk
```

---

## 示例：`pluginEcho`

| 项 | 值 |
| --- | --- |
| Type | `pluginEcho` |
| 行为 | 在 `msg.Data` 前追加可配置的 `prefix` |
| 默认前缀 | `"[echo] "` |
| 出边 | `Success` |
| 文档 | `pluginEcho_zh.md` / `pluginEcho_en.md` |

集成测试：`sdk/client_test.go`（`TestEchoPluginRoundTrip`）。

---

## 服务端生命周期（参考）

1. 启动时 `pluginhost` 扫描 `data/plugins/` 并 `LoadAll()`。
2. 对每个插件：启动二进制 → `hello` + `list_defs` → 注册代理工厂。
3. 引擎执行时调用 `ProxyNode.OnMsg`，RPC 进入插件进程。
4. 启用 / 停用 / 卸载：

| 方法 | 路径 |
| --- | --- |
| `POST` | `/api/components/plugins/load` |
| `PUT` | `/api/components/plugins/{id}/enabled` |
| `DELETE` | `/api/components/plugins/{id}` |

约束：

- 类型名不可与**内置**节点冲突（可覆盖已有 **plugin** 源类型）。
- `disabled: true` 时保留磁盘文件但不启动进程。

---

## 测试

```bash
cd sdk
go test ./...
```

Echo 相关测试在二进制缺失时可能会按需编译示例插件。

---

## 许可证

本仓库尚未发布 LICENSE 文件。如需再分发条款，请联系维护者。

---

## 贡献指南

1. 面板/MCP 可见文案保持中英双语，并在适用时附带两份 Markdown 文档。
2. 插件业务代码不要向 stdout 打印内容。
3. 优先在 `flowgo-nodes/` 下按目录拆分独立插件 module。
4. 发布二进制时匹配目标服务端 OS；若新增插件 README，请注明所需 GOOS/GOARCH。
