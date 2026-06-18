package partition

import (
	"fmt"
	"MIAP1/types"
	"MIAP1/utils"
	"os"
	"unsafe"
)

func CreatePartition(tamanio int64, unidad, path, tipo, ajuste, nombre string) {

	newPartition := types.Partition{
		Estado:  '1',
		Tipo:    tipo[0],
		Ajuste:  ajuste[0],
		Tamanio: utils.Tamanio(tamanio, unidad),
	}
	copy(newPartition.Nombre[:], nombre)

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

	espacio := mbr.Tamanio - int64(unsafe.Sizeof(mbr))
	if tamanio > espacio {
		fmt.Println("El tamanio de la particion no puede ser mayor al disco")
		return
	}

	for i := 0; i < 4; i++ {
		if tipo == "P" || tipo == "E" {
			if mbr.Particiones[i].Inicio == 0 {
				fmt.Println("Particion insertada exitosamente")

				newPartition.Inicio = int64(unsafe.Sizeof(mbr))
				mbr.Particiones[0] = newPartition
				utils.EscribirEnDisco(archivo, mbr)

				fmt.Println("************************************************")
				fmt.Println("++++++++++++++ Datos Particiones ++++++++++++++")
				fmt.Println("Nombre: ", mbr.Particiones[i].Nombre)
				fmt.Println("Estado: ", string(mbr.Particiones[i].Estado))
				fmt.Println("Ajuste: ", mbr.Particiones[i].Ajuste)
				fmt.Println("Inicio: ", mbr.Particiones[i].Inicio)
				fmt.Println("Tamanio: ", mbr.Particiones[i].Tamanio)
				fmt.Println("Tipo: ", string(mbr.Particiones[i].Tipo))
				fmt.Println("************************************************")
				break
			} else {
				posicionLibre := 0
				anterior := int64(0)
				for j := 1; j < 4; j++ {
					if mbr.Particiones[j].Inicio == 0 {
						anterior = int64(mbr.Particiones[j-1].Inicio)
						newPartition.Inicio = anterior + mbr.Particiones[j-1].Tamanio
						posicionLibre = j
						break
					}
				}

				mbr.Particiones[posicionLibre] = newPartition
				utils.EscribirEnDisco(archivo, mbr)
				fmt.Println("************************************************")
				fmt.Println("++++++++++++++ Datos Particiones ++++++++++++++")
				fmt.Println("Nombre: ", mbr.Particiones[i].Nombre)
				fmt.Println("Estado: ", string(mbr.Particiones[i].Estado))
				fmt.Println("Ajuste: ", mbr.Particiones[i].Ajuste)
				fmt.Println("Inicio: ", mbr.Particiones[i].Inicio)
				fmt.Println("Tamanio: ", mbr.Particiones[i].Tamanio)
				fmt.Println("Tipo: ", string(mbr.Particiones[i].Tipo))
				fmt.Println("************************************************")
				break
			}
		}
	}

}
