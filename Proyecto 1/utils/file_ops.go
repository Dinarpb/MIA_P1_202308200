package utils

import (
	"os"
)

func LeerArchivoUsers(path string) string {
	data, _ := os.ReadFile(path)
	return string(data)
}

func EscribirArchivoUsers(path string, contenido string) {
	os.WriteFile(path, []byte(contenido), 0644)
}

func ObtenerNuevoUID() int {
	return 2 // Aquí deberías implementar la lógica de contar usuarios en el archivo
}

func ObtenerNuevoGID() int {
	return 2 // Aquí deberías implementar la lógica de contar grupos en el archivo
}
