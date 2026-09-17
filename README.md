# FlowGo Node

[中文文档](./README_ZH.md)

**FlowGo Node** provides the **out-of-process plugin SDK** and example nodes for FlowGo. Plugins are standalone executables that speak a **JSON-lines RPC protocol** over stdin/stdout. [flowgo-server](https://github.com/aki-wang-zhuo/flowgo-server) hosts them, registers proxy nodes into the [flowgo](https://github.com/aki-wang-zhuo/flowgo) engine registry, and can hot-load / enable / uninstall packages from the editor or REST API.

> **Built-in nodes belong in `flowgo`, not here.** This repository is for third-party / importable extensions only.

| Related project | Role |
| --- | --- |
| [flowgo](https://github.com/aki-wang-zhuo/flowgo) | Engine + built-in nodes |
| [flowgo-server](https://github.com/aki-wang-zhuo/flowgo-server) | Plugin host (`pluginhost`) |
| [flowgo-editor](https://github.com/aki-wang-zhuo/flowgo-editor) | UI for install / enable / docs |

---

## Features

- **Process isolation** — each plugin is an OS executable (not Go `plugin` `.so`/`.dll`)
- **Shared SDK** — plugin side (`Serve`) and host side (`Client` + `ProxyNode`) in one module
- **Component metadata** — bilingual labels, descriptions, config fields, docs (same `ComponentDef` model as built-ins)
- **Lifecycle RPC** — hello → list_defs → create → init → on_msg → destroy
- **Install formats** — raw binary or zip (binary + `*_zh.md` / `*_en.md` docs)
- **Example** — `pluginEcho` under `flowgo-nodes/echo`

---

## Requirements

- Go **1.22+**
- Local [flowgo](https://github.com/aki-wang-zhuo/flowgo) checkout for `replace` during development
- Plugin binary **OS/arch must match** the server host (Windows PE / Linux ELF)

Suggested monorepo layout:

```text
RuleGo/
├── flowgo/
├── flowgo-server/
├── flowgo-node/     ← this repo
└── flowgo-editor/
```

---

## Repository layout

```text
flowgo-node/
├── sdk/                         # github.com/flowgo/flowgo-node/sdk
│   ├── protocol.go              # Envelope, WireDef, ops, version
│   ├── serve.go                 # Plugin main loop (stdin/stdout)
│   ├── client.go                # Host: start process + RPC
│   ├── proxy.go                 # ProxyNode / NewProxyFactory
│   └── client_test.go
└── flowgo-nodes/
    └── echo/                    # Example plugin (own go.mod)
        ├── main.go
        ├── build.ps1
        ├── pluginEcho_zh.md
        ├── pluginEcho_en.md
        └── dist/                # build output (gitignored)
```

There is **no** root `go.mod`. The SDK and each plugin are separate modules.

---

## Quick start — author a plugin

### 1. Implement `types.Node`

```go
type EchoNode struct {
    prefix string
}

func (n *EchoNode) Type() string { return "pluginEcho" }

func (n *EchoNode) Init(config map[string]interface{}) error {
    // read config["prefix"], etc.
    return nil
}

func (n *EchoNode) OnMsg(ctx context.Context, msg types.Msg) (types.Msg, string, error) {
    msg.Data = n.prefix + msg.Data
    return msg, types.RelationSuccess, nil
}

func (n *EchoNode) Destroy() {}
```

### 2. Declare bilingual `ComponentDef`

Provide Chinese defaults plus English entries in `Labels` / `Descriptions` / `Docs` (and field descriptions). Ship `pluginEcho_zh.md` / `pluginEcho_en.md` next to the binary when packaging.

### 3. Call `sdk.Serve` from `main`

```go
func main() {
    sdk.Serve(sdk.Registration{
        Def:     Def,
        Factory: func() types.Node { return &EchoNode{} },
    })
}
```

One process may register multiple types via multiple `Registration` values.

### 4. Build & package

```powershell
cd flowgo-nodes/echo
.\build.ps1
# → dist/echo.exe (or echo), docs, and echo.zip
```

### 5. Install on the server

- **Editor**: Settings → node/plugin management → upload `.exe` or `.zip`
- **API**: `POST /api/components/plugins/load` (multipart `file`, authenticated)

Plugins land under `flowgo-server/data/plugins/<id>/` with a `manifest.json`.

---

## Protocol (overview)

| Request `op` | Success response | Purpose |
| --- | --- | --- |
| `hello` | `hello_ok` | Handshake; returns protocol version + types |
| `list_defs` | `list_defs_ok` | Return `WireDef[]` metadata |
| `create` | `create_ok` | Create instance by type + instanceId |
| `init` | `init_ok` | Apply configuration |
| `on_msg` | `on_msg_ok` / `on_msg_err` | Handle message; return msg + relation |
| `destroy` | `destroy_ok` | Destroy instance |

- Protocol version: **1** (`ProtocolVersion`)
- Framing: **one JSON object per line** on stdin/stdout
- Host sets env `FLOWGO_PLUGIN=1`
- **Never write logs to stdout** (it is the RPC channel). Use stderr if needed.

---

## SDK APIs

### Plugin side

| API | Description |
| --- | --- |
| `sdk.Serve(regs ...Registration)` | Main loop |
| `sdk.Registration{Def, Factory}` | Metadata + `types.NodeFactory` |
| `sdk.ToWire` / `FromWire` | `ComponentDef` ↔ JSON-safe `WireDef` |

### Host side (used by flowgo-server)

| API | Description |
| --- | --- |
| `sdk.Start(binPath)` | Spawn process, handshake, fetch defs |
| `(*Client).Create / Init / OnMsg / Destroy / Close` | Instance lifecycle |
| `sdk.NewProxyFactory(client, typeName)` | Factory for `engine.Registry.Register` |
| `sdk.ProxyNode` | Implements `types.Node` by forwarding RPC |

Module path:

```text
github.com/flowgo/flowgo-node/sdk
```

---

## Example: `pluginEcho`

| Field | Value |
| --- | --- |
| Type | `pluginEcho` |
| Behavior | Prepend a configurable `prefix` to `msg.Data` |
| Default prefix | `"[echo] "` |
| Relation | `Success` |
| Docs | `pluginEcho_zh.md` / `pluginEcho_en.md` |

Integration test: `sdk/client_test.go` (`TestEchoPluginRoundTrip`).

---

## Server lifecycle (reference)

1. On startup, `pluginhost` scans `data/plugins/` and `LoadAll()`.
2. For each plugin: start binary → `hello` + `list_defs` → register proxy factories.
3. Engine execution calls `ProxyNode.OnMsg`, which RPCs into the plugin process.
4. Enable/disable/uninstall:

| Method | Path |
| --- | --- |
| `POST` | `/api/components/plugins/load` |
| `PUT` | `/api/components/plugins/{id}/enabled` |
| `DELETE` | `/api/components/plugins/{id}` |

Constraints:

- Type names must not collide with **built-in** node types (plugin types may replace previous plugin registrations).
- `disabled: true` keeps files on disk but does not start the process.

---

## Testing

```bash
cd sdk
go test ./...
```

Echo tests may compile the example binary on demand when it is missing.

---

## License

License file is not yet published in this repository. Contact the maintainers if you need redistributable terms.

---

## Contributing

1. Keep palette/MCP-visible strings bilingual (zh-CN + en-US) and ship both Markdown docs when applicable.
2. Do not print to stdout from plugin business code.
3. Prefer one plugin module per directory under `flowgo-nodes/`.
4. Match the server OS when producing release binaries; document required GOOS/GOARCH in your plugin README if you add one.
