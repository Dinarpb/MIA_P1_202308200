package users

import (
	"MIAP1/utils"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
)

var SesionActiva bool = false
var UsuarioActual string = ""

func Login(user string, pass string, id string) {
	if SesionActiva {
		fmt.Println("[ERROR] Ya existe una sesión activa. Cierre sesión antes de iniciar otra.")
		return
	}

	utils.IdParticionActual = id

	archivo, sb, _, inodoUsers, _, err := utils.ObtenerContextoParticion()
	if err != nil {
		fmt.Println("[ERROR] No se pudo acceder a la partición o a txt para validar el login.")
		utils.IdParticionActual = ""
		return
	}
	defer archivo.Close()

	contenido := utils.LeerArchivoUsers(archivo, sb, inodoUsers)
	lineas := strings.Split(contenido, "\n")

	loginExitoso := false

	for _, linea := range lineas {
		if linea == "" {
			continue
		}
		datos := strings.Split(linea, ",")

		if len(datos) >= 5 && datos[1] == "U" && datos[0] != "0" {
			if datos[3] == user && datos[4] == pass {
				loginExitoso = true
				break
			}
		}
	}

	if loginExitoso {
		SesionActiva = true
		UsuarioActual = user
		fmt.Printf("[ÉXITO] Sesión iniciada para el usuario '%s' en la partición '%s'.\n", user, id)
	} else {
		utils.IdParticionActual = "" // Limpiamos la variable porque falló el login
		fmt.Printf("[ERROR] Usuario '%s' o contraseña incorrectos (o el usuario no existe).\n", user)
	}
}

func Logout() {
	if !SesionActiva {
		fmt.Println("[ERROR] No hay ninguna sesión activa en este momento.")
		return
	}
	fmt.Printf("[ÉXITO] Sesión del usuario '%s' cerrada exitosamente.\n", UsuarioActual)
	SesionActiva = false
	UsuarioActual = ""
	utils.IdParticionActual = ""
}

func Mkgrp(name string) {
	if !SesionActiva || UsuarioActual != "root" {
		fmt.Println("[ERROR] Solo el usuario root con sesión activa puede crear grupos.")
		return
	}

	archivo, sb, inodoIndex, inodoUsers, partStart, err := utils.ObtenerContextoParticion()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer archivo.Close()

	contenido := utils.LeerArchivoUsers(archivo, sb, inodoUsers)
	lineas := strings.Split(contenido, "\n")
	correlativoMaximo := 0

	for _, linea := range lineas {
		if linea == "" {
			continue
		}
		datos := strings.Split(linea, ",")
		if datos[1] == "G" {
			if datos[0] != "0" && datos[2] == name {
				fmt.Println("[ERROR] El grupo ya existe.")
				return
			}
			idActual, _ := strconv.Atoi(datos[0])
			if idActual > correlativoMaximo {
				correlativoMaximo = idActual
			}
		}
	}

	contenido += fmt.Sprintf("%d,G,%s\n", correlativoMaximo+1, name)
	utils.EscribirArchivoUsers(archivo, &sb, inodoIndex, &inodoUsers, contenido)

	archivo.Seek(partStart, 0)
	binary.Write(archivo, binary.LittleEndian, &sb)

	fmt.Printf("[ÉXITO] Grupo '%s' creado exitosamente.\n", name)
}

func Rmgrp(name string) {
	if !SesionActiva || UsuarioActual != "root" {
		fmt.Println("[ERROR] Permisos insuficientes.")
		return
	}

	archivo, sb, inodoIndex, inodoUsers, _, err := utils.ObtenerContextoParticion()
	if err != nil {
		return
	}
	defer archivo.Close()

	contenido := utils.LeerArchivoUsers(archivo, sb, inodoUsers)
	lineas := strings.Split(contenido, "\n")
	nuevoContenido := ""
	encontrado := false

	for _, linea := range lineas {
		if strings.TrimSpace(linea) == "" {
			continue
		}
		datos := strings.Split(linea, ",")

		// Si es el grupo que queremos borrar, lo saltamos
		if strings.TrimSpace(datos[1]) == "G" && strings.TrimSpace(datos[2]) == name {
			encontrado = true
			continue
		}

		// SI ES UN USUARIO Y PERTENECE AL GRUPO A BORRAR, CAMBIARLO A ROOT
		if strings.TrimSpace(datos[1]) == "U" && strings.TrimSpace(datos[2]) == name {
			// Cambiamos el grupo a 'root' para evitar que se quede sin grupo
			datos[2] = "root"
			nuevoContenido += strings.Join(datos, ",") + "\n"
		} else {
			nuevoContenido += linea + "\n"
		}
	}

	if !encontrado {
		fmt.Println("[ERROR] El grupo no existe.")
		return
	}

	utils.EscribirArchivoUsers(archivo, &sb, inodoIndex, &inodoUsers, nuevoContenido)
	fmt.Printf("[ÉXITO] Grupo '%s' eliminado. Los usuarios fueron movidos a 'root'.\n", name)
}

