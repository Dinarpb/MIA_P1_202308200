package utils

import "MIAP1/types"

func TienePermiso(uid int32, gid int32, inodo types.Inodo, operacion int) bool {
	if uid == 1 { // Usuario root siempre tiene acceso
		return true
	}

	permisos := inodo.I_perm // Ejemplo: [6, 6, 4] en formato octal
	var categoria int

	if uid == inodo.I_uid {
		categoria = 0 // User
	} else if gid == inodo.I_gid {
		categoria = 1 // Group
	} else {
		categoria = 2 // Other
	}

	// operacion: 4=lectura, 2=escritura, 1=ejecución
	return (int(permisos[categoria]) & operacion) != 0
}
