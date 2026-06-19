package disk

import (
	"MIAP1/types"
	"encoding/binary"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func CreateDisk(size int64, fit string, unit string, path string) {
	var totalSize int64
	if unit == "m" {
		totalSize = size * 1024 * 1024
	} else {
		totalSize = size * 1024
	}

	if _, err := os.Stat(path); err == nil {
		fmt.Println("Error: El archivo ya existe.")
		return
	}

	err := os.MkdirAll(filepath.Dir(path), os.ModePerm)
	if err != nil {
		fmt.Println("Error al crear carpetas base:", err)
		return
	}

	file, err := os.Create(path)
	if err != nil {
		fmt.Println("Error al crear el archivo:", err)
		return
	}
	defer file.Close()

	buffer := make([]byte, 1024)
	limite := totalSize / 1024
	for i := int64(0); i < limite; i++ {
		file.Write(buffer)
	}

	file.Seek(0, 0)

	var mbr types.MBR
	mbr.Mbr_tamano = totalSize
	fecha := time.Now().Format("2006-01-02 15:04:05")
	copy(mbr.Mbr_fecha_creacion[:], fecha)
	mbr.Mbr_dsk_signature = rand.Int31()
	mbr.Dsk_fit = fit[0]

	for i := 0; i < 4; i++ {
		mbr.Mbr_partitions[i].Part_status = '0'
		mbr.Mbr_partitions[i].Part_type = '0'
		mbr.Mbr_partitions[i].Part_fit = '0'
		mbr.Mbr_partitions[i].Part_start = -1
		mbr.Mbr_partitions[i].Part_s = 0
		mbr.Mbr_partitions[i].Part_correlative = 0
	}

	err = binary.Write(file, binary.BigEndian, &mbr)
	if err != nil {
		fmt.Println("Error al escribir el MBR:", err)
		return
	}

	fmt.Println("Disco creado exitosamente en:", path)
}

func DeleteDisk(path string) {
	if _, err := os.Stat(path); err != nil {
		fmt.Println("Error: El disco no existe en la ruta especificada.")
		return
	}

	fmt.Println("¿Seguro que desea eliminar el disco? (si/no)")
	mess := ""
	fmt.Scanln(&mess)

	if strings.ToLower(mess) == "si" {
		err := os.Remove(path)
		if err != nil {
			fmt.Println("Error al eliminar el disco:", err)
		} else {
			fmt.Println("Disco eliminado exitosamente.")
		}
	} else {
		fmt.Println("Operación cancelada.")
	}
}
