package filesystem

import (
	"MIAP1/types"
	"MIAP1/users"
	"MIAP1/utils"
	"bufio"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func Mkfile(ruta string, r bool, size int, rutaContenido string) {
	contenidoAEscribir := ""

	if !users.SesionActiva {
		fmt.Println("[ERROR] No hay sesión activa.")
		return
	}

	if size < 0 {
		fmt.Println("[ERROR] El tamaño no puede ser negativo.")
		return
	}

	// -cont tiene prioridad sobre -size, pero la ruta debe existir
	var contenidoExterno string
	usarCont := rutaContenido != ""
	if usarCont {
		datos, errLectura := os.ReadFile(rutaContenido)
		if errLectura != nil {
			fmt.Printf("[ERROR] No se pudo leer el archivo de contenido: '%s'.\n", rutaContenido)
			return
		}
		contenidoExterno = string(datos)
	}

	ruta = strings.TrimSpace(ruta)
	ruta = strings.ReplaceAll(ruta, "\"", "")

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
		if !crearRutaRecursiva(archivo, &sb, directorioPadre, partStart) {
			fmt.Println("[ERROR] Falló la creación de las carpetas padre.")
			return
		}
		padreIndex, inodoPadre, errPadre = utils.BuscarInodoPorRuta(archivo, sb, directorioPadre)
		if errPadre != nil {
			fmt.Println("[ERROR] Falló la creación de las carpetas padre.")
			return
		}
	}

	if !utils.TienePermiso(users.UsuarioActualUID, users.UsuarioActualGID, inodoPadre, 2) {
		fmt.Printf("[ERROR] No tiene permiso de escritura sobre '%s'.\n", directorioPadre)
		return
	}

	inodoIndexExistente, inodoExistente, errArchivo := utils.BuscarInodoPorRuta(archivo, sb, ruta)
	sobreescribiendo := false
	if errArchivo == nil {
		if inodoExistente.I_type == '0' {
			fmt.Printf("[ERROR] '%s' ya existe y es una carpeta.\n", ruta)
			return
		}
		if !confirmarSobreescritura(ruta) {
			fmt.Println("[INFO] Operación cancelada por el usuario.")
			return
		}
		sobreescribiendo = true
	}

	var contenidoNuevo string
	if usarCont {
		contenidoNuevo = contenidoExterno
	} else {
		contenidoNuevo = generarContenidoMkfile(size)
	}

	var inodoArchivo types.Inodo
	var nuevoInodoIndex int32

	if sobreescribiendo {
		nuevoInodoIndex = int32(inodoIndexExistente)
		inodoArchivo = inodoExistente
		// Liberar los bloques viejos antes de escribir los nuevos
		for i := 0; i < 12; i++ {
			if inodoArchivo.I_block[i] != -1 {
				archivo.Seek(int64(sb.S_bm_block_start)+int64(inodoArchivo.I_block[i]), 0)
				binary.Write(archivo, binary.LittleEndian, byte(0))
				sb.S_free_blocks_count++
				inodoArchivo.I_block[i] = -1
			}
		}
	} else {
		nuevoInodoIndex = utils.BuscarInodoLibre(archivo, &sb)
		if nuevoInodoIndex == -1 {
			fmt.Println("[ERROR] No hay inodos libres. Disco lleno.")
			return
		}
		inodoArchivo = types.Inodo{
			I_uid:  users.UsuarioActualUID,
			I_gid:  users.UsuarioActualGID,
			I_type: '1',
			I_perm: [3]byte{'6', '6', '4'},
		}
		for i := 0; i < 15; i++ {
			inodoArchivo.I_block[i] = -1
		}
	}

	inodoArchivo.I_size = int64(len(contenidoNuevo))
	tiempo := time.Now().Format("2006-01-02 15:04:05")
	copy(inodoArchivo.I_mtime[:], tiempo)
	if !sobreescribiendo {
		copy(inodoArchivo.I_atime[:], tiempo)
		copy(inodoArchivo.I_ctime[:], tiempo)
	}

	restante := contenidoNuevo
	for i := 0; i < 12; i++ {
		if len(restante) == 0 {
			break
		}
		pedazo := restante
		if len(restante) > 64 {
			pedazo = restante[:64]
			restante = restante[64:]
		} else {
			restante = ""
		}

		nuevoBloque := utils.BuscarBloqueLibre(archivo, &sb)
		if nuevoBloque == -1 {
			fmt.Println("[ERROR] No hay bloques libres. Disco lleno.")
			return
		}
		inodoArchivo.I_block[i] = nuevoBloque

		var ba types.BloqueArchivo
		copy(ba.B_content[:], pedazo)
		archivo.Seek(int64(sb.S_block_start)+int64(nuevoBloque)*int64(sb.S_block_s), 0)
		binary.Write(archivo, binary.LittleEndian, &ba)
	}

	archivo.Seek(int64(sb.S_inode_start)+int64(nuevoInodoIndex)*int64(sb.S_inode_s), 0)
	binary.Write(archivo, binary.LittleEndian, &inodoArchivo)

	if !sobreescribiendo {
		exitoEnlace := EnlazarInodoEnCarpeta(archivo, &sb, inodoPadre, int32(padreIndex), nuevoInodoIndex, nombreArchivo)
		if !exitoEnlace {
			fmt.Println("[ERROR] No hay espacio en la carpeta padre para este archivo.")
			return
		}
	}

	archivo.Seek(partStart, 0)
	binary.Write(archivo, binary.LittleEndian, &sb)

	fmt.Printf("[ÉXITO] Archivo creado en %s con tamaño %d bytes.\n", ruta, len(contenidoNuevo))

	fmt.Println("[ÉXITO] Archivo creado en EXT2.")
	utils.ReflejarCreacion(ruta, false, contenidoAEscribir)

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

	if p {
		if !crearRutaRecursiva(archivo, &sb, ruta, partStart) {
			return
		}
		archivo.Seek(partStart, 0)
		binary.Write(archivo, binary.LittleEndian, &sb)
		return
	}

	_, _, errExiste := utils.BuscarInodoPorRuta(archivo, sb, ruta)
	if errExiste == nil {
		fmt.Printf("[ERROR] El directorio '%s' ya existe.\n", ruta)
		return
	}

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

	if !utils.TienePermiso(users.UsuarioActualUID, users.UsuarioActualGID, inodoPadre, 2) {
		fmt.Printf("[ERROR] No tiene permiso de escritura sobre '%s'.\n", directorioPadre)
		return
	}

	exito := crearDirectorioFisico(archivo, &sb, padreIndex, inodoPadre, nombreNuevaCarpeta, partStart)
	if exito {
		archivo.Seek(partStart, 0)
		binary.Write(archivo, binary.LittleEndian, &sb)
		fmt.Printf("[ÉXITO] Directorio '%s' creado.\n", ruta)
	}
	fmt.Println("[ÉXITO] Carpeta creada en EXT2.")
	utils.ReflejarCreacion(ruta, true, "")
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
	for i := 0; i < 12; i++ {
		if inodoPadre.I_block[i] != -1 {
			var bc types.BloqueCarpeta
			archivo.Seek(int64(sb.S_block_start)+int64(inodoPadre.I_block[i])*int64(sb.S_block_s), 0)
			binary.Read(archivo, binary.LittleEndian, &bc)

			for j := 0; j < 4; j++ {
				if bc.B_content[j].B_inodo == -1 {
					bc.B_content[j].B_inodo = nuevoInodo
					copy(bc.B_content[j].B_name[:], nombreArchivo)
					archivo.Seek(int64(sb.S_block_start)+int64(inodoPadre.I_block[i])*int64(sb.S_block_s), 0)
					binary.Write(archivo, binary.LittleEndian, &bc)
					return true
				}
			}
		}
	}

	for i := 0; i < 12; i++ {
		if inodoPadre.I_block[i] == -1 {
			nuevoBloqueIndex := utils.BuscarBloqueLibre(archivo, sb)
			if nuevoBloqueIndex == -1 {
				return false
			}

			var bc types.BloqueCarpeta
			for j := 0; j < 4; j++ {
				bc.B_content[j].B_inodo = -1
			}
			bc.B_content[0].B_inodo = nuevoInodo
			copy(bc.B_content[0].B_name[:], nombreArchivo)

			archivo.Seek(int64(sb.S_block_start)+int64(nuevoBloqueIndex)*int64(sb.S_block_s), 0)
			binary.Write(archivo, binary.LittleEndian, &bc)

			inodoPadre.I_block[i] = nuevoBloqueIndex

			archivo.Seek(int64(sb.S_inode_start)+int64(indexPadre)*int64(sb.S_inode_s), 0)
			binary.Write(archivo, binary.LittleEndian, &inodoPadre)

			return true
		}
	}
	return false
}

