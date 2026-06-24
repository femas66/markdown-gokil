![](assets/gokil.jpg)

# Markdown Gokil (DOCX to MD Converter)

A robust Go-based utility to convert .docx documents into clean, structured Markdown (.md) files. Designed to preserve essential formatting from original documents.

## Key Features

- **Smart Text Conversion**: Converts paragraphs, bold, italic, and combined formatting.
- **Automatic Heading Detection**: Intelligently determines heading levels (#, ##, ###, ####) based on the original font size.
- **Table Support**: Automatically converts DOCX tables into clean Markdown table format.
- **List and Sub-list Handling**: Supports numbered lists (1, A, a, i) and bullet points with proper hierarchical (sub-list) support.
- **Image Extraction**: Automatically extracts images from the original document and saves them to an 'images' folder in the output directory with correct markdown links.
- **Organized Output Structure**: Each conversion creates a dedicated folder to keep markdown files and images organized.

## Project Structure

- `cmd/markdown-gokil/`: CLI entry point.
- `internal/docx/`: DOCX XML parsing logic, table, and numbering handling.
- `internal/converter/`: Conversion orchestration and output folder management.
- `internal/markdown/`: Writer for generating clean Markdown syntax.

## Installation

Ensure you have Go installed on your system.

```bash
go build -o build/markdown-gokil ./cmd/markdown-gokil/main.go
```

## Usage

Run the following command to convert a document:

```bash
# Otomatis membuat output berdasarkan nama file (misal: outputs/input/input.md)
./build/markdown-gokil input.docx

# Dengan folder output spesifik
./build/markdown-gokil input.docx hasil
```

The conversion output will be available in the `outputs/result/` folder.

## MCP (Model Context Protocol) Support

This application can run as an MCP server, allowing AI agents (such as Claude Desktop, Cursor, Windsurf, etc.) to use it as a tool to convert `.docx` files into Markdown.

### Tool: `convert_docx`
- **Arguments:**
  - `inputPath` (string, required): The absolute or relative path to the input `.docx` file to convert.
  - `outputPath` (string, optional): The optional path where the markdown file will be saved. If omitted, saves in `outputs/<foldername>/<filename>.md`.

### Running the MCP Server
To start the MCP server over stdio:
```bash
./build/markdown-gokil -mcp
```
or via `just`:
```bash
just mcp
```

### AI Agent Integration Example

#### Claude Desktop
Add the following to your `claude_desktop_config.json` (usually located at `~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):

```json
{
  "mcpServers": {
    "markdown-gokil": {
      "command": "/usr/local/bin/go",
      "args": [
        "run",
        "cmd/markdown-gokil/main.go",
        "-mcp"
      ],
      "cwd": "/Users/femasakbarfathurohim/Documents/Developments/GO/markdown-hebat"
    }
  }
}
```
> [!NOTE]
> Make sure to adjust the `command` (path to your `go` binary if not in path) and `cwd` (path to this project directory) in your config file.

## Development

This project uses Justfile (a modern alternative to Makefile) for task automation:

- `just build`: Compiles the application to `build/`.
- `just run <input.docx> [output_name]`: Runs the conversion directly.
- `just mcp`: Starts the MCP server over stdio.
- `just tidy`: Cleans up Go dependencies.
- `just clean`: Removes the build folder and binary.

## License
Created for fast and efficient document conversion.
