package filesystem

import (
	"MIAP1/global"
	"MIAP1/utils"
	"fmt"
	"strings"
)

func Cat(archivos []string) {
	if len(archivos) == 0 {
		return
	}

	for _, archivo := range archivos {
		fmt.Printf("Contenido de %s:\n", archivo)
		fmt.Println("...")
	}
}

func Mkgrp(name string) {
	if !global.SesionActiva {
		fmt.Println("Error: No hay sesión activa.")
		return
	}
	if global.UsuarioActual != "root" {
		fmt.Println("Error: Solo root puede crear grupos.")
		return
	}

	contenido := utils.LeerArchivoUsers(global.IdParticionActual)
	if strings.Contains(contenido, ", G, "+name) {
		fmt.Println("Error: El grupo ya existe.")
		return
	}

	nuevoGrupo := fmt.Sprintf("\n%d, G, %s", utils.ObtenerNuevoGID(), name)
	utils.EscribirArchivoUsers(global.IdParticionActual, contenido+nuevoGrupo)
	fmt.Println("Grupo creado exitosamente.")
}

func Rmgrp(name string) {
	if !global.SesionActiva {
		fmt.Println("Error: No hay sesión activa.")
		return
	}
	if global.UsuarioActual != "root" {
		fmt.Println("Error: Solo root puede eliminar grupos.")
		return
	}

	contenido := utils.LeerArchivoUsers(global.IdParticionActual)
	lineaGrupo := fmt.Sprintf(", G, %s", name)

	if !strings.Contains(contenido, lineaGrupo) {
		fmt.Println("Error: El grupo no existe.")
		return
	}

	lineas := strings.Split(contenido, "\n")
	var nuevoContenido []string
	for _, linea := range lineas {
		if !strings.Contains(linea, lineaGrupo) {
			nuevoContenido = append(nuevoContenido, linea)
		}
	}

	utils.EscribirArchivoUsers(global.IdParticionActual, strings.Join(nuevoContenido, "\n"))
	fmt.Println("Grupo eliminado exitosamente.")
}
