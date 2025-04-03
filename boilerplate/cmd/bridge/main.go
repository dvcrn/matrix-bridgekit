package main

import (
	"context"

	"github.com/dvcrn/bridgekit-boilerplate/internal"
	"github.com/dvcrn/matrix-bridgekit/bridgekit"
)

func main() {
	br := bridgekit.NewBridgeKit(
		"MyBridge",
		"sh-mybridge",
		"",
		"Integration",
		"1.0",
		&internal.Config{},
		internal.ExampleConfig,
	)
	connector := internal.NewBridgeConnector(br)
	br.StartBridgeConnector(context.Background(), connector)
}
