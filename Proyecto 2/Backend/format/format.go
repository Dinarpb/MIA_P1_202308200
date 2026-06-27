package format

import (
	"MIAP1/global"
	"MIAP1/types"
	"MIAP1/utils"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"time"
	"unsafe"
)

func CalcularEstructuras(tamanioParticion int64) int64 {
	tamanioSuperbloque := int64(unsafe.Sizeof(types.SuperBloque{}))
	tamanioInodo := int64(binary.Size(types.Inodo{}))

	tamanioBloque := int64(1024)

	numerador := tamanioParticion - tamanioSuperbloque
	denominador := 4 + tamanioInodo + (3 * tamanioBloque)

	n := numerador / denominador
	return n
}

func Mkfs(id string, tipo string) {
	var rutaDisco string
	var tamanioParticion int64
	var inicioParticion int64 = -1
	const BLOCK_SIZE = 1024
	encontrado := false

	for _, disco := range global.DiscosMontados {
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
						inicioParticion = mbr.Mbr_partitions[i].Part_start
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

	if !encontrado || inicioParticion == -1 {
		fmt.Println("[ERROR] No se encontró la partición montada con el ID especificado.")
		return
	}

	n := CalcularEstructuras(tamanioParticion)

	// Volvemos a abrir el archivo para escribir
	archivo, err := os.OpenFile(rutaDisco, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("[ERROR] No se pudo abrir el disco para formatear.")
		return
	}
	defer archivo.Close()

	// instanciamos y llenamos el SuperBloque
	superBloque := types.SuperBloque{
		S_filesystem_type:   2,
		S_inodes_count:      int32(n),
		S_blocks_count:      int32(3 * n),
		S_free_blocks_count: int32(3 * n),
		S_free_inodes_count: int32(n),
		S_mnt_count:         1,
		S_magic:             0xEF53,
		S_inode_s:           int32(binary.Size(types.Inodo{})),
		S_block_s:           int32(BLOCK_SIZE), S_first_ino: 0,
		S_first_blo: 0,
	}

	// Calculamos en que byte arranca cada estructura
	superBloque.S_bm_inode_start = int32(inicioParticion + int64(unsafe.Sizeof(types.SuperBloque{})))
	superBloque.S_bm_block_start = superBloque.S_bm_inode_start + int32(n)
	superBloque.S_inode_start = superBloque.S_bm_block_start + int32(3*n)
	superBloque.S_block_start = superBloque.S_inode_start + int32(n*int64(unsafe.Sizeof(types.Inodo{})))

	// Agregamos las fechas
	fechaActual := time.Now().Format("2006-01-02 15:04:05")
	copy(superBloque.S_mtime[:], fechaActual)
	copy(superBloque.S_umtime[:], fechaActual)

	//Escribimos el Superbloque
	archivo.Seek(inicioParticion, 0)
	err = binary.Write(archivo, binary.LittleEndian, &superBloque)
	if err != nil {
		fmt.Println("[ERROR] Falló la escritura del Superbloque.")
		return
	}

	cerosInodos := make([]byte, n)
	archivo.Seek(int64(superBloque.S_bm_inode_start), 0)
	binary.Write(archivo, binary.LittleEndian, &cerosInodos)

	cerosBloques := make([]byte, 3*n)
	archivo.Seek(int64(superBloque.S_bm_block_start), 0)
	binary.Write(archivo, binary.LittleEndian, &cerosBloques)

	cerosInodos[0] = '1'
	cerosInodos[1] = '1'
	cerosBloques[0] = '1'
	cerosBloques[1] = '1'

	archivo.Seek(int64(superBloque.S_bm_inode_start), 0)
	binary.Write(archivo, binary.LittleEndian, &cerosInodos)

	archivo.Seek(int64(superBloque.S_bm_block_start), 0)
	binary.Write(archivo, binary.LittleEndian, &cerosBloques)

	inodoRaiz := types.Inodo{
		I_uid:  1,
		I_gid:  1,
		I_size: 0,
		I_type: '0', // '0' significa Carpeta
		I_perm: [3]byte{'6', '6', '4'},
	}
	copy(inodoRaiz.I_atime[:], fechaActual)
	copy(inodoRaiz.I_ctime[:], fechaActual)
	copy(inodoRaiz.I_mtime[:], fechaActual)

	for i := 0; i < 15; i++ {
		inodoRaiz.I_block[i] = -1
	}
	inodoRaiz.I_block[0] = 0

	bloqueRaiz := types.BloqueCarpeta{}
	for i := 0; i < 4; i++ {
		bloqueRaiz.B_content[i].B_inodo = -1
	}

	copy(bloqueRaiz.B_content[0].B_name[:], ".")
	bloqueRaiz.B_content[0].B_inodo = 0

	copy(bloqueRaiz.B_content[1].B_name[:], "..")
	bloqueRaiz.B_content[1].B_inodo = 0

	copy(bloqueRaiz.B_content[2].B_name[:], "users.txt")
	bloqueRaiz.B_content[2].B_inodo = 1

	contenidoUsers := "1,G,root\n1,U,root,root,123\n"
	inodoUsers := types.Inodo{
		I_uid:  1,
		I_gid:  1,
		I_size: int64(len(contenidoUsers)),
		I_type: '1',
		I_perm: [3]byte{'6', '6', '4'},
	}
	copy(inodoUsers.I_atime[:], fechaActual)
	copy(inodoUsers.I_ctime[:], fechaActual)
	copy(inodoUsers.I_mtime[:], fechaActual)

	for i := 0; i < 15; i++ {
		inodoUsers.I_block[i] = -1
	}
	inodoUsers.I_block[0] = 1

	bloqueUsers := types.BloqueArchivo{}
	copy(bloqueUsers.B_content[:], contenidoUsers)

	// Escribir Inodos
	archivo.Seek(int64(superBloque.S_inode_start), 0)
	binary.Write(archivo, binary.LittleEndian, &inodoRaiz)
	binary.Write(archivo, binary.LittleEndian, &inodoUsers)

	// Escribir Bloques (USANDO LA FUNCIÓN SEGURA)
	// Definimos el tamaño de bloque real
	superBloque.S_block_s = int32(BLOCK_SIZE)

	// Escribir Bloque 0 (Raíz) en la posición S_block_start
	utils.WriteBlock(archivo, int64(superBloque.S_block_start), &bloqueRaiz, BLOCK_SIZE)

	// Escribir Bloque 1 (users.txt) en la posición S_block_start + 1024
	offsetBloque1 := int64(superBloque.S_block_start) + int64(BLOCK_SIZE)
	archivo.Seek(offsetBloque1, 0)

	// Creamos un bloque vacío de 1024 bytes
	bufferUsersDirecto := make([]byte, BLOCK_SIZE)

	// Copiamos el string EXACTO directamente en los primeros bytes del buffer
	texto := "1,G,root\n1,U,root,root,123\n"
	copy(bufferUsersDirecto, texto)

	/* Lo escribimos al disco a la fuerza
	_, errWrite := archivo.Write(bufferUsersDirecto)
	if errWrite != nil {
		fmt.Println("[ERROR FATAL] El sistema operativo no nos deja escribir:", errWrite)
	} else {
		fmt.Println("[DEBUG] users.txt inyectado a la fuerza en pos:", offsetBloque1)
	}
	*/

	fmt.Printf("[ÉXITO] Formateo exitoso. Se escribirán %d inodos.\n", n)

	os.RemoveAll(utils.RutaBaseEspejo)
	os.MkdirAll(utils.RutaBaseEspejo, 0755)

	// Crear el users.txt inicial en el espejo
	utils.ReflejarCreacion("/users.txt", false, "1,G,root\n1,U,root,root,123\n")

	fmt.Println("[ÉXITO] Partición formateada y espejo reiniciado.")
}
