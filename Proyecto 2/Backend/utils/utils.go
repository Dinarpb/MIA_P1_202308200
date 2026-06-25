package utils

import (
	"MIAP1/global"
	"MIAP1/types"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strings"
	"unsafe"
)

var IdParticionActual string = ""

func Tamanio(tamanio int64, unit string) int64 {
	if strings.Compare(unit, "k") == 0 {
		return int64(tamanio * 1024)
	} else if strings.Compare(unit, "m") == 0 {
		return int64(tamanio * 1024 * 1024)
	} else if strings.Compare(unit, "b") == 0 {
		return int64(tamanio)
	}

	return int64(-1)
}

func ObtenerMBR(archivo *os.File) types.MBR {
	mbr := types.MBR{}
	content := make([]byte, int(unsafe.Sizeof(mbr)))
	archivo.Seek(0, 0)
	archivo.Read(content)
	buffer := bytes.NewBuffer(content)
	binary.Read(buffer, binary.BigEndian, &mbr)

	return mbr
}

func EscribirEnDisco(archivo *os.File, mbr types.MBR) {
	archivo.Seek(0, 0)
	buffer := bytes.NewBuffer([]byte{})
	binary.Write(buffer, binary.BigEndian, &mbr)
	archivo.Write(buffer.Bytes())
}

func BuscarInodoPorRuta(archivo *os.File, sb types.SuperBloque, ruta string) (int, types.Inodo, error) {
	var inodoActual types.Inodo
	inodoActualIndex := 0

	fmt.Printf("[DEBUG] Verificando Inodo 0 (Raíz) en bloque %d\n", inodoActual.I_block[0])

	// Limpieza absoluta
	ruta = strings.TrimSpace(ruta)
	ruta = strings.ReplaceAll(ruta, "\r", "")
	ruta = strings.ReplaceAll(ruta, "\n", "")
	ruta = strings.ReplaceAll(ruta, "\"", "")

	archivo.Seek(int64(sb.S_inode_start), 0)
	binary.Read(archivo, binary.LittleEndian, &inodoActual)

	if ruta == "/" || ruta == "" {
		return inodoActualIndex, inodoActual, nil
	}

	pasos := strings.Split(ruta, "/")
	var pasosLimpios []string
	for _, p := range pasos {
		if p != "" {
			pasosLimpios = append(pasosLimpios, p)
		}
	}

	for _, paso := range pasosLimpios {
		encontrado := false
		//fmt.Printf("🔍 Buscando: [%s] dentro del Inodo %d\n", paso, inodoActualIndex)

		for i := 0; i < 12; i++ {
			if inodoActual.I_block[i] != -1 {
				var bc types.BloqueCarpeta
				// CORRECCIÓN MATEMÁTICA: Convertir a int64 ANTES de multiplicar
				offset := int64(sb.S_block_start) + (int64(inodoActual.I_block[i]) * int64(sb.S_block_s))
				archivo.Seek(offset, 0)
				binary.Read(archivo, binary.LittleEndian, &bc)

				for _, content := range bc.B_content {
					if content.B_inodo != -1 {
						nombreItem := strings.TrimRight(string(content.B_name[:]), "\x00")
						nombreItem = strings.TrimSpace(nombreItem)
						nombreItem = strings.ReplaceAll(nombreItem, "\r", "")
						nombreItem = strings.ReplaceAll(nombreItem, "\n", "")
						nombreItem = strings.ReplaceAll(nombreItem, "\"", "")

						//fmt.Printf("   -> Leído en disco: [%s] (Apunta al Inodo %d)\n", nombreItem, content.B_inodo)

						if nombreItem == paso {
							inodoActualIndex = int(content.B_inodo)
							// CORRECCIÓN MATEMÁTICA
							offsetInodo := int64(sb.S_inode_start) + (int64(inodoActualIndex) * int64(sb.S_inode_s))
							archivo.Seek(offsetInodo, 0)
							binary.Read(archivo, binary.LittleEndian, &inodoActual)
							encontrado = true
							break
						}
					}
				}
			}
			if encontrado {
				break
			}
		}

		if !encontrado {
			fmt.Printf("❌ ERROR FATAL: No existe [%s] dentro del Inodo %d\n\n", paso, inodoActualIndex)
			return -1, types.Inodo{}, errors.New("ruta no encontrada: " + paso)
		}
	}

	return inodoActualIndex, inodoActual, nil
}

