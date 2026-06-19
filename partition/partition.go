package partition

import (
	"MIAP1/types"
	"MIAP1/utils"
	"fmt"
	"os"
	"unsafe"
)

func CreatePartition(tamanio int64, unidad, path, tipo, ajuste, nombre string) {

	newPartition := types.Partition{
		Part_status: '1',
		Part_type:   tipo[0],
		Part_fit:    ajuste[0],
		Part_s:      utils.Tamanio(tamanio, unidad),
	}
	copy(newPartition.Part_name[:], nombre)

	archivo, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Error al abrir el archivo.")
		return
	}

	defer archivo.Close()

	if archivo == nil {
		fmt.Println("Disco no existe, no es posible crear una particion sin disco")
		return
	}

	mbr := utils.ObtenerMBR(archivo)

	if tamanio <= 0 {
		fmt.Println("El tamanio de la particion no puede ser 0")
		return
	}

	espacio := mbr.Mbr_tamano - int64(unsafe.Sizeof(mbr))
	if tamanio > espacio {
		fmt.Println("El tamanio de la particion no puede ser mayor al disco")
		return
	}

	for i := 0; i < 4; i++ {
		if tipo == "P" || tipo == "E" {
			if mbr.Mbr_partitions[i].Part_start == -1 || mbr.Mbr_partitions[i].Part_start == 0 {
				fmt.Println("Particion insertada exitosamente")

				newPartition.Part_start = int64(unsafe.Sizeof(mbr))
				mbr.Mbr_partitions[0] = newPartition
				utils.EscribirEnDisco(archivo, mbr)

				fmt.Println("************************************************")
				fmt.Println("++++++++++++++ Datos Particiones ++++++++++++++")
				fmt.Println("Nombre: ", string(mbr.Mbr_partitions[0].Part_name[:]))
				fmt.Println("Estado: ", string(mbr.Mbr_partitions[0].Part_status))
				fmt.Println("Ajuste: ", string(mbr.Mbr_partitions[0].Part_fit))
				fmt.Println("Inicio: ", mbr.Mbr_partitions[0].Part_start)
				fmt.Println("Tamanio: ", mbr.Mbr_partitions[0].Part_s)
				fmt.Println("Tipo: ", string(mbr.Mbr_partitions[0].Part_type))
				fmt.Println("************************************************")
				break
			} else {
				posicionLibre := 0
				anterior := int64(0)
				for j := 1; j < 4; j++ {
					if mbr.Mbr_partitions[j].Part_start == -1 || mbr.Mbr_partitions[j].Part_start == 0 {
						anterior = int64(mbr.Mbr_partitions[j-1].Part_start)
						newPartition.Part_start = anterior + mbr.Mbr_partitions[j-1].Part_s
						posicionLibre = j
						break
					}
				}

				mbr.Mbr_partitions[posicionLibre] = newPartition
				utils.EscribirEnDisco(archivo, mbr)
				fmt.Println("************************************************")
				fmt.Println("++++++++++++++ Datos Particiones ++++++++++++++")
				fmt.Println("Nombre: ", string(mbr.Mbr_partitions[posicionLibre].Part_name[:]))
				fmt.Println("Estado: ", string(mbr.Mbr_partitions[posicionLibre].Part_status))
				fmt.Println("Ajuste: ", string(mbr.Mbr_partitions[posicionLibre].Part_fit))
				fmt.Println("Inicio: ", mbr.Mbr_partitions[posicionLibre].Part_start)
				fmt.Println("Tamanio: ", mbr.Mbr_partitions[posicionLibre].Part_s)
				fmt.Println("Tipo: ", string(mbr.Mbr_partitions[posicionLibre].Part_type))
				fmt.Println("************************************************")
				break
			}
		}
	}
}
