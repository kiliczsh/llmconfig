package cmd

import (
	"fmt"
	"net/http"

	"github.com/kiliczsh/llmconfig/internal/gateway"
	"github.com/spf13/cobra"
)

func newGatewayCmd() *cobra.Command {
	var port int

	cmd := &cobra.Command{
		Use:   "gateway",
		Short: "Start a unified API gateway for all running models",
		Long: `Start a single OpenAI-compatible endpoint that routes requests
to the correct model based on the "model" parameter.

Example:
  llmconfig gateway --port 4000

  curl http://localhost:4000/v1/chat/completions \
    -H "Content-Type: application/json" \
    -d '{"model":"llama3","messages":[{"role":"user","content":"hello"}]}'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			appCtx := appCtxFrom(cmd.Context())
			proxy := gateway.New(appCtx.StateStore)
			addr := fmt.Sprintf(":%d", port)
			appCtx.Printer.Info("gateway listening on %s", addr)
			return http.ListenAndServe(addr, proxy.Handler())
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 4000, "port to listen on")
	return cmd
}