func LeerArchivoUsers(archivo *os.File, sb types.SuperBloque, inodo types.Inodo) string {
	var contenidoCompleto strings.Builder

	for i := 0; i < 12; i++ {
		if inodo.I_block[i] == -1 {
			break // los bloques se asignan en orden; el primero libre marca el final
		}

		posicion := int64(sb.S_block_start) + (int64(inodo.I_block[i]) * int64(sb.S_block_s))

		buffer := make([]byte, 64)
		archivo.Seek(posicion, 0)
		archivo.Read(buffer)
		contenidoCompleto.Write(buffer)
	}

	contenidoFinal := bytes.Trim([]byte(contenidoCompleto.String()), "\x00")
	return string(contenidoFinal)
}

func EscribirArchivoUsers(archivo *os.File, sb *types.SuperBloque, inodoIndex int32, inodo *types.Inodo, contenidoNuevo string, partStart int64) {
	inodo.I_size = int64(len(contenidoNuevo))

	for i := 0; i < 12; i++ {
		pedazo := ""
		if len(contenidoNuevo) > 0 {
			if len(contenidoNuevo) > 64 {
				pedazo = contenidoNuevo[:64]
				contenidoNuevo = contenidoNuevo[64:]
			} else {
				pedazo = contenidoNuevo
				contenidoNuevo = ""
			}
		}

		if pedazo == "" && inodo.I_block[i] != -1 {
			var ba types.BloqueArchivo
			archivo.Seek(int64(sb.S_block_start)+int64(inodo.I_block[i])*int64(sb.S_block_s), 0)
			binary.Write(archivo, binary.LittleEndian, &ba)
			continue
		}

		if pedazo == "" && inodo.I_block[i] == -1 {
			break
		}

		if inodo.I_block[i] == -1 {
			nuevoBloque := BuscarBloqueLibre(archivo, sb)
			if nuevoBloque == -1 {
				fmt.Println("[ERROR] Disco lleno.")
				return
			}
			inodo.I_block[i] = nuevoBloque
		}

		var ba types.BloqueArchivo
		copy(ba.B_content[:], pedazo)
		archivo.Seek(int64(sb.S_block_start)+int64(inodo.I_block[i])*int64(sb.S_block_s), 0)
		binary.Write(archivo, binary.LittleEndian, &ba)
	}

	archivo.Seek(int64(sb.S_inode_start)+int64(inodoIndex)*int64(sb.S_inode_s), 0)
	binary.Write(archivo, binary.LittleEndian, inodo)

	// Persistir el superbloque en su posición REAL (inicio de la partición)
	archivo.Seek(partStart, 0)
	binary.Write(archivo, binary.LittleEndian, sb)
}

