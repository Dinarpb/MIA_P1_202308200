package format

import (
	"MIAP1/mount"
	"MIAP1/types"
	"MIAP1/utils"
	"fmt"
	"os"
	"strings"
	"unsafe"
)

func CalcularEstructuras(tamanioParticion int64) int64 {
	tamanioSuperbloque := int64(unsafe.Sizeof(types.Superblock{}))
	tamanioInodo := int64(unsafe.Sizeof(types.Inodo{}))
	tamanioBloque := int64(unsafe.Sizeof(types.BloqueArchivo{}))

	numerador := tamanioParticion - tamanioSuperbloque
	denominador := 4 + tamanioInodo + (3 * tamanioBloque)

	n := numerador / denominador

	return n
}

func Mkfs(id string, tipo string) {
	var rutaDisco string
	var tamanioParticion int64
	encontrado := false

	for _, disco := range mount.DiscosMontados {
		for _, particion := range disco.Particiones {
			if particion.ID == id {
				rutaDisco = disco.Path
				archivo, err := os.OpenFile(rutaDisco, os.O_RDWR, 0644)
				if err != nil {
					fmt.Println("Error al abrir el disco.")
					return
				}
				mbr := utils.ObtenerMBR(archivo)
				for i := 0; i < 4; i++ {
					nombreParticion := string(mbr.Mbr_partitions[i].Part_name[:])
					nombreLimpio := strings.TrimRight(nombreParticion, "\x00")

					if nombreLimpio == particion.Nombre {
						tamanioParticion = mbr.Mbr_partitions[i].Part_s
						encontrado = true
						break
					}
				}
				archivo.Close()
				break
			}
		}
		if encontrado {
			break
		}
	}

	if !encontrado {
		fmt.Println("Error: No se encontró la partición montada con el ID especificado.")
		return
	}

	n := CalcularEstructuras(tamanioParticion)
	fmt.Printf("Formateo %s exitoso. Se escribirán %d inodos y %d bloques en la partición.\n", tipo, n, 3*n)
}
