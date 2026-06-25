package main

import (
	"MIAP1/analyzer"
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Println("\n=================================================")
		fmt.Println("      SISTEMA DE ARCHIVOS EXT2 - FASE 1")
		fmt.Println("=================================================")
		fmt.Println("1. Ejecutar script de comandos")
		fmt.Println("2. Salir")
		fmt.Print("\nElige una opción: ")

		scanner.Scan()
		opcion := strings.TrimSpace(scanner.Text())

		switch opcion {
		case "1":
			fmt.Print("Ingresa la ruta absoluta del archivo (ej. /home/dinaarpb/cali1.txt): ")
			scanner.Scan()
			ruta := strings.TrimSpace(scanner.Text())

			if ruta != "" {
				comando := fmt.Sprintf("execute -path=%s", ruta)
				fmt.Println("\n[INFO] Procesando script:", ruta)
				analyzer.AnalizarComando(comando)
			} else {
				fmt.Println("Error: La ruta no puede estar vacía.")
			}

		case "2":
			fmt.Println("Cerrando el sistema. ¡Hasta pronto!")
			return

		default:
			fmt.Println("Opción no válida. Por favor, ingresa 1 o 2.")
		}
	}
}
