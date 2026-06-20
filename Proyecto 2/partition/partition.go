package partition

import (
	"MIAP1/types"
	"MIAP1/utils"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
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
		fmt.Println("[ERROR] Error al abrir el archivo.")
		return
	}

	defer archivo.Close()

	if archivo == nil {
		fmt.Println("[ERROR] Disco no existe, no es posible crear una particion sin disco")
		return
	}

	mbr := utils.ObtenerMBR(archivo)

	if tamanio <= 0 {
		fmt.Println("[ERROR] El tamanio de la particion no puede ser 0")
		return
	}

	extendidas := 0
	posicionLibre := -1
	var inicioNuevo int64 = int64(unsafe.Sizeof(mbr))

	for i := 0; i < 4; i++ {
		if mbr.Mbr_partitions[i].Part_status == '1' {
			nombreActual := strings.TrimRight(string(mbr.Mbr_partitions[i].Part_name[:]), "\x00")
			if nombreActual == nombre {
				fmt.Println("[ERROR] Ya existe una partición con ese nombre en este disco.")
				return
			}
		}

		if mbr.Mbr_partitions[i].Part_status == '1' && (mbr.Mbr_partitions[i].Part_type == 'e' || mbr.Mbr_partitions[i].Part_type == 'E') {
			extendidas++
		}

		if (mbr.Mbr_partitions[i].Part_start == -1 || mbr.Mbr_partitions[i].Part_start == 0) && posicionLibre == -1 {
			posicionLibre = i
		}

		if mbr.Mbr_partitions[i].Part_status == '1' {
			finParticion := mbr.Mbr_partitions[i].Part_start + mbr.Mbr_partitions[i].Part_s
			if finParticion > inicioNuevo {
				inicioNuevo = finParticion
			}
		}
	}

	if tipo == "l" || tipo == "L" {
		if extendidas == 0 {
			fmt.Println("[ERROR] No hay partición extendida para crear lógica")
			return
		}

		insertarLogica(archivo, mbr, tamanio, unidad, ajuste, nombre)
		fmt.Printf("[ÉXITO] Partición LÓGICA creada exitosamente: %s\n", nombre)
		return
	}

	if tipo == "e" || tipo == "E" {
		if extendidas > 0 {
			fmt.Println("[ERROR] Ya existe una partición extendida en este disco")
			return
		}
	}

	if (tipo == "p" || tipo == "P" || tipo == "e" || tipo == "E") && posicionLibre == -1 {
		fmt.Println("[ERROR] No hay slots libres (máximo 4)")
		return
	}

	if tipo == "p" || tipo == "P" || tipo == "e" || tipo == "E" {
		if inicioNuevo+newPartition.Part_s > mbr.Mbr_tamano {
			fmt.Println("[ERROR] No hay espacio suficiente en el disco")
			return
		}

		newPartition.Part_start = inicioNuevo
		mbr.Mbr_partitions[posicionLibre] = newPartition
		utils.EscribirEnDisco(archivo, mbr)

		fmt.Printf("[ÉXITO] Partición creada exitosamente: %s\n", nombre)
		fmt.Println("=============== Datos Particiones ===============")
		fmt.Println("Nombre: ", string(mbr.Mbr_partitions[posicionLibre].Part_name[:]))
		fmt.Println("Estado: ", string(mbr.Mbr_partitions[posicionLibre].Part_status))
		fmt.Println("Ajuste: ", string(mbr.Mbr_partitions[posicionLibre].Part_fit))
		fmt.Println("Inicio: ", mbr.Mbr_partitions[posicionLibre].Part_start)
		fmt.Println("Tamanio: ", mbr.Mbr_partitions[posicionLibre].Part_s)
		fmt.Println("Tipo: ", string(mbr.Mbr_partitions[posicionLibre].Part_type))
		fmt.Println("=================================================")
	}
}

func insertarLogica(archivo *os.File, mbr types.MBR, tamanio int64, unidad string, ajuste string, nombre string) {
	var extPart types.Partition
	var extStart int64 = -1
	var extSize int64 = -1

	// Encontrar la partición Extendida
	for i := 0; i < 4; i++ {
		if mbr.Mbr_partitions[i].Part_status == '1' && (mbr.Mbr_partitions[i].Part_type == 'e' || mbr.Mbr_partitions[i].Part_type == 'E') {
			extPart = mbr.Mbr_partitions[i]
			extStart = extPart.Part_start
			extSize = extPart.Part_s
			break
		}
	}

	if extStart == -1 {
		fmt.Println("[ERROR] No hay partición extendida para crear lógica")
		return
	}

	tamanoBytes := utils.Tamanio(tamanio, unidad)
	var ebrActual types.EBR
	posicionActual := extStart

	// Recorrer la lista enlazada de EBR
	for {
		archivo.Seek(posicionActual, 0)
		binary.Read(archivo, binary.LittleEndian, &ebrActual)

		if ebrActual.Part_s == 0 || ebrActual.Part_mount == '0' {
			// Validar que quepa en la extendida
			if posicionActual+tamanoBytes > extStart+extSize {
				fmt.Println("[ERROR] La partición lógica excede el límite de la partición extendida")
				return
			}

			// Llenar datos
			ebrActual.Part_mount = '1'
			ebrActual.Part_fit = ajuste[0]
			ebrActual.Part_start = posicionActual
			ebrActual.Part_s = tamanoBytes
			ebrActual.Part_next = -1
			copy(ebrActual.Part_name[:], nombre)

			// Escribir en disco
			archivo.Seek(posicionActual, 0)
			binary.Write(archivo, binary.LittleEndian, &ebrActual)

			fmt.Printf("[ÉXITO] Partición LÓGICA creada exitosamente: %s\n", nombre)
			return
		}

		if ebrActual.Part_next != -1 {
			// Saltamos al siguiente EBR
			posicionActual = ebrActual.Part_next
		} else {
			nuevoInicio := ebrActual.Part_start + ebrActual.Part_s

			// Validar espacio
			if nuevoInicio+tamanoBytes > extStart+extSize {
				fmt.Println("[ERROR] No hay espacio en la partición extendida para otra lógica")
				return
			}

			ebrActual.Part_next = nuevoInicio
			archivo.Seek(posicionActual, 0)
			binary.Write(archivo, binary.LittleEndian, &ebrActual)

			nuevoEBR := types.EBR{
				Part_mount: '1',
				Part_fit:   ajuste[0],
				Part_start: nuevoInicio,
				Part_s:     tamanoBytes,
				Part_next:  -1,
			}
			copy(nuevoEBR.Part_name[:], nombre)

			archivo.Seek(nuevoInicio, 0)
			binary.Write(archivo, binary.LittleEndian, &nuevoEBR)

			fmt.Printf("[ÉXITO] Partición LÓGICA creada exitosamente: %s\n", nombre)
			return
		}
	}
}
