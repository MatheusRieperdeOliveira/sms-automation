package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"sms-automation/src/internal/file"
)

const filesDir = "./src/files"

func main() {
	path, err := selectFile(filesDir)
	if err != nil {
		log.Fatal(err)
	}

	selected := file.New(path)
	selected.Open()
	selected.Process()
	selected.Close()
}

func selectFile(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("não foi possível ler o diretório %s: %w", dir, err)
	}

	var files []string
	for _, entry := range entries {
		if entry.Type().IsRegular() {
			files = append(files, entry.Name())
		}
	}

	if len(files) == 0 {
		return "", fmt.Errorf("nenhum arquivo encontrado em %s", dir)
	}

	fmt.Println("Arquivos disponíveis:")
	for i, name := range files {
		fmt.Printf("[%d] %s\n", i+1, name)
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Selecione o número do arquivo: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("não foi possível ler a seleção: %w", err)
		}

		choice, err := strconv.Atoi(strings.TrimSpace(input))
		if err != nil || choice < 1 || choice > len(files) {
			fmt.Printf("Opção inválida. Digite um número entre 1 e %d.\n", len(files))
			continue
		}

		return filepath.Join(dir, files[choice-1]), nil
	}
}
