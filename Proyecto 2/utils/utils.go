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

	ruta = strings.TrimSpace(ruta)
	ruta = strings.ReplaceAll(ruta, "\r", "")
	ruta = strings.ReplaceAll(ruta, "\n", "")

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

		for i := 0; i < 12; i++ {
			if inodoActual.I_block[i] != -1 {
				var bc types.BloqueCarpeta
				archivo.Seek(int64(sb.S_block_start)+(int64(inodoActual.I_block[i])*int64(sb.S_block_s)), 0)
				binary.Read(archivo, binary.LittleEndian, &bc)

				for _, content := range bc.B_content {
					if content.B_inodo != -1 {
						nombreItem := strings.TrimRight(string(content.B_name[:]), "\x00")
						nombreItem = strings.TrimSpace(nombreItem)
						nombreItem = strings.ReplaceAll(nombreItem, "\r", "")
						nombreItem = strings.ReplaceAll(nombreItem, "\n", "")
						nombreItem = strings.ReplaceAll(nombreItem, "\"", "")

						if nombreItem == paso {
							inodoActualIndex = int(content.B_inodo)
							archivo.Seek(int64(sb.S_inode_start)+int64(int32(inodoActualIndex)*sb.S_inode_s), 0)
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
			return -1, types.Inodo{}, errors.New("ruta no encontrada: " + paso)
		}
	}

	return inodoActualIndex, inodoActual, nil
}

func LeerArchivoUsers(archivo *os.File, sb types.SuperBloque, inodo types.Inodo) string {
	contenidoTotal := ""
	for i := 0; i < 12; i++ {
		if inodo.I_block[i] != -1 {
			var ba types.BloqueArchivo
			archivo.Seek(int64(sb.S_block_start)+int64(inodo.I_block[i])*int64(sb.S_block_s), 0)
			binary.Read(archivo, binary.LittleEndian, &ba)
			contenidoTotal += string(ba.B_content[:])
		}
	}

	if int64(len(contenidoTotal)) > inodo.I_size {
		contenidoTotal = contenidoTotal[:inodo.I_size]
	}

	return strings.TrimRight(contenidoTotal, "\x00")
}

func EscribirArchivoUsers(archivo *os.File, sb *types.SuperBloque, inodoIndex int32, inodo *types.Inodo, contenidoNuevo string) {
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
			inodo.I_block[i] = sb.S_first_blo
			archivo.Seek(int64(sb.S_bm_block_start)+int64(sb.S_first_blo), 0)
			binary.Write(archivo, binary.LittleEndian, byte('1'))
			sb.S_first_blo++
			sb.S_free_blocks_count--
		}

		var ba types.BloqueArchivo
		copy(ba.B_content[:], pedazo)
		archivo.Seek(int64(sb.S_block_start)+int64(inodo.I_block[i])*int64(sb.S_block_s), 0)
		binary.Write(archivo, binary.LittleEndian, &ba)
	}

	archivo.Seek(int64(sb.S_inode_start)+int64(inodoIndex)*int64(sb.S_inode_s), 0)
	binary.Write(archivo, binary.LittleEndian, inodo)
}

func ObtenerContextoParticion() (*os.File, types.SuperBloque, int32, types.Inodo, int64, error) {
	var rutaDisco string
	var partStart int64 = -1

	for _, disco := range global.DiscosMontados {
		for _, particion := range disco.Particiones {
			if particion.ID == IdParticionActual {
				rutaDisco = disco.Path

				archivoTemp, errTemp := os.OpenFile(rutaDisco, os.O_RDONLY, 0644)
				if errTemp == nil {
					mbr := ObtenerMBR(archivoTemp)
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

	archivo, err := os.OpenFile(rutaDisco, os.O_RDWR, 0644)
	if err != nil {
		return nil, types.SuperBloque{}, -1, types.Inodo{}, -1, fmt.Errorf("error abriendo disco")
	}

	var sb types.SuperBloque
	archivo.Seek(partStart, 0)
	binary.Read(archivo, binary.LittleEndian, &sb)

	inodoIndex, inodoUsers, errInodo := BuscarInodoPorRuta(archivo, sb, "/users.txt")
	if errInodo != nil {
		archivo.Close()
		return nil, types.SuperBloque{}, -1, types.Inodo{}, -1, fmt.Errorf("error buscando users.txt")
	}

	return archivo, sb, int32(inodoIndex), inodoUsers, partStart, nil
}