func crearRutaRecursiva(archivo *os.File, sb *types.SuperBloque, ruta string, partStart int64) bool {
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
			continue // este nivel ya existe, seguimos con el siguiente
		}

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

		if !utils.TienePermiso(users.UsuarioActualUID, users.UsuarioActualGID, inodoPadre, 2) {
			fmt.Printf("[ERROR] No tiene permiso de escritura sobre '%s'.\n", padre)
			return false
		}

		if !crearDirectorioFisico(archivo, sb, padreIndex, inodoPadre, segmento, partStart) {
			fmt.Printf("[ERROR] No se pudo crear '%s'.\n", rutaActual)
			return false
		}
		fmt.Printf("[ÉXITO] Directorio '%s' creado.\n", rutaActual)
	}
	return true
}

func crearDirectorioFisico(archivo *os.File, sb *types.SuperBloque, padreIndex int32, inodoPadre types.Inodo, nombre string, partStart int64) bool {
	nuevoInodoIndex := utils.BuscarInodoLibre(archivo, sb)
	if nuevoInodoIndex == -1 {
		fmt.Println("[ERROR] No hay inodos libres. Disco lleno.")
		return false
	}

	inodoCarpeta := types.Inodo{
		I_uid: users.UsuarioActualUID, I_gid: users.UsuarioActualGID,
		I_size: 0, I_type: '0',
		I_perm: [3]byte{'6', '6', '4'},
	}
	for j := 0; j < 15; j++ {
		inodoCarpeta.I_block[j] = -1
	}
	tiempo := time.Now().Format("2006-01-02 15:04:05")
	copy(inodoCarpeta.I_ctime[:], tiempo)
	copy(inodoCarpeta.I_atime[:], tiempo)
	copy(inodoCarpeta.I_mtime[:], tiempo)

	nuevoBloqueIndex := utils.BuscarBloqueLibre(archivo, sb)
	if nuevoBloqueIndex == -1 {
		fmt.Println("[ERROR] No hay bloques libres. Disco lleno.")
		return false
	}

	inodoCarpeta.I_block[0] = nuevoBloqueIndex
	var bc types.BloqueCarpeta
	for j := 0; j < 4; j++ {
		bc.B_content[j].B_inodo = -1
	}
	copy(bc.B_content[0].B_name[:], ".")
	bc.B_content[0].B_inodo = nuevoInodoIndex
	copy(bc.B_content[1].B_name[:], "..")
	bc.B_content[1].B_inodo = padreIndex

	archivo.Seek(int64(sb.S_block_start)+int64(nuevoBloqueIndex)*int64(sb.S_block_s), 0)
	binary.Write(archivo, binary.LittleEndian, &bc)

	archivo.Seek(int64(sb.S_inode_start)+int64(nuevoInodoIndex)*int64(sb.S_inode_s), 0)
	binary.Write(archivo, binary.LittleEndian, &inodoCarpeta)

	exito := EnlazarInodoEnCarpeta(archivo, sb, inodoPadre, padreIndex, nuevoInodoIndex, nombre)
	if !exito {
		fmt.Println("[ERROR] Carpeta padre llena. Falta implementar indirectos.")
		return false
	}

	archivo.Seek(partStart, 0)
	binary.Write(archivo, binary.LittleEndian, sb)
	return true
}

