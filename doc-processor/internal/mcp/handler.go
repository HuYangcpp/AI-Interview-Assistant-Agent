package mcp

import (
	"ai-gozero-agent/doc-processor/internal/utils"
	"context"
	"encoding/json"
	"fmt"
	"os"
)

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type CallToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type ExtractPdfArgs struct {
	FilePath string `json:"file_path"`
}

func HandleStdio(ctx context.Context) error {
	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			var req JSONRPCRequest
			if err := decoder.Decode(&req); err != nil {
				return err
			}

			resp := JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
			}

			switch req.Method {
			case "initialize":
				resp.Result = map[string]interface{}{
					"protocolVersion": "2024-11-05",
					"capabilities":    map[string]interface{}{},
					"serverInfo": map[string]string{
						"name":    "doc-processor",
						"version": "1.0.0",
					},
				}
			case "tools/list":
				resp.Result = map[string]interface{}{
					"tools": []interface{}{
						map[string]interface{}{
							"name":        "extract_pdf_text",
							"description": "Extract text from a PDF file",
							"inputSchema": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"file_path": map[string]string{"type": "string"},
								},
								"required": []string{"file_path"},
							},
						},
					},
				}
			case "tools/call":
				var params CallToolParams
				if err := json.Unmarshal(req.Params, &params); err != nil {
					resp.Error = &RPCError{Code: -32602, Message: "Invalid params"}
				} else if params.Name == "extract_pdf_text" {
					var args ExtractPdfArgs
					if err := json.Unmarshal(params.Arguments, &args); err != nil {
						resp.Error = &RPCError{Code: -32602, Message: "Invalid arguments"}
					} else {
						content, err := processPdf(args.FilePath)
						if err != nil {
							resp.Result = map[string]interface{}{
								"content": []interface{}{
									map[string]string{"type": "text", "text": fmt.Sprintf("Error: %v", err)},
								},
								"isError": true,
							}
						} else {
							resp.Result = map[string]interface{}{
								"content": []interface{}{
									map[string]string{"type": "text", "text": content},
								},
							}
						}
					}
				} else {
					resp.Error = &RPCError{Code: -32601, Message: "Tool not found"}
				}
			case "notifications/initialized":
				continue // No response needed
			default:
				resp.Error = &RPCError{Code: -32601, Message: fmt.Sprintf("Method not found: %s", req.Method)}
			}

			if err := encoder.Encode(resp); err != nil {
				return err
			}
		}
	}
}

func processPdf(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return utils.ExtractPDFText(f)
}
