package main

import (
	"fmt"
	"os"

	"github.com/alpyxn/aeterna/backend/internal/services"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "generate":
		handleGenerate()
	case "validate":
		handleValidate()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Aeterna Encryption Key Management Tool

Usage: keytool <command>

Commands:
  generate    Generate a new encryption key (outputs to stdout)
  validate    Test key retrieval from configured source

Examples:
  # Generate a new key and save to file
  keytool generate > /secure/path/to/key
  chmod 600 /secure/path/to/key

  # Test key retrieval (tries Docker secrets, then file)
  keytool validate
`)
}

func handleGenerate() {
	key, err := services.GenerateKey()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating key: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(key)
}

func handleValidate() {
	// Try to initialize key manager with empty file path (will use Docker secrets if available)
	services.InitKeyManager("")

	cryptoService := services.CryptoService{}
	testData := "test validation"
	encrypted, err := cryptoService.Encrypt(testData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Key validation failed: %v\n", err)
		fmt.Fprintf(os.Stderr, "\nPlease configure one of the following:\n")
		fmt.Fprintf(os.Stderr, "  1. Docker secrets: mount key at /run/secrets/encryption_key\n")
		fmt.Fprintf(os.Stderr, "  2. Secure file: use --encryption-key-file flag (file must have 0600 permissions)\n")
		os.Exit(1)
	}

	decrypted, err := cryptoService.Decrypt(encrypted)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Key validation failed: encryption works but decryption failed: %v\n", err)
		os.Exit(1)
	}

	if decrypted != testData {
		fmt.Fprintf(os.Stderr, "Key validation failed: decrypted data does not match\n")
		os.Exit(1)
	}

	fmt.Println("Key validation successful")
	fmt.Println("Encryption and decryption working correctly")
}
