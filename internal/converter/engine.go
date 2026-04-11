package converter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"femas66/markdown-gokil/internal/docx"
	"femas66/markdown-gokil/internal/markdown"
)

// Engine handles the conversion process
type Engine struct{}

func NewEngine() *Engine {
	return &Engine{}
}

// Convert parses docx and writes markdown output
func (e *Engine) Convert(inputPath, outputPath string) (string, error) {
	// Get filename without extension for folder name
	baseName := filepath.Base(outputPath)
	ext := filepath.Ext(baseName)
	folderName := strings.TrimSuffix(baseName, ext)

	// Ensure 'outputs/' is the root storage folder
	outputRootDir := filepath.Dir(outputPath)
	baseOutputDir := filepath.Join(outputRootDir, "outputs")

	// Final output folder path
	finalFolder := filepath.Join(baseOutputDir, folderName)

	// Check if output folder already exists
	if _, err := os.Stat(finalFolder); !os.IsNotExist(err) {
		return "", fmt.Errorf("output folder '%s' already exists. Use a different name or delete the folder", finalFolder)
	}

	// Set final paths for .md file and images folder
	finalMDPath := filepath.Join(finalFolder, baseName)
	imageDir := filepath.Join(finalFolder, "images")

	// Process docx: unzip, extract images, map relations, and parse content
	doc, err := docx.Parse(inputPath, imageDir)
	if err != nil {
		return "", fmt.Errorf("failed to process docx: %w", err)
	}

	// Generate Markdown content
	mdContent, err := markdown.FromDocument(doc)
	if err != nil {
		return "", fmt.Errorf("failed to generate markdown: %w", err)
	}

	// Create output directory and its parents
	if err := os.MkdirAll(finalFolder, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write markdown to file in the output folder
	err = os.WriteFile(finalMDPath, []byte(mdContent), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write output file: %w", err)
	}

	return finalMDPath, nil
}
