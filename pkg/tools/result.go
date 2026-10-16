package tools

import (
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolSuccess returns a successful CallToolResult. If output is provided,
// the SDK's default structured output handling can be used or custom content returned.
func ToolSuccess[T any](output *T) (*mcp.CallToolResult, *T, error) {
	return nil, output, nil
}

// ToolSuccessWithImage returns a successful CallToolResult containing text/JSON summary and image data.
func ToolSuccessWithImage[T any](output *T, imgData []byte, mimeType string, textSummary string) (*mcp.CallToolResult, *T, error) {
	var contents []mcp.Content
	if textSummary != "" {
		contents = append(contents, &mcp.TextContent{Text: textSummary})
	}
	if output != nil {
		if jsonBytes, err := json.Marshal(output); err == nil {
			contents = append(contents, &mcp.TextContent{Text: string(jsonBytes)})
		}
	}
	if len(imgData) > 0 {
		contents = append(contents, &mcp.ImageContent{
			Data:     imgData,
			MIMEType: mimeType,
		})
	}
	return &mcp.CallToolResult{
		Content: contents,
	}, output, nil
}

// ToolError returns a CallToolResult with IsError: true and descriptive explanation.
// In the MCP protocol, tool execution failures must be reported as tool results with IsError: true
// so AI clients can inspect the reason and self-correct, rather than receiving JSON-RPC -32603 faults.
func ToolError[T any](userMessage string, err error) (*mcp.CallToolResult, *T, error) {
	msg := userMessage
	if err != nil {
		if msg != "" {
			msg = fmt.Sprintf("%s: %v", msg, err)
		} else {
			msg = err.Error()
		}
	}
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}, nil, nil
}
