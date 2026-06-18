package disk

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"MIAP1/types"
	"MIAP1/utils"
	"os"
	"os/exec"
	"time"
)

func CreateDisk(tamanio int64, ajuste, unit, path string) {

	rand.Seed(time.Now().Unix())
	particionVacia := types.Partition{Tamanio: -1}
	mbr := types.MBR{
		Tamanio: utils.Tamanio(tamanio, unit),
		Id:      int16(rand.Intn(100)),
		Ajuste:  ajuste[0],
		Particiones: [4]types.Partition{
			particionVacia,
			particionVacia,
			particionVacia,
			particionVacia,
		},
	}
	copy(mbr.FechaCreacion[:], date())

	exec.Command("mkdir", "-p", path).Output()
	exec.Command("rmdir", path).Output()

	if _, err := os.Stat(path); err == nil {
		fmt.Println("El archivo ya existe, vuelva a intentarlo....")
		return
	}

	file, err := os.Create(path)
	if err != nil {
		fmt.Println("Ocurrio un error al crear el archivo")
		return
	}

	defer file.Close()

	// Reservamos memeria, guardamos un 0 en el buffer y escribimos eso en el disco.
	buffer := bytes.NewBuffer([]byte{}) // reservamos espacio en memoria
	binary.Write(buffer, binary.BigEndian, uint8(0))
	file.Write(buffer.Bytes())

	// Nos posicionamos al inicio del disco binario para llenarlo de 0's.
	file.Seek(mbr.Tamanio-int64(1), 0)
	file.Write(buffer.Bytes())

	// Nos posicionamos al inicio del discoEscribir el MBR
	file.Seek(0, 0)
	buffer.Reset()
	binary.Write(buffer, binary.BigEndian, &mbr)
	file.Write(buffer.Bytes())

	fmt.Println("Disco creado exitosamente")

}

func DeleteDisk(paht string) {
	fmt.Println("Quieres eliminar el disco? , (si/no)")
	mess := "no"
	fmt.Scanln(&mess)

	if mess == "si" {
		err := os.Remove(paht)
		if err != nil {
			fmt.Println("Error al eliminar el disco")
		} else {
			fmt.Println("Disco eliminado exitosamente")
		}
	} else {
		fmt.Println("Operacion cancelada")
	}
}

func date() string {
	time := time.Now()
	fecha := fmt.Sprintf("%02d-%02d-%d %02d:%02d:%02d", time.Day(), time.Month(), time.Year(), time.Hour(), time.Minute(), time.Second())

	return fecha
}
