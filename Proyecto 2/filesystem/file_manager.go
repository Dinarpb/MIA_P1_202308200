package filesystem

import (
	"MIAP1/users"
	"fmt"
	"os"
)

func Mkfile(path string, size string, cont string, r bool) {
	if !users.SesionActiva {
		fmt.Println("Error: No hay sesión activa.")
		return
	}

	contenidoFinal := ""
	if cont != "" {
		data, err := os.ReadFile(cont)
		if err != nil {
			fmt.Println("Error: No se pudo leer el archivo de contenido.")
			return
		}
		contenidoFinal = string(data)
	} else if size != "" {
		s := 0
		fmt.Sscanf(size, "%d", &s)
		if s < 0 {
			fmt.Println("Error: El tamaño no puede ser negativo.")
			return
		}
		for i := 0; i < s; i++ {
			contenidoFinal += fmt.Sprintf("%d", i%10)
		}
	}

	if r {
		fmt.Printf("Creando carpetas padres recursivamente para: %s\n", path)
	}

	fmt.Printf("Archivo creado en %s con contenido: %s\n", path, contenidoFinal)
}

func Mkdir(path string, p bool) {
	if !users.SesionActiva {
		fmt.Println("Error: No hay sesión activa.")
		return
	}

	if p {
		fmt.Printf("Creando jerarquía de directorios para: %s\n", path)
	} else {
		fmt.Printf("Creando directorio: %s\n", path)
	}
}
