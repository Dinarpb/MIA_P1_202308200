package report

import (
	"MIAP1/utils"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func MBR(rutaReporte, rutaDisco string) {
	// Asumamos que ya esta montada la particion
	exec.Command("mkdir", "-p", rutaReporte).Output()
	exec.Command("rmdir", rutaReporte).Output()

	archivo, err := os.OpenFile(rutaDisco, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Ocurrio un error al crear el archivo")
		return
	}

	defer archivo.Close()

	if archivo == nil {
		fmt.Println("Disco no exite")
		return
	}

	mbr := utils.ObtenerMBR(archivo)

	grafo := "digraph Disk {\n"
	grafo += "graph [ratio = fill]; \n "
	grafo += "node  [label=\"N\", fontsize=15, shape=plaintext]; \n "
	grafo += "graph [bb=\"0,0,352,154\"]; \n"
	grafo += "arset [label=< \n"
	grafo += "<TABLE ALIGN=\"LEFT\"> \n"
	grafo += "<TR> \n"
	grafo += "<TD> MBR </TD> \n"

	for i := 0; i < 4; i++ {
		porcentaje := float64((float64(mbr.Mbr_partitions[i].Tamanio) * 100) / float64(mbr.Mbr_tamano))
		porcentajeConvertido := fmt.Sprintf("%.2f", porcentaje)
		tipo := string(mbr.Mbr_partitions[i].Tipo)
		println(mbr.Mbr_partitions[i].Inicio != 0)
		if mbr.Mbr_partitions[i].Inicio != 0 {

			grafo = grafo + "<TD> <TABLE BORDER=\"0\"> \n"
			grafo = grafo + "<TR><TD>" + cadenaLimpia(mbr.Mbr_partitions[i].Nombre[:]) + "</TD></TR> \n"
			grafo = grafo + "<TR><TD>" + tipo + "</TD></TR> \n"
			grafo = grafo + "<TR><TD>" + porcentajeConvertido + "%</TD></TR> \n"
			grafo = grafo + "</TABLE> </TD>; \n"

		}
	}

	espacioUsado := int64(0)

	for i := 0; i < 4; i++ {
		if mbr.Mbr_partitions[i].Inicio != 0 {
			espacioUsado += int64(mbr.Mbr_partitions[i].Tamanio)
		}
	}

	espacioLibre := int64(mbr.Mbr_tamano) - espacioUsado

	if float64(mbr.Mbr_tamano) > 0 {
		porcentajeLibre := (float64(espacioLibre) * 100) / float64(mbr.Mbr_tamano)
		porcentajeConvertido := fmt.Sprintf("%.3f", porcentajeLibre)
		grafo = grafo + "<TD> <TABLE BORDER=\"0\"> \n"
		grafo = grafo + "<TR><TD> LIBRE </TD></TR> \n"
		grafo = grafo + "<TR><TD>" + porcentajeConvertido + "%</TD></TR> \n"
		grafo = grafo + "</TABLE> </TD>; \n"
	}

	grafo += "</TR> \n"
	grafo += "</TABLE> \n"
	grafo += ">]; \n"
	grafo += "} \n"

	rutaTxt, rutaPng := obteneRutaReporte(rutaReporte)
	reporte, _ := os.Create(rutaTxt)
	reporte.WriteString(grafo)
	reporte.Close()
	exec.Command("dot", rutaTxt, "-Tpng", "-o", rutaPng).Output()
	fmt.Println("Reporte MBR generado exitosamente")

}

func obteneRutaReporte(rutaR string) (string, string) {
	ruta := strings.ReplaceAll(rutaR, "\"", "")

	dir := filepath.Dir(ruta)
	nombre := strings.TrimSuffix(filepath.Base(ruta), filepath.Ext(ruta))

	rutaTxt := filepath.Join(dir, nombre+".txt")
	rutaPng := filepath.Join(dir, nombre+".png")

	return rutaTxt, rutaPng
}

func cadenaLimpia(cadena []byte) string {

	cadenaLimpia := ""
	for i := 0; i < len(cadena); i++ {

		if cadena[i] == 0 {
			cadenaLimpia += ""
		} else {

			cadenaLimpia += string(cadena[i])
		}
	}

	return cadenaLimpia
}
