package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"femas66/markdown-gokil/internal/converter"
)

func main() {
	// Parse flags for any possible future flags, but we mainly care about positional args now
	flag.Usage = func() {
		fmt.Println("Usage: markdown-gokil <input.docx> [output_name]")
	}
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	inputPath := args[0]

	// 1. Validate file extension
	if strings.ToLower(filepath.Ext(inputPath)) != ".docx" {
		log.Fatalf("Error: Input file must have .docx extension")
	}

	// 2. Validate file existence and type
	info, err := os.Stat(inputPath)
	if os.IsNotExist(err) {
		log.Fatalf("Error: File '%s' not found", inputPath)
	}
	if info.IsDir() {
		log.Fatalf("Error: '%s' is a directory, not a file", inputPath)
	}

	var outputPath string

	if len(args) >= 2 {
		// Use second argument as output path
		outputPath = args[1]
	}

	// Default output path if not provided
	if outputPath == "" {
		ext := filepath.Ext(inputPath)
		outputPath = inputPath[0:len(inputPath)-len(ext)]
	}

	// Ensure .md extension for proper conversion logic
	if filepath.Ext(outputPath) != ".md" {
		outputPath += ".md"
	}

	fmt.Printf("🚀 Converting %s to %s...\n", inputPath, outputPath)

	engine := converter.NewEngine()
	err = engine.Convert(inputPath, outputPath)
	if err != nil {
		log.Fatalf("❌ Error during conversion: %v", err)
	}

	fmt.Println("✨ Conversion successful!")
}
