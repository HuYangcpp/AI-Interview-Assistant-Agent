package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"ai-gozero-agent/doc-processor/internal/mcp"

	"github.com/zeromicro/go-zero/core/logx"
)

func main() {
	// Redirect all logx output to stderr so it doesn't corrupt the JSON-RPC on stdout
	logx.SetWriter(logx.NewWriter(os.Stderr))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := mcp.HandleStdio(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "MCP Server Error: %v\n", err)
		os.Exit(1)
	}
}
