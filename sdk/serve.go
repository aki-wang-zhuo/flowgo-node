package sdk

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/flowgo/flowgo/api/types"
)

// Serve 以插件进程身份运行：从 stdin 读请求、向 stdout 写响应。
// 须在 main 中调用；勿向 stdout 打印其它日志。
func Serve(regs ...Registration) error {
	byType := make(map[string]Registration, len(regs))
	defs := make([]WireDef, 0, len(regs))
	for _, r := range regs {
		if r.Def.Type == "" || r.Factory == nil {
			continue
		}
		byType[r.Def.Type] = r
		defs = append(defs, ToWire(r.Def))
	}
	if len(byType) == 0 {
		return fmt.Errorf("no node registrations")
	}

	instances := map[string]types.Node{}
	var mu sync.Mutex

	in := bufio.NewReader(os.Stdin)
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)

	write := func(env Envelope) error {
		return enc.Encode(env)
	}

	for {
		line, err := in.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		var req Envelope
		if err := json.Unmarshal(line, &req); err != nil {
			_ = write(Envelope{Op: OpError, Error: "invalid json: " + err.Error()})
			continue
		}
		switch req.Op {
		case OpHello:
			typesList := make([]string, 0, len(byType))
			for t := range byType {
				typesList = append(typesList, t)
			}
			_ = write(Envelope{ID: req.ID, Op: OpHelloOK, Protocol: ProtocolVersion, Types: typesList})
		case OpListDefs:
			_ = write(Envelope{ID: req.ID, Op: OpListDefsOK, Defs: defs})
		case OpCreate:
			reg, ok := byType[req.Type]
			if !ok {
				_ = write(Envelope{ID: req.ID, Op: OpError, Error: "unknown type: " + req.Type})
				continue
			}
			if req.InstanceID == "" {
				_ = write(Envelope{ID: req.ID, Op: OpError, Error: "instanceId required"})
				continue
			}
			node := reg.Factory()
			mu.Lock()
			instances[req.InstanceID] = node
			mu.Unlock()
			_ = write(Envelope{ID: req.ID, Op: OpCreateOK, InstanceID: req.InstanceID})
		case OpInit:
			mu.Lock()
			node, ok := instances[req.InstanceID]
			mu.Unlock()
			if !ok {
				_ = write(Envelope{ID: req.ID, Op: OpError, Error: "unknown instance"})
				continue
			}
			if err := node.Init(req.Config); err != nil {
				_ = write(Envelope{ID: req.ID, Op: OpError, Error: err.Error()})
				continue
			}
			_ = write(Envelope{ID: req.ID, Op: OpInitOK, InstanceID: req.InstanceID})
		case OpOnMsg:
			mu.Lock()
			node, ok := instances[req.InstanceID]
			mu.Unlock()
			if !ok {
				_ = write(Envelope{ID: req.ID, Op: OpOnMsgErr, Error: "unknown instance"})
				continue
			}
			if req.Msg == nil {
				_ = write(Envelope{ID: req.ID, Op: OpOnMsgErr, Error: "msg required"})
				continue
			}
			out, rel, err := node.OnMsg(context.Background(), *req.Msg)
			if err != nil {
				_ = write(Envelope{ID: req.ID, Op: OpOnMsgErr, Error: err.Error()})
				continue
			}
			msg := out
			_ = write(Envelope{ID: req.ID, Op: OpOnMsgOK, Msg: &msg, Relation: rel})
		case OpDestroy:
			mu.Lock()
			node, ok := instances[req.InstanceID]
			if ok {
				delete(instances, req.InstanceID)
			}
			mu.Unlock()
			if ok {
				node.Destroy()
			}
			_ = write(Envelope{ID: req.ID, Op: OpDestroyOK, InstanceID: req.InstanceID})
		default:
			_ = write(Envelope{ID: req.ID, Op: OpError, Error: "unknown op: " + req.Op})
		}
	}
}
