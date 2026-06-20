package users

import (
	"MIAP1/utils"
	"fmt"
	"strings"
)

var SesionActiva bool = false
var UsuarioActual string = ""
var IdParticionActual string = ""

func Login(user string, pass string, id string) {
	if SesionActiva {
		fmt.Println("Error: Ya existe una sesión activa. Cierre sesión antes de iniciar otra.")
		return
	}

	SesionActiva = true
	UsuarioActual = user
	IdParticionActual = id

	fmt.Printf("Sesión iniciada exitosamente para el usuario '%s' en la partición '%s'.\n", user, id)
}

func Logout() {
	if !SesionActiva {
		fmt.Println("Error: No hay ninguna sesión activa en este momento.")
		return
	}

	fmt.Printf("Sesión del usuario '%s' cerrada exitosamente.\n", UsuarioActual)

	SesionActiva = false
	UsuarioActual = ""
	IdParticionActual = ""
}

func Mkgrp(name string) {
	if !SesionActiva {
		fmt.Println("Error: No hay ninguna sesión activa. Inicie sesión para ejecutar este comando.")
		return
	}

	if UsuarioActual != "root" {
		fmt.Println("Error: Solo el usuario root tiene permisos para ejecutar MKGRP.")
		return
	}

	fmt.Printf("Ejecutando MKGRP para crear el grupo: %s\n", name)
}

func Rmgrp(name string) {
	if !SesionActiva {
		fmt.Println("Error: No hay ninguna sesión activa. Inicie sesión para ejecutar este comando.")
		return
	}

	if UsuarioActual != "root" {
		fmt.Println("Error: Solo el usuario root tiene permisos para ejecutar RMGRP.")
		return
	}

	fmt.Printf("Ejecutando RMGRP para eliminar el grupo: %s\n", name)
}

func Mkusr(user string, pass string, grp string) {
	if !SesionActiva {
		fmt.Println("Error: No hay sesión activa.")
		return
	}
	if UsuarioActual != "root" {
		fmt.Println("Error: Solo root puede crear usuarios.")
		return
	}

	contenido := utils.LeerArchivoUsers(IdParticionActual)

	if strings.Contains(contenido, ", U, "+user) {
		fmt.Println("Error: El usuario ya existe.")
		return
	}

	if !strings.Contains(contenido, ", G, "+grp) {
		fmt.Println("Error: El grupo especificado no existe.")
		return
	}

	nuevoUsuario := fmt.Sprintf("\n%d, U, %s, %s, %s", utils.ObtenerNuevoUID(), grp, user, pass)
	utils.EscribirArchivoUsers(IdParticionActual, contenido+nuevoUsuario)
	fmt.Println("Usuario creado exitosamente.")
}

func Rmusr(user string) {
	if !SesionActiva || UsuarioActual != "root" {
		fmt.Println("Error: Permisos insuficientes o sesión no activa.")
		return
	}

	contenido := utils.LeerArchivoUsers(IdParticionActual)
	if !strings.Contains(contenido, ", U, "+user) {
		fmt.Println("Error: El usuario no existe.")
		return
	}

	lineas := strings.Split(contenido, "\n")
	var nuevoContenido []string
	for _, linea := range lineas {
		if !strings.Contains(linea, ", U, "+user) {
			nuevoContenido = append(nuevoContenido, linea)
		}
	}
	utils.EscribirArchivoUsers(IdParticionActual, strings.Join(nuevoContenido, "\n"))
	fmt.Println("Usuario eliminado exitosamente.")
}

func Chgrp(user string, grp string) {
	if !SesionActiva || UsuarioActual != "root" {
		fmt.Println("Error: Permisos insuficientes o sesión no activa.")
		return
	}

	contenido := utils.LeerArchivoUsers(IdParticionActual)
	if !strings.Contains(contenido, ", U, "+user) || !strings.Contains(contenido, ", G, "+grp) {
		fmt.Println("Error: Usuario o grupo no existen.")
		return
	}

	lineas := strings.Split(contenido, "\n")
	for i, linea := range lineas {
		if strings.Contains(linea, ", U, "+user) {
			partes := strings.Split(linea, ", ")
			partes[2] = grp
			lineas[i] = strings.Join(partes, ", ")
			break
		}
	}
	utils.EscribirArchivoUsers(IdParticionActual, strings.Join(lineas, "\n"))
	fmt.Println("Grupo cambiado exitosamente.")
}
