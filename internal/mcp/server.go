package mcp

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"femas66/markdown-gokil/internal/converter"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ConvertArgs struct {
	InputPath  string `json:"inputPath" jsonschema:"The absolute or relative path to the input .docx file to convert"`
	OutputPath string `json:"outputPath,omitempty" jsonschema:"The optional path where the markdown file will be saved. If omitted, saves in the same directory as inputPath with a .md extension"`
}

func StartServer() error {
	// Redirect standard log to stderr so that logs do not corrupt the stdio transport
	log.SetOutput(os.Stderr)

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "markdown-gokil",
		Version: "1.0.0",
	}, nil)

	// Add the convert_docx tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "convert_docx",
		Description: "Convert a DOCX file to markdown format, extracting images into an outputs folder",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args ConvertArgs) (*mcp.CallToolResult, any, error) {
		if args.InputPath == "" {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: "Error: inputPath is required"},
				},
				IsError: true,
			}, nil, nil
		}

		engine := converter.NewEngine()
		
		outputPath := args.OutputPath
		if outputPath == "" {
			ext := filepath.Ext(args.InputPath)
			outputPath = args.InputPath[0:len(args.InputPath)-len(ext)]
		}
		if filepath.Ext(outputPath) != ".md" {
			outputPath += ".md"
		}

		finalPath, err := engine.Convert(args.InputPath, outputPath)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Error during conversion: %v", err)},
				},
				IsError: true,
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Conversion successful! Output markdown saved to: %s", finalPath)},
			},
		}, nil, nil
	})

	log.Println("Starting MCP server on stdio...")
	return server.Run(context.Background(), &mcp.StdioTransport{})
}
