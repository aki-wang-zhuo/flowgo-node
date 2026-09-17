package sdk_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/flowgo/flowgo/api/types"
	"github.com/flowgo/flowgo-node/sdk"
)

func TestEchoPluginRoundTrip(t *testing.T) {
	bin := filepath.Join("..", "flowgo-nodes", "echo", "dist", "echo")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	if _, err := os.Stat(bin); err != nil {
		// 尝试现场编译
		dir := filepath.Join("..", "flowgo-nodes", "echo")
		out := filepath.Join(dir, "dist")
		_ = os.MkdirAll(out, 0o755)
		cmd := exec.Command("go", "build", "-o", bin, ".")
		cmd.Dir = dir
		if outb, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("echo plugin not built: %v\n%s", err, outb)
		}
	}
	client, err := sdk.Start(bin)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if len(client.Defs()) == 0 {
		t.Fatal("no defs")
	}
	id := "t1"
	if err := client.Create("pluginEcho", id); err != nil {
		t.Fatal(err)
	}
	if err := client.Init(id, map[string]interface{}{"prefix": "X:"}); err != nil {
		t.Fatal(err)
	}
	out, rel, err := client.OnMsg(nil, id, types.NewMsg("T", types.TEXT, "hi", nil))
	if err != nil {
		t.Fatal(err)
	}
	if rel != types.RelationSuccess || out.Data != "X:hi" {
		t.Fatalf("got data=%q rel=%q", out.Data, rel)
	}
}