func Remove(ruta string) {
	if !users.SesionActiva {
		fmt.Println("[ERROR] No hay sesión activa.")
		return
	}

	archivo, sb, _, _, _, err := utils.ObtenerContextoParticion()
	if err != nil {
		return
	}
	defer archivo.Close()

	// 1. Separar la ruta para obtener el padre y el nombre a eliminar
	rutaLimpia := strings.TrimRight(ruta, "/")
	partes := strings.Split(rutaLimpia, "/")
	nombreAEliminar := partes[len(partes)-1]

	rutaPadre := "/"
	if len(partes) > 2 {
		rutaPadre = strings.Join(partes[:len(partes)-1], "/")
	}

	// 2. Buscar Inodo del Padre
	_, inodoPadre, errP := utils.BuscarInodoPorRuta(archivo, sb, rutaPadre)
	if errP != nil {
		fmt.Println("[ERROR] La ruta base (padre) no existe.")
		return
	}

	// 3. Iterar los bloques del padre para encontrar la referencia y borrarla
	modificado := false
	for i := 0; i < 12; i++ {
		if inodoPadre.I_block[i] != -1 {
			var bc types.BloqueCarpeta
			offset := int64(sb.S_block_start) + (int64(inodoPadre.I_block[i]) * int64(sb.S_block_s))
			archivo.Seek(offset, 0)
			binary.Read(archivo, binary.LittleEndian, &bc)

			// Buscar el archivo/carpeta a eliminar
			for j := 0; j < 4; j++ {
				nombre := strings.TrimRight(string(bc.B_content[j].B_name[:]), "\x00")
				if nombre == nombreAEliminar {

					// ELIMINACIÓN LÓGICA:
					// 1. Vaciamos el nombre
					bc.B_content[j].B_name = [12]byte{}
					// 2. Rompemos el apuntador al Inodo (-1)
					bc.B_content[j].B_inodo = -1

					// Guardar el bloque carpeta modificado en el disco .dsk
					archivo.Seek(offset, 0)
					binary.Write(archivo, binary.LittleEndian, &bc)
					modificado = true
					break
				}
			}
		}
		if modificado {
			break
		}
	}

	if modificado {
		archivo.Sync()
		fmt.Printf("[ÉXITO] '%s' eliminado correctamente del sistema EXT2.\n", nombreAEliminar)

		// LLAMADA AL ESPEJO: Borrar el archivo en tu carpeta física de Ubuntu
		utils.ReflejarEliminacion(ruta)
	} else {
		fmt.Println("[ERROR] No se encontró el archivo/carpeta a eliminar.")
	}
}