func Mkusr(user string, pass string, grp string) {
	if !SesionActiva || UsuarioActual != "root" {
		fmt.Println("[ERROR] Permisos insuficientes o sesión no activa.")
		return
	}

	archivo, sb, inodoIndex, inodoUsers, partStart, err := utils.ObtenerContextoParticion()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer archivo.Close()

	contenido := utils.LeerArchivoUsers(archivo, sb, inodoUsers)
	lineas := strings.Split(contenido, "\n")

	correlativoMaximo := 0
	grupoExiste := false
	nuevoContenido := ""

	for _, linea := range lineas {
		if strings.TrimSpace(linea) == "" {
			continue
		}

		datos := strings.Split(linea, ",")

		tipo := strings.TrimSpace(datos[1])
		idActualStr := strings.TrimSpace(datos[0])

		if tipo == "G" && idActualStr != "0" && strings.TrimSpace(datos[2]) == grp {
			grupoExiste = true
		}

		if tipo == "U" {
			if idActualStr != "0" && strings.TrimSpace(datos[3]) == user {
				fmt.Println("[ERROR] El usuario ya existe en el sistema.")
				return
			}
			idActual, _ := strconv.Atoi(idActualStr)
			if idActual > correlativoMaximo {
				correlativoMaximo = idActual
			}
		}

		nuevoContenido += linea + "\n"
	}

	if !grupoExiste {
		fmt.Println("[ERROR] El grupo especificado no existe.")
		return
	}

	nuevoContenido += fmt.Sprintf("%d, U, %s, %s, %s\n", correlativoMaximo+1, grp, user, pass)

	utils.EscribirArchivoUsers(archivo, &sb, inodoIndex, &inodoUsers, nuevoContenido)

	archivo.Seek(partStart, 0)
	binary.Write(archivo, binary.LittleEndian, &sb)

	fmt.Printf("[ÉXITO] Usuario '%s' creado correctamente.\n", user)
}

func Rmusr(user string) {
	if !SesionActiva || UsuarioActual != "root" {
		fmt.Println("[ERROR] Permisos insuficientes o sesión no activa.")
		return
	}

	archivo, sb, inodoIndex, inodoUsers, _, err := utils.ObtenerContextoParticion()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer archivo.Close()

	contenido := utils.LeerArchivoUsers(archivo, sb, inodoUsers)
	lineas := strings.Split(contenido, "\n")
	nuevoContenido := ""
	encontrado := false

	for _, linea := range lineas {
		if strings.TrimSpace(linea) == "" {
			continue
		}
		datos := strings.Split(linea, ",")

		if len(datos) >= 4 {
			tipo := strings.TrimSpace(datos[1])
			nombreUsuarioExtraido := strings.TrimSpace(datos[3])

			if tipo == "U" && nombreUsuarioExtraido == user {
				encontrado = true
				continue
			}
		}

		nuevoContenido += linea + "\n"
	}

	if !encontrado {
		fmt.Println("[ERROR] El usuario no existe.")
		return
	}

	utils.EscribirArchivoUsers(archivo, &sb, inodoIndex, &inodoUsers, nuevoContenido)
	fmt.Printf("[ÉXITO] Usuario '%s' eliminado físicamente del sistema.\n", user)
}

func Chgrp(user string, grp string) {
	if !SesionActiva || UsuarioActual != "root" {
		fmt.Println("[ERROR] Permisos insuficientes o sesión no activa.")
		return
	}
	archivo, sb, inodoIndex, inodoUsers, _, err := utils.ObtenerContextoParticion()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer archivo.Close()

	contenido := utils.LeerArchivoUsers(archivo, sb, inodoUsers)
	lineas := strings.Split(contenido, "\n")
	nuevoContenido := ""

	grupoExiste := false
	for _, linea := range lineas {
		if strings.TrimSpace(linea) == "" {
			continue
		}
		datos := strings.Split(linea, ",")
		if len(datos) < 3 {
			continue
		}
		tipo := strings.TrimSpace(datos[1])
		idActual := strings.TrimSpace(datos[0])
		nombreGrupo := strings.TrimSpace(datos[2])

		if tipo == "G" && idActual != "0" && nombreGrupo == grp {
			grupoExiste = true
			break
		}
	}

	if !grupoExiste {
		fmt.Println("[ERROR] El grupo de destino no existe.")
		return
	}

	usuarioEncontrado := false
	for _, linea := range lineas {
		if strings.TrimSpace(linea) == "" {
			continue
		}
		datos := strings.Split(linea, ",")
		if len(datos) < 5 {
			nuevoContenido += linea + "\n"
			continue
		}
		tipo := strings.TrimSpace(datos[1])
		idActual := strings.TrimSpace(datos[0])
		nombreUsuario := strings.TrimSpace(datos[3])

		if tipo == "U" && idActual != "0" && nombreUsuario == user {
			linea = fmt.Sprintf("%s, U, %s, %s, %s",
				idActual, grp, nombreUsuario, strings.TrimSpace(datos[4]))
			usuarioEncontrado = true
		}
		nuevoContenido += linea + "\n"
	}

	if !usuarioEncontrado {
		fmt.Println("[ERROR] El usuario especificado no existe.")
		return
	}

	utils.EscribirArchivoUsers(archivo, &sb, inodoIndex, &inodoUsers, nuevoContenido)
	fmt.Printf("[ÉXITO] El usuario '%s' ha sido movido al grupo '%s'.\n", user, grp)
}
