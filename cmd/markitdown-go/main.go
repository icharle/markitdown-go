package main

import (
	"fmt"
	"log"
	"os"

	"github.com/icharle/markitdown-go/pkg/markitdown"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: markitdown-go <file>")
		os.Exit(1)
	}

	filePath := os.Args[1]

	// 调用核心功能
	result, err := markitdown.Convert(filePath)
	if err != nil {
		log.Fatalf("Error converting file: %v", err)
	}

	fmt.Println("Markdown Output:")
	fmt.Println(result)
}
