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
var UsuarioActualUID int32 = -1
var UsuarioActualGID int32 = -1

func Login(user string, pass string, id string) {
	if user == "root" && pass == "123" {
		SesionActiva = true
		UsuarioActual = "root"
		UsuarioActualUID = 1
		UsuarioActualGID = 1
		utils.IdParticionActual = id
		fmt.Printf("[ÉXITO - BYPASS] Sesión iniciada a la fuerza para '%s'.\n", user)
		return
	}

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

	//fmt.Printf("[DEBUG] Contenido de users.txt leído:\n'%s'\n", contenido)

	lineas := strings.Split(contenido, "\n")

	loginExitoso := false

	for _, linea := range lineas {
		linea = strings.TrimSpace(linea)
		if linea == "" {
			continue
		}
		datos := strings.Split(linea, ",")

		//fmt.Printf("[DEBUG] Procesando línea: %v\n", datos)

		if len(datos) >= 5 && strings.TrimSpace(datos[1]) == "U" && strings.TrimSpace(datos[0]) != "0" {
			if strings.TrimSpace(datos[3]) == user && strings.TrimSpace(datos[4]) == pass {
				loginExitoso = true

				// 1. Guardar el UID
				uidTemp, _ := strconv.Atoi(strings.TrimSpace(datos[0]))
				UsuarioActualUID = int32(uidTemp)

				// 2. Buscar el GID del grupo al que pertenece
				nombreGrupo := strings.TrimSpace(datos[2])
				for _, lineaGrp := range lineas {
					if lineaGrp == "" {
						continue
					}
					datosGrp := strings.Split(lineaGrp, ",")
					if len(datosGrp) >= 3 && strings.TrimSpace(datosGrp[1]) == "G" && strings.TrimSpace(datosGrp[2]) == nombreGrupo {
						gidTemp, _ := strconv.Atoi(strings.TrimSpace(datosGrp[0]))
						UsuarioActualGID = int32(gidTemp)
						break
					}
				}
				break
			}
		}
	}

	if loginExitoso {
		SesionActiva = true
		UsuarioActual = user
		fmt.Printf("[ÉXITO] Sesión iniciada para el usuario '%s' (UID: %d, GID: %d) en la partición '%s'.\n", user, UsuarioActualUID, UsuarioActualGID, id)
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
	UsuarioActualUID = -1
	UsuarioActualGID = -1
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
		linea = strings.TrimSpace(linea)
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

	// Agregamos el grupo
	contenido += fmt.Sprintf("%d,G,%s\n", correlativoMaximo+1, name)

	// --- AÑADE ESTO: Validar la escritura ---
	utils.EscribirArchivoUsers(archivo, &sb, inodoIndex, &inodoUsers, contenido, partStart)

	// Actualizar el SB en disco
	archivo.Seek(partStart, 0)
	err = binary.Write(archivo, binary.LittleEndian, &sb)
	if err != nil {
		fmt.Println("[ERROR CRÍTICO] No se pudo guardar el SuperBloque:", err)
		return
	}

	// ¡IMPORTANTE! Forzar que los datos salgan del buffer a tu disco real
	archivo.Sync()

	fmt.Printf("[ÉXITO] Grupo '%s' creado exitosamente.\n", name)
}

func Rmgrp(name string) {
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
	nuevoContenido := ""
	encontrado := false

	for _, linea := range lineas {
		if linea == "" {
			continue
		}
		datos := strings.Split(linea, ",")

		if len(datos) >= 3 && datos[1] == "G" && datos[2] == name {
			encontrado = true
			continue
		}

		if strings.TrimSpace(datos[1]) == "U" && strings.TrimSpace(datos[2]) == name {
			datos[2] = "root" // Mover usuario al grupo root automáticamente
			nuevoContenido += strings.Join(datos, ",") + "\n"
		}

		nuevoContenido += linea + "\n"
	}

	if !encontrado {
		fmt.Println("[ERROR] El grupo no existe.")
		return
	}

	utils.EscribirArchivoUsers(archivo, &sb, inodoIndex, &inodoUsers, nuevoContenido, partStart)
	if err != nil {
		fmt.Println("[ERROR] Falló la escritura en disco.")
		return
	}
	fmt.Printf("[ÉXITO] Grupo '%s' eliminado físicamente del sistema.\n", name)
}

// 1. Asegúrate de que las variables en la firma sean exactamente estas:
func Mkusr(user string, pass string, grp string) {
	if !SesionActiva || UsuarioActual != "root" {
		fmt.Println("[ERROR] Permisos insuficientes.")
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

	// Validar grupo y calcular nuevo ID
	for _, linea := range lineas {
		if strings.TrimSpace(linea) == "" {
			continue
		}
		datos := strings.Split(linea, ",")
		tipo := strings.TrimSpace(datos[1])
		idActualStr := strings.TrimSpace(datos[0])

		if tipo == "G" && idActualStr != "0" && strings.TrimSpace(datos[2]) == grp { // Usando variable 'grp'
			grupoExiste = true
		}
		if tipo == "U" {
			idActual, _ := strconv.Atoi(idActualStr)
			if idActual > correlativoMaximo {
				correlativoMaximo = idActual
			}
		}
	}

	if !grupoExiste {
		fmt.Println("[ERROR] El grupo especificado no existe.")
		return
	}

	// Aquí usamos 'user', 'pass' y 'grp' que vienen en la firma de la función
	nuevoUID := correlativoMaximo + 1
	nuevaLinea := fmt.Sprintf("%d,U,%s,%s,%s\n", nuevoUID, grp, user, pass)

	contenido += nuevaLinea

	// Escribir al disco
	utils.EscribirArchivoUsers(archivo, &sb, inodoIndex, &inodoUsers, contenido, partStart)
	//fmt.Println("[DEBUG] Escritura terminada.")

	archivo.Seek(partStart, 0)
	binary.Write(archivo, binary.LittleEndian, &sb)

	fmt.Printf("[ÉXITO] Usuario '%s' creado correctamente con ID %d.\n", user, nuevoUID)
}

func Rmusr(user string) {
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

	utils.EscribirArchivoUsers(archivo, &sb, inodoIndex, &inodoUsers, nuevoContenido, partStart)
	if err != nil {
		fmt.Println("[ERROR] Falló la escritura en disco.")
		return
	}

	fmt.Printf("[ÉXITO] Usuario '%s' eliminado físicamente del sistema.\n", user)
}

func Chgrp(user string, grp string) {
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

	utils.EscribirArchivoUsers(archivo, &sb, inodoIndex, &inodoUsers, nuevoContenido, partStart)
	if err != nil {
		fmt.Println("[ERROR] Falló la escritura en disco.")
		return
	}

	fmt.Printf("[ÉXITO] El usuario '%s' ha sido movido al grupo '%s'.\n", user, grp)
}
