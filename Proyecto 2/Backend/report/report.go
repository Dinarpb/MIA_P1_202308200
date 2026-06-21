package report

import (
	"MIAP1/global"
	"MIAP1/types"
	"MIAP1/utils"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func MBR(rutaReporte, rutaDisco string) {
	// asumimos que ya esta montada la particion
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
		porcentaje := float64((float64(mbr.Mbr_partitions[i].Part_s) * 100) / float64(mbr.Mbr_tamano))
		porcentajeConvertido := fmt.Sprintf("%.2f", porcentaje)
		tipo := string(mbr.Mbr_partitions[i].Part_type)
		println(mbr.Mbr_partitions[i].Part_start != 0)
		if mbr.Mbr_partitions[i].Part_start != 0 {

			grafo = grafo + "<TD> <TABLE BORDER=\"0\"> \n"
			grafo = grafo + "<TR><TD>" + cadenaLimpia(mbr.Mbr_partitions[i].Part_name[:]) + "</TD></TR> \n"
			grafo = grafo + "<TR><TD>" + tipo + "</TD></TR> \n"
			grafo = grafo + "<TR><TD>" + porcentajeConvertido + "%</TD></TR> \n"
			grafo = grafo + "</TABLE> </TD>; \n"

		}
	}

	espacioUsado := int64(0)

	for i := 0; i < 4; i++ {
		if mbr.Mbr_partitions[i].Part_start != 0 {
			espacioUsado += int64(mbr.Mbr_partitions[i].Part_s)
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

	rutaTxt, rutaFinal, formato := obteneRutaReporte(rutaReporte)
	prepararDirectorio(rutaTxt)
	prepararDirectorio(rutaFinal)
	reporte, _ := os.Create(rutaTxt)
	reporte.WriteString(grafo)
	reporte.Close()

	salida, errCmd := exec.Command("dot", "-T"+formato, rutaTxt, "-o", rutaFinal).CombinedOutput()
	if errCmd != nil {
		fmt.Println("[ERROR] Falló Graphviz al generar la imagen:")
		fmt.Println(string(salida))
	} else {
		fmt.Printf("[ÉXITO] Reporte MBR generado exitosamente en: %s\n", rutaFinal)
	}

}

func obteneRutaReporte(rutaR string) (string, string, string) {
	ruta := strings.ReplaceAll(rutaR, "\"", "")
	dir := filepath.Dir(ruta)

	ext := strings.ToLower(filepath.Ext(ruta))
	if ext == "" {
		ext = ".jpg"
	}

	nombre := strings.TrimSuffix(filepath.Base(ruta), filepath.Ext(ruta))
	rutaTxt := filepath.Join(dir, nombre+".txt")
	rutaFinal := filepath.Join(dir, nombre+ext)

	formato := strings.TrimPrefix(ext, ".")

	return rutaTxt, rutaFinal, formato
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

func GenerarReporte(name string, path string, id string, pathFileLs string) {
	var rutaDisco string
	encontrado := false

	for _, disco := range global.DiscosMontados {
		for _, particion := range disco.Particiones {
			if particion.ID == id {
				rutaDisco = disco.Path
				encontrado = true
				break
			}
		}
		if encontrado {
			break
		}
	}

	if !encontrado {
		fmt.Println("Error: No se encontró la partición montada con el ID especificado para el reporte.")
		return
	}

	switch name {
	case "mbr":
		ReporteMBR(path, rutaDisco)
	case "disk":
		ReporteDisk(path, rutaDisco)
	case "inode":
		ReporteInode(path, rutaDisco, id)
	case "block":
		ReporteBlock(path, rutaDisco, id)
	case "bm_inode":
		ReporteBitmap(path, rutaDisco, id, "inode")
	case "bm_block":
		ReporteBitmap(path, rutaDisco, id, "block")
	case "sb":
		ReporteSB(path, rutaDisco, id)
	case "file":
		ReporteFile(path, rutaDisco, id, pathFileLs)
	case "ls":
		ReporteLs(path, rutaDisco, id, pathFileLs)
	case "tree":
		ReporteTree(path, rutaDisco, id)
	}
}

func buscarPartStart(archivo *os.File, rutaDisco, id string) int64 {
	nombreParticion := ""
	for _, disco := range global.DiscosMontados {
		if disco.Path == rutaDisco {
			for _, p := range disco.Particiones {
				if p.ID == id {
					nombreParticion = p.Nombre
					break
				}
			}
		}
	}

	mbr := utils.ObtenerMBR(archivo)
	var partStart int64 = -1

	for i := 0; i < 4; i++ {
		if mbr.Mbr_partitions[i].Part_status == '1' && cadenaLimpia(mbr.Mbr_partitions[i].Part_name[:]) == nombreParticion {
			partStart = mbr.Mbr_partitions[i].Part_start
			break
		} else if strings.ToLower(string(mbr.Mbr_partitions[i].Part_type)) == "e" {
			nextEBR := mbr.Mbr_partitions[i].Part_start
			for nextEBR != -1 && nextEBR != 0 {
				archivo.Seek(nextEBR, 0)
				var ebr types.EBR
				binary.Read(archivo, binary.LittleEndian, &ebr)
				if cadenaLimpia(ebr.Part_name[:]) == nombreParticion {
					partStart = ebr.Part_start + int64(binary.Size(ebr))
					break
				}
				nextEBR = ebr.Part_next
			}
		}
	}

	return partStart
}

func ReporteMBR(rutaReporte, rutaDisco string) {
	exec.Command("mkdir", "-p", filepath.Dir(rutaReporte)).Output()

	archivo, err := os.OpenFile(rutaDisco, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Error: No se pudo abrir el disco para el reporte MBR.")
		return
	}
	defer archivo.Close()

	mbr := utils.ObtenerMBR(archivo)

	grafo := "digraph G {\n"
	grafo += "  node [shape=plaintext];\n"
	grafo += "  tabla [label=<\n"
	grafo += "    <table border=\"1\" cellborder=\"1\" cellspacing=\"0\">\n"

	grafo += "      <tr><td colspan=\"2\" bgcolor=\"#4b225c\"><font color=\"white\"><b>REPORTE DE MBR</b></font></td></tr>\n"
	grafo += fmt.Sprintf("      <tr><td>mbr_tamano</td><td>%d</td></tr>\n", mbr.Mbr_tamano)
	fecha := strings.Trim(string(mbr.Mbr_fecha_creacion[:]), "\x00")
	grafo += fmt.Sprintf("      <tr><td>mbr_fecha_creacion</td><td>%s</td></tr>\n", fecha)
	grafo += fmt.Sprintf("      <tr><td>mbr_disk_signature</td><td>%d</td></tr>\n", mbr.Mbr_dsk_signature)
	grafo += fmt.Sprintf("      <tr><td>Dsk_fit</td><td>%c</td></tr>\n", mbr.Dsk_fit)

	for i := 0; i < 4; i++ {
		if mbr.Mbr_partitions[i].Part_start != 0 {
			grafo += "      <tr><td colspan=\"2\" bgcolor=\"#4b225c\"><font color=\"white\"><b>Particion</b></font></td></tr>\n"
			grafo += fmt.Sprintf("      <tr><td>part_status</td><td>%c</td></tr>\n", mbr.Mbr_partitions[i].Part_status)
			grafo += fmt.Sprintf("      <tr><td>part_type</td><td>%c</td></tr>\n", mbr.Mbr_partitions[i].Part_type)
			grafo += fmt.Sprintf("      <tr><td>part_fit</td><td>%c</td></tr>\n", mbr.Mbr_partitions[i].Part_fit)
			grafo += fmt.Sprintf("      <tr><td>part_start</td><td>%d</td></tr>\n", mbr.Mbr_partitions[i].Part_start)
			grafo += fmt.Sprintf("      <tr><td>part_size</td><td>%d</td></tr>\n", mbr.Mbr_partitions[i].Part_s)
			grafo += fmt.Sprintf("      <tr><td>part_name</td><td>%s</td></tr>\n", cadenaLimpia(mbr.Mbr_partitions[i].Part_name[:]))
			grafo += fmt.Sprintf("      <tr><td>part_correlative</td><td>%d</td></tr>\n", mbr.Mbr_partitions[i].Part_correlative)
			grafo += fmt.Sprintf("      <tr><td>part_id</td><td>%s</td></tr>\n", cadenaLimpia(mbr.Mbr_partitions[i].Part_id[:]))

			tipo := strings.ToLower(string(mbr.Mbr_partitions[i].Part_type))
			if tipo == "e" {
				nextEBR := mbr.Mbr_partitions[i].Part_start
				for nextEBR != -1 {
					archivo.Seek(int64(nextEBR), 0)
					var ebr types.EBR
					binary.Read(archivo, binary.LittleEndian, &ebr)

					if ebr.Part_s > 0 {
						grafo += "      <tr><td colspan=\"2\" bgcolor=\"#f47c7c\"><font color=\"black\"><b>Particion Logica</b></font></td></tr>\n"
						grafo += fmt.Sprintf("      <tr><td>part_status</td><td>%c</td></tr>\n", ebr.Part_mount)
						grafo += fmt.Sprintf("      <tr><td>part_next</td><td>%d</td></tr>\n", ebr.Part_next)
						grafo += fmt.Sprintf("      <tr><td>part_fit</td><td>%c</td></tr>\n", ebr.Part_fit)
						grafo += fmt.Sprintf("      <tr><td>part_start</td><td>%d</td></tr>\n", ebr.Part_start)
						grafo += fmt.Sprintf("      <tr><td>part_size</td><td>%d</td></tr>\n", ebr.Part_s)
						grafo += fmt.Sprintf("      <tr><td>part_name</td><td>%s</td></tr>\n", cadenaLimpia(ebr.Part_name[:]))
					}

					if ebr.Part_next == -1 || ebr.Part_next == 0 {
						break
					}
					nextEBR = ebr.Part_next
				}
			}
		}
	}

	grafo += "    </table>\n"
	grafo += "  >];\n"
	grafo += "}\n"

	rutaTxt, rutaFinal, formato := obteneRutaReporte(rutaReporte)
	prepararDirectorio(rutaTxt)
	prepararDirectorio(rutaFinal)
	reporte, _ := os.Create(rutaTxt)
	reporte.WriteString(grafo)
	reporte.Close()

	salida, errCmd := exec.Command("dot", "-T"+formato, rutaTxt, "-o", rutaFinal).CombinedOutput()
	if errCmd != nil {
		fmt.Println("[ERROR] Falló Graphviz al generar la imagen:")
		fmt.Println(string(salida))
	} else {
		fmt.Printf("[ÉXITO] Reporte MBR generado exitosamente en: %s\n", rutaFinal)
	}
}

func ReporteDisk(rutaReporte, rutaDisco string) {
	exec.Command("mkdir", "-p", filepath.Dir(rutaReporte)).Output()

	archivo, err := os.OpenFile(rutaDisco, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Error: No se pudo abrir el disco para el reporte DISK.")
		return
	}
	defer archivo.Close()

	mbr := utils.ObtenerMBR(archivo)

	var particiones []types.Partition
	for i := 0; i < 4; i++ {
		if mbr.Mbr_partitions[i].Part_status == '1' || (mbr.Mbr_partitions[i].Part_status == '0' && mbr.Mbr_partitions[i].Part_s > 0) {
			particiones = append(particiones, mbr.Mbr_partitions[i])
		}
	}
	sort.Slice(particiones, func(i, j int) bool {
		return particiones[i].Part_start < particiones[j].Part_start
	})

	grafo := "digraph G {\n"
	grafo += "  rankdir=LR;\n"
	grafo += "  node [shape=plaintext];\n"
	grafo += "  tabla [label=<\n"
	grafo += "    <table border=\"0\" cellborder=\"1\" cellspacing=\"0\">\n"
	grafo += "      <tr>\n"
	grafo += "        <td>MBR</td>\n"

	posicionActual := int64(binary.Size(mbr))

	for _, p := range particiones {
		if p.Part_start > posicionActual {
			libre := p.Part_start - posicionActual
			porcentaje := (float64(libre) / float64(mbr.Mbr_tamano)) * 100
			grafo += fmt.Sprintf("        <td>Libre<br/>%.2f%% del disco</td>\n", porcentaje)
		}

		tipo := strings.ToLower(string(p.Part_type))
		porcentajePart := (float64(p.Part_s) / float64(mbr.Mbr_tamano)) * 100

		if tipo == "p" {
			grafo += fmt.Sprintf("        <td>Primaria<br/>%.2f%% del disco</td>\n", porcentajePart)
		} else if tipo == "e" {
			grafo += "        <td>\n"
			grafo += "          <table border=\"0\" cellborder=\"1\" cellspacing=\"0\">\n"

			cantidadCeldas := contarCeldasLogicas(archivo, p)
			grafo += fmt.Sprintf("            <tr><td colspan=\"%d\">Extendida</td></tr>\n", cantidadCeldas)

			grafo += "            <tr>\n"

			nextEBR := p.Part_start
			posicionExtendida := p.Part_start

			for nextEBR != -1 && nextEBR != 0 {
				archivo.Seek(nextEBR, 0)
				var ebr types.EBR
				err := binary.Read(archivo, binary.LittleEndian, &ebr)
				if err != nil {
					break
				}

				if ebr.Part_start > posicionExtendida {
					libreExt := ebr.Part_start - posicionExtendida
					porcExt := (float64(libreExt) / float64(mbr.Mbr_tamano)) * 100
					grafo += fmt.Sprintf("              <td>Libre<br/>%.2f%% del disco</td>\n", porcExt)
				}

				if ebr.Part_mount == '1' || (ebr.Part_mount == '0' && ebr.Part_s > 0) {
					grafo += "              <td>EBR</td>\n"
					porcLog := (float64(ebr.Part_s) / float64(mbr.Mbr_tamano)) * 100
					grafo += fmt.Sprintf("              <td>Lógica<br/>%.2f%% del disco</td>\n", porcLog)
					posicionExtendida = ebr.Part_start + ebr.Part_s
				} else {
					posicionExtendida = ebr.Part_start + int64(binary.Size(ebr))
				}

				if ebr.Part_next == -1 {
					break
				}
				nextEBR = ebr.Part_next
			}

			if posicionExtendida < (p.Part_start + p.Part_s) {
				libreExt := (p.Part_start + p.Part_s) - posicionExtendida
				porcExt := (float64(libreExt) / float64(mbr.Mbr_tamano)) * 100
				grafo += fmt.Sprintf("              <td>Libre<br/>%.2f%% del disco</td>\n", porcExt)
			}

			grafo += "            </tr>\n"
			grafo += "          </table>\n"
			grafo += "        </td>\n"
		}

		posicionActual = p.Part_start + p.Part_s
	}

	if posicionActual < mbr.Mbr_tamano {
		libre := mbr.Mbr_tamano - posicionActual
		porcentaje := (float64(libre) / float64(mbr.Mbr_tamano)) * 100
		grafo += fmt.Sprintf("        <td>Libre<br/>%.2f%% del disco</td>\n", porcentaje)
	}

	grafo += "      </tr>\n"
	grafo += "    </table>\n"
	grafo += "  >];\n"
	grafo += "}\n"

	rutaTxt, rutaFinal, formato := obteneRutaReporte(rutaReporte)
	prepararDirectorio(rutaTxt)
	prepararDirectorio(rutaFinal)
	reporte, _ := os.Create(rutaTxt)
	reporte.WriteString(grafo)
	reporte.Close()

	salida, errCmd := exec.Command("dot", "-T"+formato, rutaTxt, "-o", rutaFinal).CombinedOutput()
	if errCmd != nil {
		fmt.Println("[ERROR] Falló Graphviz al generar la imagen:")
		fmt.Println(string(salida))
	} else {
		fmt.Printf("[ÉXITO] Reporte DISK generado exitosamente en: %s\n", rutaFinal)
	}
}

func ReporteInode(rutaReporte, rutaDisco, id string) {
	exec.Command("mkdir", "-p", filepath.Dir(rutaReporte)).Output()

	archivo, err := os.OpenFile(rutaDisco, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Error: No se pudo abrir el disco para el reporte INODE.")
		return
	}
	defer archivo.Close()

	partStart := buscarPartStart(archivo, rutaDisco, id)
	if partStart == -1 {
		fmt.Println("Error: No se encontró el inicio de la partición.")
		return
	}

	var sb types.SuperBloque
	archivo.Seek(partStart, 0)
	binary.Read(archivo, binary.LittleEndian, &sb)

	bitmapInodos := make([]byte, sb.S_inodes_count)
	archivo.Seek(int64(sb.S_bm_inode_start), 0)
	binary.Read(archivo, binary.LittleEndian, &bitmapInodos)

	grafo := "digraph G {\n"
	grafo += "  rankdir=LR;\n"
	grafo += "  node [shape=plaintext];\n"

	var inodoAnterior int = -1

	for i := 0; i < int(sb.S_inodes_count); i++ {
		if bitmapInodos[i] == '1' {
			var inodo types.Inodo
			archivo.Seek(int64(sb.S_inode_start)+int64(i)*int64(sb.S_inode_s), 0)
			binary.Read(archivo, binary.LittleEndian, &inodo)

			grafo += fmt.Sprintf("  inodo_%d [label=<\n", i)
			grafo += "    <table border=\"1\" cellborder=\"0\" cellspacing=\"0\">\n"
			grafo += fmt.Sprintf("      <tr><td colspan=\"2\" align=\"center\"><b>Inodo %d</b></td></tr>\n", i)
			grafo += fmt.Sprintf("      <tr><td align=\"left\">i_uid</td><td align=\"left\">%d</td></tr>\n", inodo.I_uid)
			grafo += fmt.Sprintf("      <tr><td align=\"left\">i_gid</td><td align=\"left\">%d</td></tr>\n", inodo.I_gid)
			grafo += fmt.Sprintf("      <tr><td align=\"left\">i_size</td><td align=\"left\">%d</td></tr>\n", inodo.I_size)

			atime := strings.Trim(string(inodo.I_atime[:]), "\x00")
			ctime := strings.Trim(string(inodo.I_ctime[:]), "\x00")
			mtime := strings.Trim(string(inodo.I_mtime[:]), "\x00")

			grafo += fmt.Sprintf("      <tr><td align=\"left\">i_atime</td><td align=\"left\">%s</td></tr>\n", atime)
			grafo += fmt.Sprintf("      <tr><td align=\"left\">i_ctime</td><td align=\"left\">%s</td></tr>\n", ctime)
			grafo += fmt.Sprintf("      <tr><td align=\"left\">i_mtime</td><td align=\"left\">%s</td></tr>\n", mtime)

			for j := 0; j < 15; j++ {
				grafo += fmt.Sprintf("      <tr><td align=\"left\">i_block_%d</td><td align=\"left\">%d</td></tr>\n", j+1, inodo.I_block[j])
			}

			grafo += fmt.Sprintf("      <tr><td align=\"left\">i_type</td><td align=\"left\">%c</td></tr>\n", inodo.I_type)
			grafo += fmt.Sprintf("      <tr><td align=\"left\">i_perm</td><td align=\"left\">%d</td></tr>\n", inodo.I_perm)
			grafo += "    </table>\n  >];\n"

			if inodoAnterior != -1 {
				grafo += fmt.Sprintf("  inodo_%d -> inodo_%d [color=\"#4287f5\"];\n", inodoAnterior, i)
			}
			inodoAnterior = i
		}
	}

	grafo += "}\n"

	rutaTxt, rutaFinal, formato := obteneRutaReporte(rutaReporte)
	prepararDirectorio(rutaTxt)
	prepararDirectorio(rutaFinal)
	reporte, _ := os.Create(rutaTxt)
	reporte.WriteString(grafo)
	reporte.Close()

	salida, errCmd := exec.Command("dot", "-T"+formato, rutaTxt, "-o", rutaFinal).CombinedOutput()
	if errCmd != nil {
		fmt.Println("[ERROR] Falló Graphviz al generar la imagen:")
		fmt.Println(string(salida))
	} else {
		fmt.Printf("[ÉXITO] Reporte INODE generado exitosamente en: %s\n", rutaFinal)
	}
}

func ReporteBlock(rutaReporte, rutaDisco, id string) {
	exec.Command("mkdir", "-p", filepath.Dir(rutaReporte)).Output()

	archivo, err := os.OpenFile(rutaDisco, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Error: No se pudo abrir el disco para el reporte BLOCK.")
		return
	}
	defer archivo.Close()

	partStart := buscarPartStart(archivo, rutaDisco, id)
	if partStart == -1 {
		fmt.Println("Error: No se encontró el inicio de la partición.")
		return
	}

	var sb types.SuperBloque
	archivo.Seek(partStart, 0)
	binary.Read(archivo, binary.LittleEndian, &sb)

	bitmapInodos := make([]byte, sb.S_inodes_count)
	archivo.Seek(int64(sb.S_bm_inode_start), 0)
	binary.Read(archivo, binary.LittleEndian, &bitmapInodos)

	bitmapBloques := make([]byte, sb.S_blocks_count)
	archivo.Seek(int64(sb.S_bm_block_start), 0)
	binary.Read(archivo, binary.LittleEndian, &bitmapBloques)

	tipoBloque := make(map[int]int)
	for i := 0; i < int(sb.S_inodes_count); i++ {
		if bitmapInodos[i] == '1' {
			var inodo types.Inodo
			archivo.Seek(int64(sb.S_inode_start)+int64(i)*int64(sb.S_inode_s), 0)
			binary.Read(archivo, binary.LittleEndian, &inodo)

			for j := 0; j < 15; j++ {
				if inodo.I_block[j] != -1 {
					if j < 12 {
						if inodo.I_type == '0' {
							tipoBloque[int(inodo.I_block[j])] = 0
						} else {
							tipoBloque[int(inodo.I_block[j])] = 1
						}
					} else {
						tipoBloque[int(inodo.I_block[j])] = 2
					}
				}
			}
		}
	}

	grafo := "digraph G {\n"
	grafo += "  rankdir=LR;\n"
	grafo += "  node [shape=plaintext];\n"

	var bloqueAnterior int = -1

	for i := 0; i < int(sb.S_blocks_count); i++ {
		if bitmapBloques[i] == '1' {
			grafo += fmt.Sprintf("  bloque_%d [label=<\n", i)
			grafo += "    <table border=\"1\" cellborder=\"0\" cellspacing=\"0\">\n"

			tipo, existe := tipoBloque[i]
			if !existe {
				tipo = 0
			}

			archivo.Seek(int64(sb.S_block_start)+int64(i)*int64(sb.S_block_s), 0)

			if tipo == 0 {
				var bc types.BloqueCarpeta
				binary.Read(archivo, binary.LittleEndian, &bc)
				grafo += fmt.Sprintf("      <tr><td colspan=\"2\" align=\"center\"><b>Bloque Carpeta %d</b></td></tr>\n", i)
				grafo += "      <tr><td align=\"left\">b_name</td><td align=\"left\">b_inodo</td></tr>\n"
				for _, c := range bc.B_content {
					nombre := limpiarCadenaHTML(c.B_name[:])
					nombre = strings.ReplaceAll(nombre, "<", "&lt;")
					nombre = strings.ReplaceAll(nombre, ">", "&gt;")

					if nombre == "" {
						nombre = "-"
					}
					grafo += fmt.Sprintf("      <tr><td align=\"left\">%s</td><td align=\"left\">%d</td></tr>\n", nombre, c.B_inodo)
				}
			} else if tipo == 1 {
				var ba types.BloqueArchivo
				binary.Read(archivo, binary.LittleEndian, &ba)

				contenido := limpiarCadenaHTML(ba.B_content[:])
				contenido = strings.ReplaceAll(contenido, "<", "&lt;")
				contenido = strings.ReplaceAll(contenido, ">", "&gt;")
				contenido = strings.ReplaceAll(contenido, "\n", "<br/>")

				grafo += fmt.Sprintf("      <tr><td align=\"center\"><b>Bloque Archivo %d</b></td></tr>\n", i)
				grafo += fmt.Sprintf("      <tr><td align=\"left\">%s</td></tr>\n", contenido)
			} else if tipo == 2 {
				var bap types.BloqueApuntadores
				binary.Read(archivo, binary.LittleEndian, &bap)
				grafo += fmt.Sprintf("      <tr><td align=\"center\"><b>Bloque Apuntadores %d</b></td></tr>\n", i)
				punterosStr := ""
				for idx, p := range bap.B_pointers {
					punterosStr += fmt.Sprintf("%d, ", p)
					if (idx+1)%4 == 0 {
						punterosStr += "<br/>"
					}
				}
				grafo += fmt.Sprintf("      <tr><td align=\"left\">%s</td></tr>\n", strings.TrimSuffix(strings.TrimSuffix(punterosStr, "<br/>"), ", "))
			}

			grafo += "    </table>\n  >];\n"

			if bloqueAnterior != -1 {
				grafo += fmt.Sprintf("  bloque_%d -> bloque_%d [color=\"#81a9ad\"];\n", bloqueAnterior, i)
			}
			bloqueAnterior = i
		}
	}

	grafo += "}\n"

	rutaTxt, rutaFinal, formato := obteneRutaReporte(rutaReporte)
	prepararDirectorio(rutaTxt)
	prepararDirectorio(rutaFinal)
	reporte, _ := os.Create(rutaTxt)
	reporte.WriteString(grafo)
	reporte.Close()

	salida, errCmd := exec.Command("dot", "-T"+formato, rutaTxt, "-o", rutaFinal).CombinedOutput()
	if errCmd != nil {
		fmt.Println("[ERROR] Falló Graphviz al generar la imagen BLOCK:")
		fmt.Println(string(salida))
	} else {
		fmt.Printf("[ÉXITO] Reporte BLOCK generado exitosamente en: %s\n", rutaFinal)
	}
}

func ReporteBitmap(rutaReporte, rutaDisco, id, tipo string) {
	exec.Command("mkdir", "-p", filepath.Dir(rutaReporte)).Output()

	archivo, err := os.OpenFile(rutaDisco, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Error: No se pudo abrir el disco para el reporte de Bitmap.")
		return
	}
	defer archivo.Close()

	partStart := buscarPartStart(archivo, rutaDisco, id)
	if partStart == -1 {
		fmt.Println("Error: No se encontró el inicio de la partición.")
		return
	}

	var sb types.SuperBloque
	archivo.Seek(partStart, 0)
	binary.Read(archivo, binary.LittleEndian, &sb)

	var startByte int64
	var totalBits int

	if tipo == "inode" {
		startByte = int64(sb.S_bm_inode_start)
		totalBits = int(sb.S_inodes_count)
	} else {
		startByte = int64(sb.S_bm_block_start)
		totalBits = int(sb.S_blocks_count)
	}

	bitmap := make([]byte, totalBits)
	archivo.Seek(startByte, 0)
	binary.Read(archivo, binary.LittleEndian, &bitmap)

	contenido := ""
	linea := 1
	for i := 0; i < totalBits; i++ {
		if i%20 == 0 {
			if i != 0 {
				contenido += "\n"
			}
			contenido += fmt.Sprintf("%-4d | ", linea)
			linea++
		}

		if bitmap[i] == '1' {
			contenido += "1 "
		} else {
			contenido += "0 "
		}
	}
	contenido += "\n"

	rutaSinExt := strings.TrimSuffix(rutaReporte, filepath.Ext(rutaReporte))
	rutaFinal := rutaSinExt + ".txt"

	err = os.WriteFile(rutaFinal, []byte(contenido), 0644)
	if err != nil {
		fmt.Println("Error: No se pudo escribir el archivo de reporte:", err)
		return
	}

	fmt.Printf("[ÉXITO] Reporte bm_%s generado exitosamente en: %s\n", tipo, rutaFinal)
}

func ReporteSB(rutaReporte, rutaDisco, id string) {
	exec.Command("mkdir", "-p", filepath.Dir(rutaReporte)).Output()

	archivo, err := os.OpenFile(rutaDisco, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Error: No se pudo abrir el disco para el reporte SB.")
		return
	}
	defer archivo.Close()

	partStart := buscarPartStart(archivo, rutaDisco, id)
	if partStart == -1 {
		fmt.Println("Error: No se encontró el inicio de la partición.")
		return
	}

	var sb types.SuperBloque
	archivo.Seek(partStart, 0)
	binary.Read(archivo, binary.LittleEndian, &sb)

	mtime := limpiarCadenaHTML(sb.S_mtime[:])
	umtime := limpiarCadenaHTML(sb.S_umtime[:])

	if mtime == "" {
		mtime = "Sin fecha"
	}
	if umtime == "" {
		umtime = "Sin fecha"
	}
	nombreDisco := filepath.Base(rutaDisco)

	grafo := "digraph G {\n"
	grafo += "  node [shape=plaintext];\n"
	grafo += "  tabla [label=<\n"
	grafo += "    <table border=\"1\" cellborder=\"1\" cellspacing=\"0\">\n"
	grafo += "      <tr><td colspan=\"2\" bgcolor=\"#126b35\"><font color=\"white\"><b>Reporte de SUPERBLOQUE</b></font></td></tr>\n"

	c1 := "white"
	c2 := "#2ecc71"

	grafo += fmt.Sprintf("      <tr><td bgcolor=\"%s\">sb_nombre_hd</td><td bgcolor=\"%s\">%s</td></tr>\n", c1, c1, nombreDisco)
	grafo += fmt.Sprintf("      <tr><td bgcolor=\"%s\">s_filesystem_type</td><td bgcolor=\"%s\">%d</td></tr>\n", c2, c2, sb.S_filesystem_type)
	grafo += fmt.Sprintf("      <tr><td bgcolor=\"%s\">s_inodes_count</td><td bgcolor=\"%s\">%d</td></tr>\n", c1, c1, sb.S_inodes_count)
	grafo += fmt.Sprintf("      <tr><td bgcolor=\"%s\">s_blocks_count</td><td bgcolor=\"%s\">%d</td></tr>\n", c2, c2, sb.S_blocks_count)
	grafo += fmt.Sprintf("      <tr><td bgcolor=\"%s\">s_free_blocks_count</td><td bgcolor=\"%s\">%d</td></tr>\n", c1, c1, sb.S_free_blocks_count)
	grafo += fmt.Sprintf("      <tr><td bgcolor=\"%s\">s_free_inodes_count</td><td bgcolor=\"%s\">%d</td></tr>\n", c2, c2, sb.S_free_inodes_count)
	grafo += fmt.Sprintf("      <tr><td bgcolor=\"%s\">s_mtime</td><td bgcolor=\"%s\">%s</td></tr>\n", c1, c1, mtime)
	grafo += fmt.Sprintf("      <tr><td bgcolor=\"%s\">s_umtime</td><td bgcolor=\"%s\">%s</td></tr>\n", c2, c2, umtime)
	grafo += fmt.Sprintf("      <tr><td bgcolor=\"%s\">s_mnt_count</td><td bgcolor=\"%s\">%d</td></tr>\n", c1, c1, sb.S_mnt_count)
	grafo += fmt.Sprintf("      <tr><td bgcolor=\"%s\">s_magic</td><td bgcolor=\"%s\">%d</td></tr>\n", c2, c2, sb.S_magic)
	grafo += fmt.Sprintf("      <tr><td bgcolor=\"%s\">s_inode_s</td><td bgcolor=\"%s\">%d</td></tr>\n", c1, c1, sb.S_inode_s)
	grafo += fmt.Sprintf("      <tr><td bgcolor=\"%s\">s_block_s</td><td bgcolor=\"%s\">%d</td></tr>\n", c2, c2, sb.S_block_s)
	grafo += fmt.Sprintf("      <tr><td bgcolor=\"%s\">s_first_ino</td><td bgcolor=\"%s\">%d</td></tr>\n", c1, c1, sb.S_first_ino)
	grafo += fmt.Sprintf("      <tr><td bgcolor=\"%s\">s_first_blo</td><td bgcolor=\"%s\">%d</td></tr>\n", c2, c2, sb.S_first_blo)
	grafo += fmt.Sprintf("      <tr><td bgcolor=\"%s\">s_bm_inode_start</td><td bgcolor=\"%s\">%d</td></tr>\n", c1, c1, sb.S_bm_inode_start)
	grafo += fmt.Sprintf("      <tr><td bgcolor=\"%s\">s_bm_block_start</td><td bgcolor=\"%s\">%d</td></tr>\n", c2, c2, sb.S_bm_block_start)
	grafo += fmt.Sprintf("      <tr><td bgcolor=\"%s\">s_inode_start</td><td bgcolor=\"%s\">%d</td></tr>\n", c1, c1, sb.S_inode_start)
	grafo += fmt.Sprintf("      <tr><td bgcolor=\"%s\">s_block_start</td><td bgcolor=\"%s\">%d</td></tr>\n", c2, c2, sb.S_block_start)

	grafo += "    </table>\n"
	grafo += "  >];\n"
	grafo += "}\n"

	rutaTxt, rutaFinal, formato := obteneRutaReporte(rutaReporte)
	prepararDirectorio(rutaTxt)
	prepararDirectorio(rutaFinal)
	reporte, _ := os.Create(rutaTxt)
	reporte.WriteString(grafo)
	reporte.Close()

	salida, errCmd := exec.Command("dot", "-T"+formato, rutaTxt, "-o", rutaFinal).CombinedOutput()
	if errCmd != nil {
		fmt.Println("[ERROR] Falló Graphviz al generar la imagen:")
		fmt.Println(string(salida))
	} else {
		fmt.Printf("[ÉXITO] Reporte SUPERBLOQUE generado exitosamente en: %s\n", rutaFinal)
	}
}

func permisosString(perm [3]byte, tipo byte) string {
	cadena := ""
	if tipo == '0' {
		cadena += "d"
	} else {
		cadena += "-"
	}

	permStr := string(perm[:])

	mapa := map[byte]string{
		'0': "---", '1': "--x", '2': "-w-", '3': "-wx",
		'4': "r--", '5': "r-x", '6': "rw-", '7': "rwx",
	}

	for i := 0; i < len(permStr) && i < 3; i++ {
		if val, ok := mapa[permStr[i]]; ok {
			cadena += val
		} else {
			cadena += "---"
		}
	}
	return cadena
}

func formatearFechaHora(fechaRaw []byte) (string, string) {
	fechaLimpia := strings.Trim(string(fechaRaw), "\x00")
	partes := strings.Split(fechaLimpia, " ")
	if len(partes) == 2 {
		return partes[0], partes[1]
	}
	return fechaLimpia, ""
}

func ReporteFile(rutaReporte, rutaDisco, id, pathFileLs string) {
	exec.Command("mkdir", "-p", filepath.Dir(rutaReporte)).Output()

	archivo, err := os.OpenFile(rutaDisco, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Error: No se pudo abrir el disco para el reporte FILE.")
		return
	}
	defer archivo.Close()

	partStart := buscarPartStart(archivo, rutaDisco, id)
	if partStart == -1 {
		fmt.Println("Error: No se encontró el inicio de la partición.")
		return
	}

	var sb types.SuperBloque
	archivo.Seek(partStart, 0)
	binary.Read(archivo, binary.LittleEndian, &sb)

	_, inodoTarget, err := utils.BuscarInodoPorRuta(archivo, sb, pathFileLs)
	if err != nil || inodoTarget.I_type == '0' {
		fmt.Println("Error: Archivo no encontrado o es una carpeta.")
		return
	}

	contenidoCompleto := ""
	for j := 0; j < 12; j++ { 
		if inodoTarget.I_block[j] != -1 {
			var ba types.BloqueArchivo
			archivo.Seek(int64(sb.S_block_start)+int64(inodoTarget.I_block[j])*int64(sb.S_block_s), 0)
			binary.Read(archivo, binary.LittleEndian, &ba)
			contenidoCompleto += strings.TrimRight(string(ba.B_content[:]), "\x00")
		}
	}

	err = os.WriteFile(rutaReporte, []byte(contenidoCompleto), 0644)
	if err != nil {
		fmt.Println("Error: No se pudo escribir el archivo de reporte FILE.")
		return
	}
	fmt.Println("Reporte FILE generado exitosamente en:", rutaReporte)
}

func ReporteLs(rutaReporte, rutaDisco, id, pathFileLs string) {
	exec.Command("mkdir", "-p", filepath.Dir(rutaReporte)).Output()

	archivo, err := os.OpenFile(rutaDisco, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Error: No se pudo abrir el disco para el reporte LS.")
		return
	}
	defer archivo.Close()

	partStart := buscarPartStart(archivo, rutaDisco, id)
	if partStart == -1 {
		fmt.Println("Error: No se encontró el inicio de la partición.")
		return
	}

	var sb types.SuperBloque
	archivo.Seek(partStart, 0)
	binary.Read(archivo, binary.LittleEndian, &sb)

	_, inodoCarpeta, err := utils.BuscarInodoPorRuta(archivo, sb, pathFileLs)
	if err != nil || inodoCarpeta.I_type == '1' {
		fmt.Println("Error: Carpeta no encontrada o es un archivo.")
		return
	}

	grafo := "digraph G {\n"
	grafo += "  node [shape=plaintext];\n"
	grafo += "  tabla [label=<\n"
	grafo += "    <table border=\"1\" cellborder=\"1\" cellspacing=\"0\">\n"
	grafo += "      <tr><td bgcolor=\"#f2f2f2\"><b>Permisos</b></td><td bgcolor=\"#f2f2f2\"><b>Owner</b></td><td bgcolor=\"#f2f2f2\"><b>Grupo</b></td><td bgcolor=\"#f2f2f2\"><b>Size (Bytes)</b></td><td bgcolor=\"#f2f2f2\"><b>Fecha</b></td><td bgcolor=\"#f2f2f2\"><b>Hora</b></td><td bgcolor=\"#f2f2f2\"><b>Tipo</b></td><td bgcolor=\"#f2f2f2\"><b>Name</b></td></tr>\n"

	for j := 0; j < 12; j++ {
		if inodoCarpeta.I_block[j] != -1 {
			var bc types.BloqueCarpeta
			archivo.Seek(int64(sb.S_block_start)+int64(inodoCarpeta.I_block[j])*int64(sb.S_block_s), 0)
			binary.Read(archivo, binary.LittleEndian, &bc)

			for _, content := range bc.B_content {
				if content.B_inodo != -1 {
					var inodoHijo types.Inodo
					archivo.Seek(int64(sb.S_inode_start)+int64(content.B_inodo)*int64(sb.S_inode_s), 0)
					binary.Read(archivo, binary.LittleEndian, &inodoHijo)

					nombre := strings.TrimRight(string(content.B_name[:]), "\x00")
					if nombre == "" {
						continue
					}

					permisosStr := permisosString(inodoHijo.I_perm, inodoHijo.I_type)
					fecha, hora := formatearFechaHora(inodoHijo.I_mtime[:])
					tipoStr := "Archivo"
					if inodoHijo.I_type == '0' {
						tipoStr = "Carpeta"
					}
					ownerStr := fmt.Sprintf("User%d", inodoHijo.I_uid)
					grupoStr := fmt.Sprintf("Group%d", inodoHijo.I_gid)

					grafo += fmt.Sprintf("      <tr><td>%s</td><td>%s</td><td>%s</td><td>%d</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>\n",
						permisosStr, ownerStr, grupoStr, inodoHijo.I_size, fecha, hora, tipoStr, nombre)
				}
			}
		}
	}

	grafo += "    </table>\n"
	grafo += "  >];\n"
	grafo += "}\n"

	rutaTxt, rutaFinal, formato := obteneRutaReporte(rutaReporte)
	prepararDirectorio(rutaTxt)
	prepararDirectorio(rutaFinal)
	reporte, _ := os.Create(rutaTxt)
	reporte.WriteString(grafo)
	reporte.Close()

	salida, errCmd := exec.Command("dot", "-T"+formato, rutaTxt, "-o", rutaFinal).CombinedOutput()
	if errCmd != nil {
		fmt.Println("[ERROR] Falló Graphviz al generar la imagen:")
		fmt.Println(string(salida))
	} else {
		fmt.Printf("[ÉXITO] Reporte LS generado exitosamente en: %s\n", rutaFinal)
	}
}

func ReporteTree(rutaReporte, rutaDisco, id string) {
	exec.Command("mkdir", "-p", filepath.Dir(rutaReporte)).Output()

	archivo, err := os.OpenFile(rutaDisco, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("[ERROR] No se pudo abrir el disco para el reporte TREE.")
		return
	}
	defer archivo.Close()

	partStart := buscarPartStart(archivo, rutaDisco, id)
	if partStart == -1 {
		fmt.Println("[ERROR] No se encontró la partición para el TREE.")
		return
	}

	var sb types.SuperBloque
	archivo.Seek(partStart, 0)
	binary.Read(archivo, binary.LittleEndian, &sb)

	bitmapInodos := make([]byte, sb.S_inodes_count)
	archivo.Seek(int64(sb.S_bm_inode_start), 0)
	binary.Read(archivo, binary.LittleEndian, &bitmapInodos)

	bitmapBloques := make([]byte, sb.S_blocks_count)
	archivo.Seek(int64(sb.S_bm_block_start), 0)
	binary.Read(archivo, binary.LittleEndian, &bitmapBloques)

	tipoBloque := make(map[int]int)
	for i := 0; i < int(sb.S_inodes_count); i++ {
		if bitmapInodos[i] == '1' {
			var inodo types.Inodo
			archivo.Seek(int64(sb.S_inode_start)+int64(i)*int64(sb.S_inode_s), 0)
			binary.Read(archivo, binary.LittleEndian, &inodo)

			for j := 0; j < 15; j++ {
				if inodo.I_block[j] != -1 {
					if j < 12 {
						if inodo.I_type == '0' {
							tipoBloque[int(inodo.I_block[j])] = 0
						} else {
							tipoBloque[int(inodo.I_block[j])] = 1
						}
					} else {
						tipoBloque[int(inodo.I_block[j])] = 2
					}
				}
			}
		}
	}

	grafo := "digraph G {\n"
	grafo += "  rankdir=LR;\n"
	grafo += "  node [shape=plaintext, fontname=\"Helvetica\"];\n"

	conexiones := ""

	for i := 0; i < int(sb.S_inodes_count); i++ {
		if bitmapInodos[i] == '1' {
			var inodo types.Inodo
			archivo.Seek(int64(sb.S_inode_start)+int64(i)*int64(sb.S_inode_s), 0)
			binary.Read(archivo, binary.LittleEndian, &inodo)

			grafo += fmt.Sprintf("  inodo_%d [label=<\n", i)
			grafo += "    <table border=\"0\" cellborder=\"1\" cellspacing=\"0\" bgcolor=\"#87CEFA\">\n"
			grafo += fmt.Sprintf("      <tr><td colspan=\"2\"><b>Inodo %d</b></td></tr>\n", i)
			grafo += fmt.Sprintf("      <tr><td>i_uid</td><td>%d</td></tr>\n", inodo.I_uid)
			grafo += fmt.Sprintf("      <tr><td>i_gid</td><td>%d</td></tr>\n", inodo.I_gid)
			grafo += fmt.Sprintf("      <tr><td>i_size</td><td>%d</td></tr>\n", inodo.I_size)
			grafo += fmt.Sprintf("      <tr><td>i_type</td><td>%c</td></tr>\n", inodo.I_type)

			for j := 0; j < 15; j++ {
				grafo += fmt.Sprintf("      <tr><td>i_block_%d</td><td port=\"b%d\">%d</td></tr>\n", j+1, j, inodo.I_block[j])
				if inodo.I_block[j] != -1 {
					conexiones += fmt.Sprintf("  inodo_%d:b%d -> bloque_%d;\n", i, j, inodo.I_block[j])
				}
			}
			grafo += "    </table>\n  >];\n"

		}
	}

	for i := 0; i < int(sb.S_blocks_count); i++ {
		if bitmapBloques[i] == '1' {
			tipo := tipoBloque[i]
			archivo.Seek(int64(sb.S_block_start)+int64(i)*int64(sb.S_block_s), 0)

			if tipo == 0 {
				var bc types.BloqueCarpeta
				binary.Read(archivo, binary.LittleEndian, &bc)

				grafo += fmt.Sprintf("  bloque_%d [label=<\n", i)
				grafo += "    <table border=\"0\" cellborder=\"1\" cellspacing=\"0\" bgcolor=\"#FA8072\">\n"
				grafo += fmt.Sprintf("      <tr><td colspan=\"2\"><b>Bloque Carpeta %d</b></td></tr>\n", i)
				grafo += "      <tr><td><b>b_name</b></td><td><b>b_inodo</b></td></tr>\n"

				for idx, c := range bc.B_content {
					nombre := limpiarCadenaHTML(c.B_name[:])
					if nombre == "" {
						nombre = "-"
					}
					grafo += fmt.Sprintf("      <tr><td>%s</td><td port=\"i%d\">%d</td></tr>\n", nombre, idx, c.B_inodo)

					if c.B_inodo != -1 && nombre != "." && nombre != ".." {
						conexiones += fmt.Sprintf("  bloque_%d:i%d -> inodo_%d;\n", i, idx, c.B_inodo)
					}
				}
				grafo += "    </table>\n  >];\n"

			} else if tipo == 1 {
				var ba types.BloqueArchivo
				binary.Read(archivo, binary.LittleEndian, &ba)
				contenido := limpiarCadenaHTML(ba.B_content[:])
				contenido = strings.ReplaceAll(contenido, "\n", "<br/>")

				grafo += fmt.Sprintf("  bloque_%d [label=<\n", i)
				grafo += "    <table border=\"0\" cellborder=\"1\" cellspacing=\"0\" bgcolor=\"#F7DC6F\">\n"
				grafo += fmt.Sprintf("      <tr><td><b>Bloque Archivo %d</b></td></tr>\n", i)
				grafo += fmt.Sprintf("      <tr><td>%s</td></tr>\n", contenido)
				grafo += "    </table>\n  >];\n"

			} else if tipo == 2 {
				var bap types.BloqueApuntadores
				binary.Read(archivo, binary.LittleEndian, &bap)

				grafo += fmt.Sprintf("  bloque_%d [label=<\n", i)
				grafo += "    <table border=\"0\" cellborder=\"1\" cellspacing=\"0\" bgcolor=\"#FA8072\">\n"
				grafo += fmt.Sprintf("      <tr><td><b>Bloque Apuntador %d</b></td></tr>\n", i)

				for idx, ptr := range bap.B_pointers {
					grafo += fmt.Sprintf("      <tr><td port=\"p%d\">%d</td></tr>\n", idx, ptr)
					if ptr != -1 {
						conexiones += fmt.Sprintf("  bloque_%d:p%d -> bloque_%d;\n", i, idx, ptr)
					}
				}
				grafo += "    </table>\n  >];\n"
			}
		}
	}

	grafo += conexiones
	grafo += fmt.Sprintf("\n  label=\"Generado el: %s\";\n", time.Now().Format("2006-01-02 15:04:05"))
	grafo += "}"

	grafo = strings.Map(func(r rune) rune {
		if (r >= 32 && r <= 126) || r == '\n' || r == '\t' {
			return r
		}
		return -1
	}, grafo)

	rutaTxt, rutaFinal, formato := obteneRutaReporte(rutaReporte)
	prepararDirectorio(rutaTxt)
	prepararDirectorio(rutaFinal)
	reporte, errFile := os.Create(rutaTxt)
	if errFile != nil {
		fmt.Println("[ERROR] No se pudo crear el archivo .txt para el reporte.")
		return
	}
	reporte.WriteString(grafo)
	reporte.Close()

	salida, errCmd := exec.Command("dot", "-T"+formato, rutaTxt, "-o", rutaFinal).CombinedOutput()
	if errCmd != nil {
		fmt.Println(" [ERROR] Graphviz falló en Reporte TREE")
		fmt.Printf("MENSAJE:\n%s\n", string(salida))
	} else {
		fmt.Println("[ÉXITO] Reporte TREE generado exitosamente en:", rutaFinal)
	}
}

func contarCeldasLogicas(archivo *os.File, p types.Partition) int {
	cantidad := 0
	nextEBR := p.Part_start
	posicionExtendida := p.Part_start

	for nextEBR != -1 && nextEBR != 0 {
		archivo.Seek(nextEBR, 0)
		var ebr types.EBR
		err := binary.Read(archivo, binary.LittleEndian, &ebr)
		if err != nil {
			break
		}

		if ebr.Part_start > posicionExtendida {
			cantidad++
		}

		if ebr.Part_mount == '1' || (ebr.Part_mount == '0' && ebr.Part_s > 0) {
			cantidad += 2
			posicionExtendida = ebr.Part_start + ebr.Part_s
		} else {
			posicionExtendida = ebr.Part_start + int64(binary.Size(ebr))
		}

		if ebr.Part_next == -1 {
			break
		}
		nextEBR = ebr.Part_next
	}

	if posicionExtendida < (p.Part_start + p.Part_s) {
		cantidad++
	}

	if cantidad == 0 {
		cantidad = 1
	}

	return cantidad
}

func limpiarCadenaHTML(cadena []byte) string {
	resultado := ""
	for _, b := range cadena {
		if b >= 32 && b <= 126 {
			resultado += string(b)
		}
	}
	resultado = strings.ReplaceAll(resultado, "<", "&lt;")
	resultado = strings.ReplaceAll(resultado, ">", "&gt;")
	resultado = strings.ReplaceAll(resultado, "&", "&amp;")
	return resultado
}

func prepararDirectorio(rutaArchivo string) {
	dir := filepath.Dir(rutaArchivo)
	os.MkdirAll(dir, 0755)

	if _, err := os.Stat(rutaArchivo); err == nil {
		os.Remove(rutaArchivo)
	}
}