func Move(rutaOrigen string, rutaDestino string) {
	if !users.SesionActiva {
		fmt.Println("[ERROR] No hay sesión activa.")
		return
	}
	rutaVirtualLimpiaOrigen := strings.TrimPrefix(rutaOrigen, "/")
	rutaVirtualLimpiaDestino := strings.TrimPrefix(rutaDestino, "/")

	// Obtenemos el nombre de lo que estamos moviendo
	partesOrigen := strings.Split(rutaVirtualLimpiaOrigen, "/")
	nombreElemento := partesOrigen[len(partesOrigen)-1]

	rutaFisicaOrigen := filepath.Join(utils.RutaBaseEspejo, rutaVirtualLimpiaOrigen)
	rutaFisicaDestino := filepath.Join(utils.RutaBaseEspejo, rutaVirtualLimpiaDestino, nombreElemento)

	err := os.Rename(rutaFisicaOrigen, rutaFisicaDestino)
	if err != nil {
		fmt.Printf("[ESPEJO-ERROR] No se pudo mover físicamente: %v\n", err)
	} else {
		fmt.Printf("[ÉXITO] Elemento movido a: %s\n", rutaDestino)
	}
}

func Copy(rutaOrigen string, rutaDestino string) {
	if !users.SesionActiva {
		fmt.Println("[ERROR] No hay sesión activa.")
		return
	}

	rutaVirtualLimpiaOrigen := strings.TrimPrefix(rutaOrigen, "/")
	rutaVirtualLimpiaDestino := strings.TrimPrefix(rutaDestino, "/")

	partesOrigen := strings.Split(rutaVirtualLimpiaOrigen, "/")
	nombreElemento := partesOrigen[len(partesOrigen)-1]

	rutaFisicaOrigen := filepath.Join(utils.RutaBaseEspejo, rutaVirtualLimpiaOrigen)
	rutaFisicaDestino := filepath.Join(utils.RutaBaseEspejo, rutaVirtualLimpiaDestino, nombreElemento)

	// Ejecutamos un comando del sistema operativo para copiar recursivamente
	cmd := exec.Command("cp", "-r", rutaFisicaOrigen, rutaFisicaDestino)
	err := cmd.Run()
	if err != nil {
		fmt.Printf("[ERROR] No se pudo copiar: %v\n", err)
	} else {
		fmt.Printf("[ÉXITO] Elemento copiado de '%s' a '%s'\n", rutaOrigen, rutaDestino)
	}
}

