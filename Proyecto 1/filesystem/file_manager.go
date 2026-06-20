package filesystem

import (
	"MIAP1/types"
	"MIAP1/users"
	"MIAP1/utils"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func Mkfile(ruta string, r bool, size int) {
	if size < 0 {
		fmt.Println("[ERROR] El tamaño no puede ser negativo.")
		return
	}

	archivo, sb, _, _, partStart, err := utils.ObtenerContextoParticion()
	if err != nil {
		fmt.Println("[ERROR] No se pudo acceder a la partición activa.")
		return
	}
	defer archivo.Close()

	directorioPadre := filepath.Dir(ruta)
	nombreArchivo := filepath.Base(ruta)

	padreIndex, inodoPadre, errPadre := utils.BuscarInodoPorRuta(archivo, sb, directorioPadre)
	if errPadre != nil {
		if !r {
			fmt.Printf("[ERROR] La ruta '%s' no existe.\n", directorioPadre)
			return
		}

		fmt.Printf("[INFO] Creando directorios intermedios para: %s\n", directorioPadre)

		padreIndex, inodoPadre, errPadre = utils.BuscarInodoPorRuta(archivo, sb, directorioPadre)
		if errPadre != nil {
			fmt.Println("[ERROR] Falló la creación de las carpetas padre.")
			return
		}
	}

	_, _, errArchivo := utils.BuscarInodoPorRuta(archivo, sb, ruta)
	if errArchivo == nil {
		fmt.Printf("[ERROR] El archivo '%s' ya existe.\n", ruta)
		return
	}

	nuevoInodoIndex := sb.S_first_ino
	sb.S_first_ino++
	sb.S_free_inodes_count--

	archivo.Seek(int64(sb.S_bm_inode_start)+int64(nuevoInodoIndex), 0)
	binary.Write(archivo, binary.LittleEndian, byte('1'))

	inodoArchivo := types.Inodo{
		I_uid:  1,
		I_gid:  1,
		I_size: int64(size),
		I_type: '1',
		I_perm: [3]byte{'6', '6', '4'},
	}
	for i := 0; i < 15; i++ {
		inodoArchivo.I_block[i] = -1
	}

	contenidoNuevo := generarContenidoMkfile(size)

	for i := 0; i < 12; i++ {
		if len(contenidoNuevo) == 0 {
			break
		}
		pedazo := contenidoNuevo
		if len(contenidoNuevo) > 64 {
			pedazo = contenidoNuevo[:64]
			contenidoNuevo = contenidoNuevo[64:]
		} else {
			contenidoNuevo = ""
		}

		inodoArchivo.I_block[i] = sb.S_first_blo
		archivo.Seek(int64(sb.S_bm_block_start)+int64(sb.S_first_blo), 0)
		binary.Write(archivo, binary.LittleEndian, byte('1'))
		sb.S_first_blo++
		sb.S_free_blocks_count--

		var ba types.BloqueArchivo
		copy(ba.B_content[:], pedazo)
		archivo.Seek(int64(sb.S_block_start)+int64(inodoArchivo.I_block[i])*int64(sb.S_block_s), 0)
		binary.Write(archivo, binary.LittleEndian, &ba)
	}

	archivo.Seek(int64(sb.S_inode_start)+int64(nuevoInodoIndex)*int64(sb.S_inode_s), 0)
	binary.Write(archivo, binary.LittleEndian, &inodoArchivo)

	//Enlazar este nuevo archivo a la carpeta padre
	exitoEnlace := EnlazarInodoEnCarpeta(archivo, &sb, inodoPadre, int32(padreIndex), nuevoInodoIndex, nombreArchivo)
	if !exitoEnlace {
		fmt.Println("[ERROR] No hay espacio en la carpeta padre para este archivo.")
		return
	}

	//Actualizar el SuperBloque general
	archivo.Seek(partStart, 0)
	binary.Write(archivo, binary.LittleEndian, &sb)

	fmt.Printf("[ÉXITO] Archivo creado en %s con tamaño %d bytes.\n", ruta, size)
}

func Mkdir(ruta string, p bool) {
	ruta = strings.TrimSpace(ruta)
	ruta = strings.ReplaceAll(ruta, "\r", "")
	ruta = strings.ReplaceAll(ruta, "\n", "")
	ruta = strings.ReplaceAll(ruta, "\"", "")
	ruta = strings.TrimSuffix(ruta, "/")

	if !users.SesionActiva {
		fmt.Println("[ERROR] No hay sesión activa.")
		return
	}

	archivo, sb, _, _, partStart, err := utils.ObtenerContextoParticion()
	if err != nil {
		fmt.Println("[ERROR] No se pudo acceder a la partición.")
		return
	}
	defer archivo.Close()

	ruta = strings.TrimSuffix(ruta, "/")

	//si viene sin pe, creamos la ruta recursiva
	if p {
		crearRutaRecursiva(archivo, &sb, ruta, partStart)
		return
	}

	// Sin -p, verificamos si la carpeta ya existe
	_, _, errExiste := utils.BuscarInodoPorRuta(archivo, sb, ruta)
	if errExiste == nil {
		fmt.Printf("[ERROR] El directorio '%s' ya existe.\n", ruta)
		return
	}

	// Extraemos el padre directo y el nombre de la nueva carpeta
	directorioPadre := filepath.Dir(ruta)
	nombreNuevaCarpeta := filepath.Base(ruta)

	var padreIndex int32
	var inodoPadre types.Inodo

	if directorioPadre == "/" || directorioPadre == "" {
		padreIndex = 0
		archivo.Seek(int64(sb.S_inode_start), 0)
		binary.Read(archivo, binary.LittleEndian, &inodoPadre)
	} else {
		pIdx, inodP, errPadre := utils.BuscarInodoPorRuta(archivo, sb, directorioPadre)
		if errPadre != nil {
			fmt.Printf("[ERROR] La ruta padre '%s' no existe.\n", directorioPadre)
			return
		}
		padreIndex = int32(pIdx)
		inodoPadre = inodP
	}

	// Creamos solo la carpeta final
	exito := crearDirectorioFisico(archivo, &sb, padreIndex, inodoPadre, nombreNuevaCarpeta, partStart)
	if exito {
		fmt.Printf("[ÉXITO] Directorio '%s' creado.\n", ruta)
	}
}

func generarContenidoMkfile(size int) string {
	if size <= 0 {
		return ""
	}
	texto := ""
	for i := 0; i < size; i++ {
		texto += strconv.Itoa(i % 10) // Concatena 0, 1, 2... 9 y repite
	}
	return texto
}

func EnlazarInodoEnCarpeta(archivo *os.File, sb *types.SuperBloque, inodoPadre types.Inodo, indexPadre int32, nuevoInodo int32, nombreArchivo string) bool {
	// Buscamos espacio en los bloques que la carpeta ya tiene asignados
	for i := 0; i < 12; i++ {
		if inodoPadre.I_block[i] != -1 {
			var bc types.BloqueCarpeta
			archivo.Seek(int64(sb.S_block_start)+int64(inodoPadre.I_block[i])*int64(sb.S_block_s), 0)
			binary.Read(archivo, binary.LittleEndian, &bc)

			// Buscamos una celda vacía
			for j := 0; j < 4; j++ {
				if bc.B_content[j].B_inodo == -1 {
					bc.B_content[j].B_inodo = nuevoInodo
					copy(bc.B_content[j].B_name[:], nombreArchivo)

					// Reescribimos el bloque actualizado
					archivo.Seek(int64(sb.S_block_start)+int64(inodoPadre.I_block[i])*int64(sb.S_block_s), 0)
					binary.Write(archivo, binary.LittleEndian, &bc)
					return true
				}
			}
		}
	}

	// Si todos estan llenos, creamos uno
	for i := 0; i < 12; i++ {
		if inodoPadre.I_block[i] == -1 {

			// reservamos el nuevo bloque en bimap
			nuevoBloqueIndex := sb.S_first_blo
			sb.S_first_blo++
			sb.S_free_blocks_count--
			archivo.Seek(int64(sb.S_bm_block_start)+int64(nuevoBloqueIndex), 0)
			binary.Write(archivo, binary.LittleEndian, byte('1'))

			// configuramos el nuevo bloque en bimap
			var bc types.BloqueCarpeta
			for j := 0; j < 4; j++ {
				bc.B_content[j].B_inodo = -1
			}

			bc.B_content[0].B_inodo = nuevoInodo
			copy(bc.B_content[0].B_name[:], nombreArchivo)

			// Escribimos el bloque al disco
			archivo.Seek(int64(sb.S_block_start)+int64(nuevoBloqueIndex)*int64(sb.S_block_s), 0)
			binary.Write(archivo, binary.LittleEndian, &bc)

			// conectamos el bloque al inodo
			inodoPadre.I_block[i] = int32(nuevoBloqueIndex)

			// guardamos el inodo padre
			archivo.Seek(int64(sb.S_inode_start)+int64(indexPadre)*int64(sb.S_inode_s), 0)
			binary.Write(archivo, binary.LittleEndian, &inodoPadre)

			return true
		}
	}

	return false
}

func crearRutaRecursiva(archivo *os.File, sb *types.SuperBloque, ruta string, partStart int64) {
	segmentos := strings.Split(strings.TrimPrefix(ruta, "/"), "/")
	rutaActual := ""

	for _, segmento := range segmentos {
		if segmento == "" {
			continue
		}

		padre := rutaActual
		if padre == "" {
			padre = "/"
		}
		rutaActual += "/" + segmento

		_, _, err := utils.BuscarInodoPorRuta(archivo, *sb, rutaActual)
		if err == nil {
			continue
		}

		// Buscamos al padre
		var padreIndex int32
		var inodoPadre types.Inodo
		if padre == "/" {
			padreIndex = 0
			archivo.Seek(int64(sb.S_inode_start), 0)
			binary.Read(archivo, binary.LittleEndian, &inodoPadre)
		} else {
			pIdx, inP, _ := utils.BuscarInodoPorRuta(archivo, *sb, padre)
			padreIndex = int32(pIdx)
			inodoPadre = inP
		}

		// Creamos el nivel actual
		exito := crearDirectorioFisico(archivo, sb, padreIndex, inodoPadre, segmento, partStart)
		if !exito {
			fmt.Printf("[ERROR] No se pudo crear '%s'.\n", rutaActual)
			return
		}
		fmt.Printf("[ÉXITO] Directorio '%s' creado.\n", rutaActual)
	}
}

func crearDirectorioFisico(archivo *os.File, sb *types.SuperBloque, padreIndex int32, inodoPadre types.Inodo, nombre string, partStart int64) bool {
	// Reservar inodo
	nuevoInodoIndex := sb.S_first_ino
	sb.S_first_ino++
	sb.S_free_inodes_count--
	archivo.Seek(int64(sb.S_bm_inode_start)+int64(nuevoInodoIndex), 0)
	binary.Write(archivo, binary.LittleEndian, byte('1'))

	inodoCarpeta := types.Inodo{
		I_uid: 1, I_gid: 1, I_size: 0, I_type: '0',
		I_perm: [3]byte{'6', '6', '4'},
	}
	for j := 0; j < 15; j++ {
		inodoCarpeta.I_block[j] = -1
	}
	tiempo := time.Now().Format("2006-01-02 15:04:05")
	copy(inodoCarpeta.I_ctime[:], tiempo)
	copy(inodoCarpeta.I_atime[:], tiempo)
	copy(inodoCarpeta.I_mtime[:], tiempo)

	// Reservar bloque
	nuevoBloqueIndex := sb.S_first_blo
	sb.S_first_blo++
	sb.S_free_blocks_count--
	archivo.Seek(int64(sb.S_bm_block_start)+int64(nuevoBloqueIndex), 0)
	binary.Write(archivo, binary.LittleEndian, byte('1'))

	inodoCarpeta.I_block[0] = nuevoBloqueIndex
	var bc types.BloqueCarpeta
	for j := 0; j < 4; j++ {
		bc.B_content[j].B_inodo = -1
	}
	copy(bc.B_content[0].B_name[:], ".")
	bc.B_content[0].B_inodo = int32(nuevoInodoIndex)
	copy(bc.B_content[1].B_name[:], "..")
	bc.B_content[1].B_inodo = padreIndex

	// Guardar bloque e inodo
	archivo.Seek(int64(sb.S_block_start)+int64(nuevoBloqueIndex)*int64(sb.S_block_s), 0)
	binary.Write(archivo, binary.LittleEndian, &bc)

	archivo.Seek(int64(sb.S_inode_start)+int64(nuevoInodoIndex)*int64(sb.S_inode_s), 0)
	binary.Write(archivo, binary.LittleEndian, &inodoCarpeta)

	// Enlazar al inodo padre
	exito := EnlazarInodoEnCarpeta(archivo, sb, inodoPadre, padreIndex, int32(nuevoInodoIndex), nombre)
	if !exito {
		fmt.Println("[ERROR] Carpeta padre llena. Falta implementar indirectos.")
		return false
	}

	// Consolidar el Superbloque
	archivo.Seek(partStart, 0)
	binary.Write(archivo, binary.LittleEndian, sb)
	return true
}