func ObtenerContextoParticion() (*os.File, types.SuperBloque, int32, types.Inodo, int64, error) {
	var rutaDisco string
	var partStart int64 = -1

	fmt.Printf("\n--- [DEBUG LOGIN] Intentando acceder a partición con ID: '%s' ---\n", IdParticionActual)

	if len(global.DiscosMontados) == 0 {
		fmt.Println("[DEBUG] ERROR: La lista global.DiscosMontados está vacía. ¡No hay particiones montadas!")
	}

	for _, disco := range global.DiscosMontados {
		for _, particion := range disco.Particiones {
			fmt.Printf("[DEBUG] Revisando partición montada -> ID: '%s', Nombre: '%s'\n", particion.ID, particion.Nombre)

			if particion.ID == IdParticionActual {
				rutaDisco = disco.Path
				fmt.Printf("[DEBUG] ¡Coincidencia de ID! Ruta del disco: %s\n", rutaDisco)

				archivoTemp, errTemp := os.OpenFile(rutaDisco, os.O_RDONLY, 0644)
				if errTemp == nil {
					mbr := ObtenerMBR(archivoTemp)
					for i := 0; i < 4; i++ {
						nombrePart := strings.TrimRight(string(mbr.Mbr_partitions[i].Part_name[:]), "\x00")
						if mbr.Mbr_partitions[i].Part_status == '1' && nombrePart == particion.Nombre {
							partStart = mbr.Mbr_partitions[i].Part_start
							fmt.Printf("[DEBUG] Partición encontrada en el MBR físico en el byte: %d\n", partStart)
							break
						}
					}
					archivoTemp.Close()
				} else {
					fmt.Printf("[DEBUG] ERROR: No se pudo abrir el disco físico en la ruta: %s\n", rutaDisco)
				}
				break
			}
		}
		if partStart != -1 {
			break
		}
	}

	if rutaDisco == "" {
		fmt.Println("[DEBUG] ERROR FINAL: No se encontró el ID en memoria.")
		return nil, types.SuperBloque{}, -1, types.Inodo{}, -1, fmt.Errorf("no se encontró partición montada con el ID '%s'", IdParticionActual)
	}

	if partStart == -1 {
		fmt.Println("[DEBUG] ERROR FINAL: La partición está en memoria pero no en el MBR (¿fue borrada con FDISK?).")
		return nil, types.SuperBloque{}, -1, types.Inodo{}, -1, fmt.Errorf("partición no encontrada en el MBR")
	}

	archivo, err := os.OpenFile(rutaDisco, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("[DEBUG] ERROR FINAL: No se pudo abrir el disco para lectura/escritura.")
		return nil, types.SuperBloque{}, -1, types.Inodo{}, -1, fmt.Errorf("error abriendo disco")
	}

	var sb types.SuperBloque
	archivo.Seek(partStart, 0)
	binary.Read(archivo, binary.LittleEndian, &sb)

	fmt.Println("[DEBUG] Buscando archivo /users.txt...")
	inodoIndex, inodoUsers, errInodo := BuscarInodoPorRuta(archivo, sb, "/users.txt")
	if errInodo != nil {
		archivo.Close()
		fmt.Printf("[DEBUG] ERROR FINAL: %v\n", errInodo)
		fmt.Println("[DEBUG] ¿Ejecutaste el comando MKFS antes de intentar hacer login?")
		return nil, types.SuperBloque{}, -1, types.Inodo{}, -1, fmt.Errorf("error buscando users.txt")
	}

	fmt.Println("[DEBUG] ¡Contexto obtenido con éxito!")
	return archivo, sb, int32(inodoIndex), inodoUsers, partStart, nil
}

func WriteBlock(archivo *os.File, pos int64, data interface{}, blockSize int32) {
	// 1. Nos ubicamos en la posición exacta
	archivo.Seek(pos, 0)

	// 2. Creamos el buffer real (1024 bytes)
	buffer := make([]byte, blockSize)

	// 3. Escribimos la data en un buffer temporal
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, data)

	// ¡Si hay un error, esto nos avisará en la consola!
	if err != nil {
		fmt.Println("[ERROR CRÍTICO] Falló la serialización binaria:", err)
	}

	// 4. Copiamos y rellenamos el resto con ceros
	copy(buffer, buf.Bytes())

	// 5. Escribimos al disco físico
	archivo.Write(buffer)
}

func BuscarBloqueLibre(archivo *os.File, sb *types.SuperBloque) int32 {
	archivo.Seek(int64(sb.S_bm_block_start), 0)

	bitmap := make([]byte, sb.S_blocks_count)
	binary.Read(archivo, binary.LittleEndian, &bitmap)

	for i := 0; i < int(sb.S_blocks_count); i++ {
		if bitmap[i] == 0 { // ANTES: bitmap[i] == '0'
			archivo.Seek(int64(sb.S_bm_block_start)+int64(i), 0)
			binary.Write(archivo, binary.LittleEndian, byte('1'))
			sb.S_free_blocks_count--
			return int32(i)
		}
	}
	return -1
}

func BuscarInodoLibre(archivo *os.File, sb *types.SuperBloque) int32 {
	archivo.Seek(int64(sb.S_bm_inode_start), 0)
	bitmap := make([]byte, sb.S_inodes_count)
	binary.Read(archivo, binary.LittleEndian, &bitmap)

	for i := 0; i < int(sb.S_inodes_count); i++ {
		if bitmap[i] == 0 {
			archivo.Seek(int64(sb.S_bm_inode_start)+int64(i), 0)
			binary.Write(archivo, binary.LittleEndian, byte('1'))
			sb.S_free_inodes_count--
			return int32(i)
		}
	}
	return -1
}