func realizarCopia(archivo *os.File, sb *types.SuperBloque, inodoOrigen types.Inodo, inodoDestino *types.Inodo, nombre string, partStart int64) {
	// Crear nuevo inodo para el duplicado
	nuevoInodoIndex := sb.S_first_ino
	sb.S_first_ino++
	sb.S_free_inodes_count--
	archivo.Seek(int64(sb.S_bm_inode_start)+int64(nuevoInodoIndex), 0)
	binary.Write(archivo, binary.LittleEndian, byte('1'))

	nuevoInodo := inodoOrigen
	// Resetear bloques del nuevo inodo (se asignarán nuevos bloques)
	for i := 0; i < 15; i++ {
		nuevoInodo.I_block[i] = -1
	}

	archivo.Seek(int64(sb.S_inode_start)+int64(nuevoInodoIndex)*int64(sb.S_inode_s), 0)
	binary.Write(archivo, binary.LittleEndian, &nuevoInodo)

	// Si es archivo, copiar bloques
	if inodoOrigen.I_type == '1' {
		for i := 0; i < 12; i++ {
			if inodoOrigen.I_block[i] != -1 {
				nuevoBloque := sb.S_first_blo
				sb.S_first_blo++
				sb.S_free_blocks_count--

				// Copiar contenido del bloque
				var ba types.BloqueArchivo
				archivo.Seek(int64(sb.S_block_start)+int64(inodoOrigen.I_block[i])*int64(sb.S_block_s), 0)
				binary.Read(archivo, binary.LittleEndian, &ba)

				archivo.Seek(int64(sb.S_block_start)+int64(nuevoBloque)*int64(sb.S_block_s), 0)
				binary.Write(archivo, binary.LittleEndian, &ba)

				nuevoInodo.I_block[i] = nuevoBloque
			}
		}
	} else {
		// Si es carpeta, copiar recursivamente sus hijos (requiere iterar bloques)
		// Aquí deberías iterar el contenido del inodo origen y llamar a realizarCopia para cada hijo
	}

	// Enlazar al destino
	EnlazarInodoEnCarpeta(archivo, sb, *inodoDestino, 0, nuevoInodoIndex, nombre) // Nota: el index del padre destino necesitarías obtenerlo

	// Guardar
	archivo.Seek(partStart, 0)
	binary.Write(archivo, binary.LittleEndian, sb)
}

