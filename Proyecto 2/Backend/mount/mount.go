package mount

import (
	"MIAP1/global"
	"MIAP1/utils"
	"fmt"
	"os"
	"path/filepath"
)

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
	for i, d := range global.DiscosMontados {
		if d.Path == path {
			indiceDisco = i
			break
		}
	}

	if indiceDisco == -1 {
		nuevoDisco := global.DiscoMontado{
			Path:        path,
			Numero:      contadorDiscos,
			Particiones: []global.ParticionMontada{},
		}
		global.DiscosMontados = append(global.DiscosMontados, nuevoDisco)
		indiceDisco = len(global.DiscosMontados) - 1
		contadorDiscos++
	}

	for _, p := range global.DiscosMontados[indiceDisco].Particiones {
		if p.Nombre == name {
			fmt.Println("[ERROR] La partición ya está montada.")
			return
		}
	}

	letra := byte('A' + len(global.DiscosMontados[indiceDisco].Particiones))
	carnet := "00"
	idGen := fmt.Sprintf("%s%d%c", carnet, global.DiscosMontados[indiceDisco].Numero, letra)

	nuevaParticion := global.ParticionMontada{
		ID:     idGen,
		Nombre: name,
		Estado: '1',
	}

	global.DiscosMontados[indiceDisco].Particiones = append(global.DiscosMontados[indiceDisco].Particiones, nuevaParticion)
	fmt.Println("Partición montada exitosamente con ID:", idGen)

	fmt.Println("\n========== PARTICIONES MONTADAS ==========")
	fmt.Printf("%-15s | %-15s | %-5s\n", "DISCO", "PARTICIÓN", "ID")
	fmt.Println("------------------------------------------")

	for _, disco := range global.DiscosMontados {
		nombreDisco := filepath.Base(disco.Path)
		for _, part := range disco.Particiones {
			fmt.Printf("%-15s | %-15s | %-5s\n", nombreDisco, part.Nombre, part.ID)
		}
	}
	fmt.Println("==========================================\n")

}

func Unmount(id string) {
	encontrado := false

	for i, disco := range global.DiscosMontados {

		for j, particion := range disco.Particiones {

			if particion.ID == id {
				global.DiscosMontados[i].Particiones = append(global.DiscosMontados[i].Particiones[:j], global.DiscosMontados[i].Particiones[j+1:]...)

				encontrado = true
				fmt.Printf("[ÉXITO] Partición con ID %s desmontada correctamente.\n", id)
				break
			}
		}

		if encontrado {
			if len(global.DiscosMontados[i].Particiones) == 0 {
				global.DiscosMontados = append(global.DiscosMontados[:i], global.DiscosMontados[i+1:]...)
			}
			return
		}
	}

	if !encontrado {
		fmt.Printf("[ERROR] No se encontró ninguna partición montada con el ID: %s\n", id)
	}
}
