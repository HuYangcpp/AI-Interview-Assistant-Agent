package svc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type PdfClient struct {
	binaryPath string
}

func NewPdfClient(endpoint string) *PdfClient {
	// Use the 'endpoint' config as the binary path.
	binaryPath := endpoint
	if binaryPath == "" {
		binaryPath = "./doc-processor"
	}

	return &PdfClient{
		binaryPath: binaryPath,
	}
}

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   interface{}     `json:"error,omitempty"`
}

// ExtractText invokes the MCP DocProcessor as a subprocess and uses JSON-RPC over stdio.
func (c *PdfClient) ExtractText(file multipart.File, filename string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Create a temporary file to pass to the MCP tool (MCP extract_pdf_text takes a file_path)
	tempDir := os.TempDir()
	tempFile, err := os.CreateTemp(tempDir, "upload-*.pdf")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, file); err != nil {
		return "", fmt.Errorf("failed to save temp file: %v", err)
	}

	// 2. Start MCP DocProcessor process
	cmd := exec.CommandContext(ctx, c.binaryPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start MCP process: %v", err)
	}
	defer cmd.Process.Kill()

	// 3. MCP Handshake: Initialize
	initReq := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  json.RawMessage(`{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"api-service","version":"1.0.0"}}`),
	}
	if err := json.NewEncoder(stdin).Encode(initReq); err != nil {
		return "", err
	}

	decoder := json.NewDecoder(stdout)
	var initResp JSONRPCResponse
	if err := decoder.Decode(&initResp); err != nil {
		return "", fmt.Errorf("failed to decode initialize response: %v", err)
	}

	// 4. Call Tool: extract_pdf_text
	absPath, _ := filepath.Abs(tempFile.Name())
	params := map[string]interface{}{
		"name": "extract_pdf_text",
		"arguments": map[string]string{
			"file_path": absPath,
		},
	}
	paramsJSON, _ := json.Marshal(params)
	callReq := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/call",
		Params:  paramsJSON,
	}

	if err := json.NewEncoder(stdin).Encode(callReq); err != nil {
		return "", err
	}

	// 5. Parse Result
	var callResp JSONRPCResponse
	if err := decoder.Decode(&callResp); err != nil {
		return "", fmt.Errorf("failed to decode call response: %v", err)
	}

	if callResp.Error != nil {
		return "", fmt.Errorf("MCP Error: %v", callResp.Error)
	}

	// MCP Tool Result structure: { "content": [ { "type": "text", "text": "..." } ] }
	var result map[string]interface{}
	if err := json.Unmarshal(callResp.Result, &result); err != nil {
		return "", err
	}

	if isError, ok := result["isError"].(bool); ok && isError {
		return "", fmt.Errorf("MCP tool execution error")
	}

	contents, ok := result["content"].([]interface{})
	if !ok || len(contents) == 0 {
		return "", fmt.Errorf("empty MCP result")
	}

	contentMap, ok := contents[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid MCP content format")
	}

	return contentMap["text"].(string), nil
}