func Fdisk(path string, name string, deleteVal string, addVal string, unit string, size int) {
	archivo, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("[ERROR] No se pudo abrir el disco.")
		return
	}
	defer archivo.Close()

	mbr := utils.ObtenerMBR(archivo)

	// 1. Buscar la partición
	idx := -1
	for i := 0; i < 4; i++ {
		nombrePart := strings.TrimRight(string(mbr.Mbr_partitions[i].Part_name[:]), "\x00")
		if nombrePart == name && mbr.Mbr_partitions[i].Part_status == '1' {
			idx = i
			break
		}
	}

	if idx == -1 {
		fmt.Println("[ERROR] Partición no encontrada.")
		return
	}

	// 2. Lógica de DELETE dentro de tu función Fdisk
	if deleteVal != "" {
		switch strings.ToLower(deleteVal) {
		case "fast":
			mbr.Mbr_partitions[idx].Part_status = '0'
		case "full":
			mbr.Mbr_partitions[idx].Part_status = '0'
			archivo.Seek(mbr.Mbr_partitions[idx].Part_start, 0)
			ceros := make([]byte, mbr.Mbr_partitions[idx].Part_s)
			archivo.Write(ceros)
		default:
			fmt.Println("[ERROR] Tipo de borrado no válido, use 'fast' o 'full'.")
			return
		}

		utils.EscribirEnDisco(archivo, mbr)
		archivo.Sync()
		archivo.Close()

		fmt.Println("[ÉXITO] Partición eliminada.")
		return
	}

	// 3. Lógica de ADD (Redimensionar)
	if addVal != "" {
		valorAdd, _ := strconv.Atoi(addVal)
		tamanioBytes := utils.Tamanio(int64(valorAdd), strings.ToLower(unit))

		// Calculamos el tamaño final deseado
		nuevoTamanio := mbr.Mbr_partitions[idx].Part_s + tamanioBytes

		// Validación 1: No redimensionar a tamaño negativo o cero
		if nuevoTamanio <= 0 {
			fmt.Println("[ERROR] El tamaño resultante no puede ser menor o igual a cero.")
			return
		}

		// Validación 2: ¿Hay una partición siguiente?
		if idx < 3 && mbr.Mbr_partitions[idx+1].Part_status == '1' {
			// Verificamos espacio contra la siguiente partición
			if (mbr.Mbr_partitions[idx].Part_start + nuevoTamanio) > mbr.Mbr_partitions[idx+1].Part_start {
				fmt.Println("[ERROR] No hay espacio suficiente: colisiona con la siguiente partición.")
				return
			}
		} else {
			// Validación 3: Si es la última partición (o no hay siguiente), validamos contra el tamaño total del disco
			// mbr.Mbr_tamano debe ser el tamaño en bytes del archivo .dsk
			if (mbr.Mbr_partitions[idx].Part_start + nuevoTamanio) > mbr.Mbr_tamano {
				fmt.Println("[ERROR] No hay espacio suficiente: excede el tamaño del disco.")
				return
			}
		}

		// Aplicar cambio
		mbr.Mbr_partitions[idx].Part_s = nuevoTamanio
		utils.EscribirEnDisco(archivo, mbr)
		fmt.Printf("[ÉXITO] Partición '%s' redimensionada correctamente a %d bytes.\n", name, nuevoTamanio)
	}
}

func TieneEspacioSuficiente(archivo *os.File, mbr types.MBR, requestedBytes int64) bool {
	var ocupado int64 = 0

	// Sumamos el tamaño de todas las particiones activas
	for i := 0; i < 4; i++ {
		if mbr.Mbr_partitions[i].Part_status == '1' {
			ocupado += mbr.Mbr_partitions[i].Part_s
		}
	}

	// Validamos: (Tamaño ocupado + lo nuevo) <= Tamaño Total del disco
	// mbr.Mbr_tamano debería ser el tamaño total en bytes del disco
	if (ocupado + requestedBytes) > mbr.Mbr_tamano {
		return false
	}

	return true
}

func confirmarSobreescritura(ruta string) bool {
	fmt.Printf("[AVISO] El archivo '%s' ya existe. ¿Desea sobrescribirlo? (s/n): ", ruta)
	reader := bufio.NewReader(os.Stdin)
	respuesta, _ := reader.ReadString('\n')
	respuesta = strings.ToLower(strings.TrimSpace(respuesta))
	return respuesta == "s" || respuesta == "si" || respuesta == "y" || respuesta == "yes"
}

