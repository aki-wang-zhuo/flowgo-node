module github.com/flowgo/flowgo-node/nodes/echo

go 1.22

require (
	github.com/flowgo/flowgo v0.0.0
	github.com/flowgo/flowgo-node/sdk v0.0.0
)

replace github.com/flowgo/flowgo => ../../../flowgo

replace github.com/flowgo/flowgo-node/sdk => ../../sdk
