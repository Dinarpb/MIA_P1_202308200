package utils

import "MIAP1/types"

func TienePermiso(uid int32, gid int32, inodo types.Inodo, operacion int) bool {
	if uid == 1 {
		return true
	}

	permisos := inodo.I_perm
	var categoria int

	if uid == inodo.I_uid {
		categoria = 0
	} else if gid == inodo.I_gid {
		categoria = 1
	} else {
		categoria = 2
	}

	return (int(permisos[categoria]) & operacion) != 0
}



