package mount

import (
	"MIAP1/utils"
	"fmt"
	"os"
)

type ParticionMontada struct {
	ID     string
	Nombre string
	Estado byte
}

type DiscoMontado struct {
	Path        string
	Numero      int
	Particiones []ParticionMontada
}

var DiscosMontados []DiscoMontado
var contadorDiscos int = 1

func MountPartition(path string, name string) {
	archivo, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Error al abrir el disco.")
		return
	}
	defer archivo.Close()

	mbr := utils.ObtenerMBR(archivo)
	particionEncontrada := false

	for i := 0; i < 4; i++ {
		nombreParticion := string(mbr.Mbr_partitions[i].Part_name[:])
		nombreLimpio := ""
		for _, b := range nombreParticion {
			if b != 0 {
				nombreLimpio += string(b)
			}
		}

		if mbr.Mbr_partitions[i].Part_status == '1' && nombreLimpio == name {
			particionEncontrada = true
			registrarMontaje(path, name)
			break
		}
	}

	if !particionEncontrada {
		fmt.Println("Error: No se encontró la partición o no es válida para montar.")
	}
}

func registrarMontaje(path string, name string) {
	indiceDisco := -1
	for i, d := range DiscosMontados {
		if d.Path == path {
			indiceDisco = i
			break
		}
	}

	if indiceDisco == -1 {
		nuevoDisco := DiscoMontado{
			Path:        path,
			Numero:      contadorDiscos,
			Particiones: []ParticionMontada{},
		}
		DiscosMontados = append(DiscosMontados, nuevoDisco)
		indiceDisco = len(DiscosMontados) - 1
		contadorDiscos++
	}

	for _, p := range DiscosMontados[indiceDisco].Particiones {
		if p.Nombre == name {
			fmt.Println("Error: La partición ya está montada.")
			return
		}
	}

	letra := byte('A' + len(DiscosMontados[indiceDisco].Particiones))
	carnet := "00"
	idGen := fmt.Sprintf("%s%d%c", carnet, DiscosMontados[indiceDisco].Numero, letra)

	nuevaParticion := ParticionMontada{
		ID:     idGen,
		Nombre: name,
		Estado: '1',
	}

	DiscosMontados[indiceDisco].Particiones = append(DiscosMontados[indiceDisco].Particiones, nuevaParticion)
	fmt.Println("Partición montada exitosamente con ID:", idGen)
}
