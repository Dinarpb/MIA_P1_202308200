package filesystem

import (
	"MIAP1/global"
	"MIAP1/types"
	"MIAP1/users"
	"MIAP1/utils"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
)

func Cat(rutaArchivo string) {
	if !users.SesionActiva {
		fmt.Println("[ERROR] No hay sesión activa. Inicie sesión para usar CAT.")
		return
	}

	var rutaDisco string
	var partStart int64 = -1
	for _, disco := range global.DiscosMontados {
		for _, particion := range disco.Particiones {
			if particion.ID == utils.IdParticionActual {
				rutaDisco = disco.Path
				archivoTemp, errTemp := os.OpenFile(rutaDisco, os.O_RDONLY, 0644)
				if errTemp == nil {
					mbr := utils.ObtenerMBR(archivoTemp)
					for i := 0; i < 4; i++ {
						nombrePart := strings.TrimRight(string(mbr.Mbr_partitions[i].Part_name[:]), "\x00")
						if mbr.Mbr_partitions[i].Part_status == '1' && nombrePart == particion.Nombre {
							partStart = mbr.Mbr_partitions[i].Part_start
							break
						}
					}
					archivoTemp.Close()
				}
				break
			}
		}
		if partStart != -1 {
			break
		}
	}
	if partStart == -1 {
		fmt.Println("[ERROR] No se pudo localizar la partición de la sesión activa.")
		return
	}

	// leemos el SuperBloque
	archivo, err := os.OpenFile(rutaDisco, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("[ERROR] No se pudo abrir el disco para CAT.")
		return
	}
	defer archivo.Close()

	var sb types.SuperBloque
	archivo.Seek(partStart, 0)
	binary.Read(archivo, binary.LittleEndian, &sb)

	_, inodoArchivo, errInodo := utils.BuscarInodoPorRuta(archivo, sb, rutaArchivo)
	if errInodo != nil {
		fmt.Printf("[ERROR] No se encontró el archivo '%s'.\n", rutaArchivo)
		return
	}

	if inodoArchivo.I_type != '1' {
		fmt.Printf("[ERROR] La ruta '%s' no es un archivo de texto.\n", rutaArchivo)
		return
	}

	contenido := utils.LeerArchivoUsers(archivo, sb, inodoArchivo)

	fmt.Printf("========== CONTENIDO DE %s ==========\n", rutaArchivo)
	fmt.Print(contenido)
	fmt.Println("==================================================")
}