func Rename(ruta string, nuevoNombre string) {
	if !users.SesionActiva {
		fmt.Println("[ERROR] No hay sesión activa.")
		return
	}

	archivo, sb, _, _, _, err := utils.ObtenerContextoParticion()
	if err != nil {
		return
	}
	defer archivo.Close()

	// 1. Separar la ruta para obtener el padre y el nombre actual
	rutaLimpia := strings.TrimRight(ruta, "/")
	partes := strings.Split(rutaLimpia, "/")
	nombreActual := partes[len(partes)-1]

	rutaPadre := "/"
	if len(partes) > 2 {
		rutaPadre = strings.Join(partes[:len(partes)-1], "/")
	}

	// 2. Buscar Inodo del Padre
	_, inodoPadre, errP := utils.BuscarInodoPorRuta(archivo, sb, rutaPadre)
	if errP != nil {
		fmt.Println("[ERROR] La ruta base no existe.")
		return
	}

	// 3. Iterar los bloques del padre para encontrar el archivo/carpeta y renombrarlo
	modificado := false
	for i := 0; i < 12; i++ {
		if inodoPadre.I_block[i] != -1 {
			var bc types.BloqueCarpeta
			offset := int64(sb.S_block_start) + (int64(inodoPadre.I_block[i]) * int64(sb.S_block_s))
			archivo.Seek(offset, 0)
			binary.Read(archivo, binary.LittleEndian, &bc)

			// Validar si el nuevo nombre ya existe
			for _, content := range bc.B_content {
				nombre := strings.TrimRight(string(content.B_name[:]), "\x00")
				if nombre == nuevoNombre {
					fmt.Println("[ERROR] Ya existe un archivo o carpeta con ese nombre.")
					return
				}
			}

			// Buscar el actual y cambiarlo
			for j := 0; j < 4; j++ {
				nombre := strings.TrimRight(string(bc.B_content[j].B_name[:]), "\x00")
				if nombre == nombreActual {
					// Comprobar permisos aquí si es necesario (según el enunciado)

					// Asignar nuevo nombre
					copiaNombre := make([]byte, 12)
					copy(copiaNombre, nuevoNombre)
					copy(bc.B_content[j].B_name[:], copiaNombre)

					// Guardar bloque modificado
					archivo.Seek(offset, 0)
					binary.Write(archivo, binary.LittleEndian, &bc)
					modificado = true
					break
				}
			}
		}
		if modificado {
			break
		}

		if modificado {
			archivo.Sync()
			fmt.Printf("[ÉXITO] Nombre cambiado de '%s' a '%s'.\n", nombreActual, nuevoNombre)

			utils.ReflejarRenombrar(ruta, nuevoNombre)
		}
	}

	if modificado {
		archivo.Sync()
		fmt.Printf("[ÉXITO] Nombre cambiado de '%s' a '%s'.\n", nombreActual, nuevoNombre)
	} else {
		fmt.Println("[ERROR] No se encontró el archivo/carpeta a renombrar.")
	}
}

func Edit(rutaDestino string, rutaContenido string) {
	if !users.SesionActiva {
		fmt.Println("[ERROR] No hay sesión activa.")
		return
	}

	// 1. Leer el archivo físico real en Ubuntu
	rutaContLimpia := strings.ReplaceAll(rutaContenido, "\"", "")
	dataFisica, errOS := os.ReadFile(rutaContLimpia)
	if errOS != nil {
		fmt.Println("[ERROR] No se pudo leer el archivo físico de contenido:", errOS)
		return
	}
	nuevoTexto := string(dataFisica)

	// 2. Obtener contexto EXT2
	archivo, sb, _, _, partStart, err := utils.ObtenerContextoParticion()
	if err != nil {
		return
	}
	defer archivo.Close()

	// 3. Buscar Inodo del archivo destino
	inodoIndex, inodoDestino, errInodo := utils.BuscarInodoPorRuta(archivo, sb, rutaDestino)
	if errInodo != nil {
		fmt.Println("[ERROR] No se encontró el archivo destino en la partición.")
		return
	}

	// Validar que sea un archivo (tipo '1' o 1)
	if inodoDestino.I_type != '1' && inodoDestino.I_type != 1 {
		fmt.Println("[ERROR] La ruta destino no es un archivo editable.")
		return
	}

	// 4. Sobrescribir el archivo
	utils.EscribirArchivoUsers(archivo, &sb, int32(inodoIndex), &inodoDestino, nuevoTexto, partStart)

	// Actualizar SB
	archivo.Seek(partStart, 0)
	binary.Write(archivo, binary.LittleEndian, &sb)
	archivo.Sync()

	fmt.Printf("[ÉXITO] Archivo '%s' editado correctamente con el contenido de '%s'.\n", rutaDestino, rutaContenido)

	fmt.Printf("[ÉXITO] Archivo '%s' editado correctamente.\n", rutaDestino)
	utils.ReflejarCreacion(rutaDestino, false, nuevoTexto)
}
