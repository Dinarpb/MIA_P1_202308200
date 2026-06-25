package filesystem

import (
	"MIAP1/types"
	"MIAP1/users"
	"MIAP1/utils"
	"bufio"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func Mkfile(ruta string, r bool, size int, rutaContenido string) {
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

func Rename(ruta string, nuevoNombre string) {
	// 1. Limpieza absoluta contra caracteres basura
	ruta = strings.TrimSpace(ruta)
	ruta = strings.ReplaceAll(ruta, "\r", "")
	ruta = strings.ReplaceAll(ruta, "\n", "")
	ruta = strings.ReplaceAll(ruta, "\"", "")
	ruta = strings.TrimSuffix(ruta, "/")

	nuevoNombre = strings.TrimSpace(nuevoNombre)
	nuevoNombre = strings.ReplaceAll(nuevoNombre, "\r", "")
	nuevoNombre = strings.ReplaceAll(nuevoNombre, "\n", "")
	nuevoNombre = strings.ReplaceAll(nuevoNombre, "\"", "")

	if !users.SesionActiva {
		fmt.Println("[ERROR] No hay sesión activa.")
		return
	}

	archivo, sb, _, _, _, err := utils.ObtenerContextoParticion()
	if err != nil {
		fmt.Println("[ERROR] No se pudo acceder a la partición activa.")
		return
	}
	defer archivo.Close()

	directorioPadre := filepath.Dir(ruta)
	nombreActual := filepath.Base(ruta)

	// 2. Buscar Inodo Padre
	_, inodoPadre, errPadre := utils.BuscarInodoPorRuta(archivo, sb, directorioPadre)
	if errPadre != nil {
		fmt.Printf("[ERROR] La ruta padre '%s' no existe.\n", directorioPadre)
		return
	}

	// 3. Buscar el nombre actual y verificar que el nuevo nombre no exista
	encontrado := false
	var bloqueModificar int32 = -1
	var indexModificar int = -1
	var bcModificar types.BloqueCarpeta

	for i := 0; i < 12; i++ {
		if inodoPadre.I_block[i] != -1 {
			var bc types.BloqueCarpeta
			offset := int64(sb.S_block_start) + (int64(inodoPadre.I_block[i]) * int64(sb.S_block_s))
			archivo.Seek(offset, 0)
			binary.Read(archivo, binary.LittleEndian, &bc)

			for j := 0; j < 4; j++ {
				if bc.B_content[j].B_inodo != -1 {
					nombreItem := strings.TrimRight(string(bc.B_content[j].B_name[:]), "\x00")

					// Regla del manual: No puede llamarse igual a otro existente
					if nombreItem == nuevoNombre {
						fmt.Printf("[ERROR] Ya existe un archivo o carpeta con el nombre '%s'.\n", nuevoNombre)
						return
					}

					// Identificar el bloque y posición que vamos a modificar
					if nombreItem == nombreActual {
						encontrado = true
						bloqueModificar = inodoPadre.I_block[i]
						indexModificar = j
						bcModificar = bc
					}
				}
			}
		}
	}

	if !encontrado {
		fmt.Printf("[ERROR] El elemento '%s' no existe.\n", nombreActual)
		return
	}

	// 4. Aplicar el cambio de nombre en la memoria RAM
	for k := 0; k < len(bcModificar.B_content[indexModificar].B_name); k++ {
		bcModificar.B_content[indexModificar].B_name[k] = 0 // Limpiar caracteres viejos (\x00)
	}
	copy(bcModificar.B_content[indexModificar].B_name[:], nuevoNombre)

	// 5. Escribir el bloque actualizado al disco físico
	offsetMod := int64(sb.S_block_start) + (int64(bloqueModificar) * int64(sb.S_block_s))
	archivo.Seek(offsetMod, 0)
	binary.Write(archivo, binary.LittleEndian, &bcModificar)

	fmt.Printf("[ÉXITO] Se renombró '%s' a '%s' correctamente.\n", nombreActual, nuevoNombre)
}

func Remove(ruta string) {
	// 1. Limpieza absoluta contra caracteres basura
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
		fmt.Println("[ERROR] No se pudo acceder a la partición activa.")
		return
	}
	defer archivo.Close()

	directorioPadre := filepath.Dir(ruta)
	nombreEliminar := filepath.Base(ruta)

	// 2. Buscar al Inodo Padre
	_, inodoPadre, errPadre := utils.BuscarInodoPorRuta(archivo, sb, directorioPadre)
	if errPadre != nil {
		fmt.Printf("[ERROR] La ruta padre '%s' no existe.\n", directorioPadre)
		return
	}

	// 3. Buscar el elemento a eliminar dentro del padre y desenlazarlo
	encontrado := false
	var bloqueModificar int32 = -1
	var indexModificar int = -1
	var bcModificar types.BloqueCarpeta
	var inodoDestinoIndex int32 = -1

	for i := 0; i < 12; i++ {
		if inodoPadre.I_block[i] != -1 {
			var bc types.BloqueCarpeta
			offset := int64(sb.S_block_start) + (int64(inodoPadre.I_block[i]) * int64(sb.S_block_s))
			archivo.Seek(offset, 0)
			binary.Read(archivo, binary.LittleEndian, &bc)

			for j := 0; j < 4; j++ {
				if bc.B_content[j].B_inodo != -1 {
					nombreItem := strings.TrimRight(string(bc.B_content[j].B_name[:]), "\x00")

					if nombreItem == nombreEliminar {
						encontrado = true
						bloqueModificar = inodoPadre.I_block[i]
						indexModificar = j
						bcModificar = bc
						inodoDestinoIndex = bc.B_content[j].B_inodo
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
		fmt.Printf("[ERROR] El elemento '%s' no existe.\n", nombreEliminar)
		return
	}

	// NOTA DE PERMISOS: Aquí deberías validar si el usuario actual (users.UsuarioActual.Uid)
	// tiene permisos de escritura ('2' o '6' o '7') sobre el inodo que está en inodoDestinoIndex.
	// Si no tiene, haces un return.

	// 4. Aplicar la eliminación lógica (Desenlazar)
	bcModificar.B_content[indexModificar].B_inodo = -1
	for k := 0; k < len(bcModificar.B_content[indexModificar].B_name); k++ {
		bcModificar.B_content[indexModificar].B_name[k] = 0 // Limpiar caracteres
	}

	// 5. Escribir el bloque del padre actualizado al disco físico
	offsetMod := int64(sb.S_block_start) + (int64(bloqueModificar) * int64(sb.S_block_s))
	archivo.Seek(offsetMod, 0)
	binary.Write(archivo, binary.LittleEndian, &bcModificar)

	// 6. Liberar el inodo en el Bitmap (Para que pueda ser reusado)
	archivo.Seek(int64(sb.S_bm_inode_start)+int64(inodoDestinoIndex), 0)
	binary.Write(archivo, binary.LittleEndian, byte('0'))

	sb.S_free_inodes_count++ // Aumentamos la cantidad de inodos libres

	// 7. Actualizar el SuperBloque
	archivo.Seek(partStart, 0)
	binary.Write(archivo, binary.LittleEndian, &sb)

	fmt.Printf("[ÉXITO] Se eliminó '%s' correctamente del árbol.\n", nombreEliminar)
}

func Edit(ruta string, rutaContenido string) {
	// 1. Limpieza absoluta de rutas
	ruta = strings.TrimSpace(ruta)
	ruta = strings.ReplaceAll(ruta, "\r", "")
	ruta = strings.ReplaceAll(ruta, "\n", "")
	ruta = strings.ReplaceAll(ruta, "\"", "")

	rutaContenido = strings.TrimSpace(rutaContenido)
	rutaContenido = strings.ReplaceAll(rutaContenido, "\r", "")
	rutaContenido = strings.ReplaceAll(rutaContenido, "\n", "")
	rutaContenido = strings.ReplaceAll(rutaContenido, "\"", "")

	if !users.SesionActiva {
		fmt.Println("[ERROR] No hay sesión activa.")
		return
	}

	// 2. Leer el archivo FÍSICO desde tu sistema operativo real
	nuevoContenido, errSO := os.ReadFile(rutaContenido)
	if errSO != nil {
		fmt.Printf("[ERROR] No se pudo leer el archivo físico en: '%s'\n", rutaContenido)
		return
	}
	cadenaContenido := string(nuevoContenido)

	archivo, sb, _, _, partStart, err := utils.ObtenerContextoParticion()
	if err != nil {
		fmt.Println("[ERROR] No se pudo acceder a la partición activa.")
		return
	}
	defer archivo.Close()

	// 3. Buscar el inodo del archivo a editar en tu .dsk
	inodoIndex, inodoArchivo, errArchivo := utils.BuscarInodoPorRuta(archivo, sb, ruta)
	if errArchivo != nil {
		fmt.Printf("[ERROR] El archivo '%s' no existe en el disco simulado.\n", ruta)
		return
	}

	// 4. Validar que sea un archivo ('1') y no una carpeta ('0')
	if inodoArchivo.I_type == '0' {
		fmt.Println("[ERROR] La ruta especificada es una carpeta. Solo se pueden editar archivos.")
		return
	}

	if !utils.TienePermiso(users.UsuarioActualUID, users.UsuarioActualGID, inodoArchivo, 2) {
		fmt.Println("[ERROR] Permiso denegado: El usuario no tiene derechos de escritura sobre este archivo.")
		return
	}

	// 5. Liberar los bloques de datos antiguos en el Bitmap
	for i := 0; i < 12; i++ {
		if inodoArchivo.I_block[i] != -1 {
			archivo.Seek(int64(sb.S_bm_block_start)+int64(inodoArchivo.I_block[i]), 0)
			binary.Write(archivo, binary.LittleEndian, byte('0')) // Marcar como libre
			inodoArchivo.I_block[i] = -1
			sb.S_free_blocks_count++
		}
	}

	// 6. Calcular cuántos bloques nuevos necesitamos (cada bloque de archivo guarda 64 bytes)
	bloquesNecesarios := int32(math.Ceil(float64(len(cadenaContenido)) / 64.0))
	if bloquesNecesarios > 12 {
		fmt.Println("[ERROR] El contenido es muy grande. Falta implementar bloques indirectos.")
		return
	}

	// 7. Escribir el nuevo contenido dividiéndolo en Bloques de Archivo
	inodoArchivo.I_size = int64(len(cadenaContenido))
	caracteresEscritos := 0

	for i := 0; int32(i) < bloquesNecesarios; i++ {
		// Reservar un nuevo bloque libre
		nuevoBloqueIndex := sb.S_first_blo
		sb.S_first_blo++
		sb.S_free_blocks_count--

		// Marcarlo en el bitmap
		archivo.Seek(int64(sb.S_bm_block_start)+int64(nuevoBloqueIndex), 0)
		binary.Write(archivo, binary.LittleEndian, byte('1'))

		// Llenar los 64 bytes de este bloque
		var ba types.BloqueArchivo
		for j := 0; j < 64; j++ {
			if caracteresEscritos < len(cadenaContenido) {
				ba.B_content[j] = cadenaContenido[caracteresEscritos]
				caracteresEscritos++
			} else {
				ba.B_content[j] = 0 // Espacios vacíos al final
			}
		}

		// Enlazar el bloque al Inodo y guardarlo en el disco físico
		inodoArchivo.I_block[i] = nuevoBloqueIndex
		offsetBloque := int64(sb.S_block_start) + (int64(nuevoBloqueIndex) * int64(sb.S_block_s))
		archivo.Seek(offsetBloque, 0)
		binary.Write(archivo, binary.LittleEndian, &ba)
	}

	// 8. Actualizar fecha de modificación (Mtime)
	tiempo := time.Now().Format("2006-01-02 15:04:05")
	copy(inodoArchivo.I_mtime[:], tiempo)

	// 9. Sobreescribir el Inodo y el SuperBloque en el disco
	offsetInodo := int64(sb.S_inode_start) + (int64(inodoIndex) * int64(sb.S_inode_s))
	archivo.Seek(offsetInodo, 0)
	binary.Write(archivo, binary.LittleEndian, &inodoArchivo)

	archivo.Seek(partStart, 0)
	binary.Write(archivo, binary.LittleEndian, &sb)

	fmt.Printf("[ÉXITO] El archivo '%s' fue editado correctamente.\n", ruta)
}

func Move(rutaOrigen string, rutaDestino string) {
	// Limpieza absoluta de rutas
	rutaOrigen = strings.TrimSpace(rutaOrigen)
	rutaOrigen = strings.ReplaceAll(rutaOrigen, "\r", "")
	rutaOrigen = strings.ReplaceAll(rutaOrigen, "\n", "")
	rutaOrigen = strings.ReplaceAll(rutaOrigen, "\"", "")
	rutaOrigen = strings.TrimSuffix(rutaOrigen, "/")

	rutaDestino = strings.TrimSpace(rutaDestino)
	rutaDestino = strings.ReplaceAll(rutaDestino, "\r", "")
	rutaDestino = strings.ReplaceAll(rutaDestino, "\n", "")
	rutaDestino = strings.ReplaceAll(rutaDestino, "\"", "")
	rutaDestino = strings.TrimSuffix(rutaDestino, "/")

	if !users.SesionActiva {
		fmt.Println("[ERROR] No hay sesión activa.")
		return
	}

	archivo, sb, _, inodoUsers, partStart, err := utils.ObtenerContextoParticion()
	if err != nil {
		fmt.Println("[ERROR] No se pudo acceder a la partición.")
		return
	}
	defer archivo.Close()

	obtenerIdsUsuarioActual := func() (int32, int32, error) {
		contenido := utils.LeerArchivoUsers(archivo, sb, inodoUsers)
		lineas := strings.Split(contenido, "\n")
		var uidActual int32 = -1
		var nombreGrupo string

		for _, linea := range lineas {
			linea = strings.TrimSpace(linea)
			if linea == "" {
				continue
			}
			datos := strings.Split(linea, ",")
			if len(datos) < 5 {
				continue
			}
			if strings.TrimSpace(datos[1]) != "U" {
				continue
			}
			if strings.TrimSpace(datos[3]) == users.UsuarioActual {
				uid, errConv := strconv.Atoi(strings.TrimSpace(datos[0]))
				if errConv != nil {
					return -1, -1, errConv
				}
				uidActual = int32(uid)
				nombreGrupo = strings.TrimSpace(datos[2])
				break
			}
		}

		if uidActual == -1 {
			return -1, -1, fmt.Errorf("usuario actual no encontrado")
		}

		var gidActual int32 = -1
		for _, linea := range lineas {
			linea = strings.TrimSpace(linea)
			if linea == "" {
				continue
			}
			datos := strings.Split(linea, ",")
			if len(datos) < 3 {
				continue
			}
			if strings.TrimSpace(datos[1]) != "G" {
				continue
			}
			if strings.TrimSpace(datos[2]) == nombreGrupo {
				gid, errConv := strconv.Atoi(strings.TrimSpace(datos[0]))
				if errConv != nil {
					return -1, -1, errConv
				}
				switch {
				case users.UsuarioActual == "root":
					return int32(uidActual), int32(gid), nil
				default:
					gidActual = int32(gid)
				}
				break
			}
		}

		if gidActual == -1 {
			return -1, -1, fmt.Errorf("grupo del usuario no encontrado")
		}

		return uidActual, gidActual, nil
	}

	tienePermisoEscritura := func(inodo types.Inodo, uidActual int32, gidActual int32) bool {
		if users.UsuarioActual == "root" {
			return true
		}

		permiso := inodo.I_perm[2]
		if inodo.I_uid == uidActual {
			permiso = inodo.I_perm[0]
		} else if inodo.I_gid == gidActual {
			permiso = inodo.I_perm[1]
		}

		if permiso < '0' || permiso > '7' {
			return false
		}

		return (permiso-'0')&2 != 0
	}

	if rutaOrigen == "/" {
		fmt.Println("[ERROR] No se puede mover la raíz del sistema.")
		return
	}

	if rutaDestino == rutaOrigen || strings.HasPrefix(rutaDestino+"/", rutaOrigen+"/") {
		fmt.Println("[ERROR] No se puede mover un elemento dentro de sí mismo o de su subdirectorio.")
		return
	}

	padreOrigen := filepath.Dir(rutaOrigen)
	nombreElemento := filepath.Base(rutaOrigen)

	// Buscar al Padre Origen
	_, inodoPadreOrigen, errPadreO := utils.BuscarInodoPorRuta(archivo, sb, padreOrigen)
	if errPadreO != nil {
		fmt.Printf("[ERROR] La ruta origen '%s' no existe.\n", padreOrigen)
		return
	}

	// Buscar y aislar el elemento que vamos a mover dentro del padre origen
	encontrado := false
	var bloqueModificar int32 = -1
	var indexModificar int = -1
	var bcModificar types.BloqueCarpeta
	var inodoMoverIndex int32 = -1

	for i := 0; i < 12; i++ {
		if inodoPadreOrigen.I_block[i] != -1 {
			var bc types.BloqueCarpeta
			offset := int64(sb.S_block_start) + (int64(inodoPadreOrigen.I_block[i]) * int64(sb.S_block_s))
			archivo.Seek(offset, 0)
			binary.Read(archivo, binary.LittleEndian, &bc)

			for j := 0; j < 4; j++ {
				if bc.B_content[j].B_inodo != -1 {
					nombreItem := strings.TrimRight(string(bc.B_content[j].B_name[:]), "\x00")
					if nombreItem == nombreElemento {
						encontrado = true
						bloqueModificar = inodoPadreOrigen.I_block[i]
						indexModificar = j
						bcModificar = bc
						inodoMoverIndex = bc.B_content[j].B_inodo
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
		fmt.Printf("[ERROR] El elemento '%s' no existe en el origen.\n", nombreElemento)
		return
	}

	var inodoMover types.Inodo
	offsetInodoMover := int64(sb.S_inode_start) + (int64(inodoMoverIndex) * int64(sb.S_inode_s))
	archivo.Seek(offsetInodoMover, 0)
	binary.Read(archivo, binary.LittleEndian, &inodoMover)

	uidActual, gidActual, errIds := obtenerIdsUsuarioActual()
	if errIds != nil {
		fmt.Println("[ERROR] No se pudo validar los permisos del usuario.")
		return
	}

	if !tienePermisoEscritura(inodoMover, uidActual, gidActual) {
		fmt.Printf("[ERROR] No tiene permiso de escritura sobre el origen '%s'.\n", rutaOrigen)
		return
	}

	// Buscar la Carpeta Destino (Donde va a aterrizar)
	destinoIndex, inodoDestino, errDestino := utils.BuscarInodoPorRuta(archivo, sb, rutaDestino)
	if errDestino != nil {
		fmt.Printf("[ERROR] La ruta destino '%s' no existe.\n", rutaDestino)
		return
	}

	if inodoDestino.I_type == '1' {
		fmt.Println("[ERROR] El destino debe ser una carpeta, no un archivo.")
		return
	}

	if !tienePermisoEscritura(inodoDestino, uidActual, gidActual) {
		fmt.Printf("[ERROR] No tiene permiso de escritura sobre la carpeta destino '%s'.\n", rutaDestino)
		return
	}

	for i := 0; i < 12; i++ {
		if inodoDestino.I_block[i] != -1 {
			var bc types.BloqueCarpeta
			offset := int64(sb.S_block_start) + (int64(inodoDestino.I_block[i]) * int64(sb.S_block_s))
			archivo.Seek(offset, 0)
			binary.Read(archivo, binary.LittleEndian, &bc)

			for j := 0; j < 4; j++ {
				if bc.B_content[j].B_inodo != -1 {
					nombreItem := strings.TrimRight(string(bc.B_content[j].B_name[:]), "\x00")
					if nombreItem == nombreElemento {
						fmt.Printf("[ERROR] Ya existe un elemento con nombre '%s' en el destino.\n", nombreElemento)
						return
					}
				}
			}
		}
	}

	// ENLAZAR AL NUEVO PADRE (Usando tu propia función)
	exitoEnlace := EnlazarInodoEnCarpeta(archivo, &sb, inodoDestino, int32(destinoIndex), inodoMoverIndex, nombreElemento)
	if !exitoEnlace {
		fmt.Println("[ERROR] No se pudo enlazar el archivo en el destino (Carpeta llena).")
		return
	}

	// DESENLAZAR DEL PADRE VIEJO
	bcModificar.B_content[indexModificar].B_inodo = -1
	for k := 0; k < len(bcModificar.B_content[indexModificar].B_name); k++ {
		bcModificar.B_content[indexModificar].B_name[k] = 0 // Limpiar caracteres
	}

	offsetMod := int64(sb.S_block_start) + (int64(bloqueModificar) * int64(sb.S_block_s))
	archivo.Seek(offsetMod, 0)
	binary.Write(archivo, binary.LittleEndian, &bcModificar)

	// Guardar SuperBloque actualizado
	archivo.Seek(partStart, 0)
	binary.Write(archivo, binary.LittleEndian, &sb)

	fmt.Printf("[ÉXITO] Se movió '%s' a '%s' correctamente.\n", nombreElemento, rutaDestino)
}

func Copy(rutaOrigen string, rutaDestino string) {
	// 1. Limpieza
	rutaOrigen = strings.TrimSuffix(strings.ReplaceAll(rutaOrigen, "\"", ""), "/")
	rutaDestino = strings.TrimSuffix(strings.ReplaceAll(rutaDestino, "\"", ""), "/")

	if !users.SesionActiva {
		fmt.Println("[ERROR] No hay sesión activa.")
		return
	}

	archivo, sb, _, _, partStart, err := utils.ObtenerContextoParticion()
	if err != nil {
		return
	}
	defer archivo.Close()

	// 2. Obtener origen
	_, inodoOrigen, errO := utils.BuscarInodoPorRuta(archivo, sb, rutaOrigen)
	if errO != nil {
		fmt.Printf("[ERROR] El origen '%s' no existe.\n", rutaOrigen)
		return
	}

	// 3. Obtener destino (debe ser carpeta)
	_, inodoDestino, errD := utils.BuscarInodoPorRuta(archivo, sb, rutaDestino)
	if errD != nil || inodoDestino.I_type == '1' {
		fmt.Printf("[ERROR] El destino '%s' no es una carpeta válida.\n", rutaDestino)
		return
	}

	// 4. Iniciar copia recursiva
	nombreItem := filepath.Base(rutaOrigen)

	// Validar espacio (simple)
	if inodoDestino.I_type == '0' {
		realizarCopia(archivo, &sb, inodoOrigen, &inodoDestino, nombreItem, partStart)
	}

	fmt.Printf("[ÉXITO] Se copió '%s' a '%s' correctamente.\n", rutaOrigen, rutaDestino)
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
