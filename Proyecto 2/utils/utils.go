package utils

import (
	"MIAP1/types"
	"bytes"
	"encoding/binary"
	"os"
	"strings"
	"unsafe"
)

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
