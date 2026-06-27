package analyzer

import (
	"MIAP1/disk"
	"MIAP1/filesystem"
	"MIAP1/format"
	"MIAP1/mount"
	"MIAP1/partition"
	"MIAP1/report"
	"MIAP1/users"
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func IniciarConsola() {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("MIA_EXT2> ")
		entrada, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error leyendo la entrada.")
			continue
		}

		entrada = strings.TrimSpace(entrada)
		if entrada == "" {
			continue
		}

		if strings.ToLower(entrada) == "exit" {
			break
		}

		AnalizarComando(entrada)
	}
}

func AnalizarComando(entrada string) {
	partes := strings.Fields(entrada)
	if len(partes) == 0 {
		return
	}

	comando := strings.ToLower(partes[0])
	parametros := extraerParametros(entrada)

	switch comando {
	case "mkdisk":
		analizarMkdisk(parametros)
	case "rmdisk":
		analizarRmdisk(parametros["path"])
	case "fdisk":
		analizarFdisk(parametros)
	case "mount":
		analizarMount(parametros)
	case "unmount":
		analizarUnmount(parametros)
	case "mkfs":
		analizarMkfs(parametros)
	case "cat":
		analizarCat(parametros)
	case "login":
		analizarLogin(parametros)
	case "logout":
		analizarLogout(parametros)
	case "mkgrp":
		analizarMkgrp(parametros)
	case "rmgrp":
		analizarRmgrp(parametros)
	case "mkusr":
		analizarMkusr(parametros)
	case "rmusr":
		user, ok := parametros["user"]
		if !ok {
			fmt.Println("[ERROR] Falta -user")
			return
		}
		users.Rmusr(user)
	case "chgrp":
		user, okU := parametros["user"]
		grp, okG := parametros["grp"]
		if !okU || !okG {
			fmt.Println("[ERROR] Falta -user o -grp")
			return
		}
		users.Chgrp(user, grp)
	case "mkdir":
		analizarMkdir(parametros)
	case "mkfile":
		analizarMkfile(parametros)
	case "pause":
		analizarPause(parametros)
	case "rep":
		analizarRep(parametros)
	case "execute", "exec":
		analizarExecute(parametros)
	case "rename":
		analizarRename(parametros)
	case "edit":
		analizarEdit(parametros)
	case "remove":
		analizarRemove(parametros)
	case "copy":
		analizarCopy(parametros)
	case "move":
		analizarMove(parametros)
	default:
		if strings.HasPrefix(comando, "#") {
			fmt.Println(entrada)
		} else {
			fmt.Println("[ERROR] Comando no reconocido.")
		}
	}
}

func extraerParametros(entrada string) map[string]string {
	parametros := make(map[string]string)

	// 1. Capturamos los pares -clave=valor (con o sin comillas)
	reValor := regexp.MustCompile(`-(\w+)=("[^"]+"|\S+)`)
	coincidencias := reValor.FindAllStringSubmatch(entrada, -1)
	for _, coincidencia := range coincidencias {
		clave := strings.ToLower(coincidencia[1])
		valor := strings.Trim(coincidencia[2], "\"")
		valor = strings.ReplaceAll(valor, "\"", "")
		valor = strings.ReplaceAll(valor, "'", "")
		parametros[clave] = valor
	}

	// 2. Quitamos lo ya capturado y buscamos flags sueltos (-p, -r, etc.)
	restante := reValor.ReplaceAllString(entrada, "")
	reFlag := regexp.MustCompile(`-(\w+)`)
	flags := reFlag.FindAllStringSubmatch(restante, -1)
	for _, f := range flags {
		clave := strings.ToLower(f[1])
		if _, existe := parametros[clave]; !existe {
			parametros[clave] = "" // presente, pero sin valor (flag booleano)
		}
	}

	return parametros
}

func analizarMkdisk(parametros map[string]string) {
	sizeStr, okSize := parametros["size"]
	path, okPath := parametros["path"]

	if !okSize || !okPath {
		fmt.Println("[ERROR] Faltan parámetros obligatorios para MKDISK.")
		return
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil || size <= 0 {
		fmt.Println("[ERROR] El tamaño debe ser un número mayor a cero.")
		return
	}

	fit := "ff"
	if val, ok := parametros["fit"]; ok {
		fit = strings.ToLower(val)
		if fit != "bf" && fit != "ff" && fit != "wf" {
			fmt.Println("[ERROR] Ajuste no válido.")
			return
		}
	}

	unit := "m"
	if val, ok := parametros["unit"]; ok {
		unit = strings.ToLower(val)
		if unit != "k" && unit != "m" {
			fmt.Println("[ERROR] Unidad no válida.")
			return
		}
	}

	disk.CreateDisk(int64(size), fit, unit, path)
}

func analizarRmdisk(parametros string) {

	println("RMDISK ejecutado con éxito.")
	// Validación de seguridad antes de llamar a la función
	if parametros == "" {
		fmt.Println("[ERROR] RMDISK requiere el parámetro -path.")
	} else {
		// Si el path no está vacío, llamamos a la función
		println("RMDISK ejecutado con éxito. Path: " + parametros)
		disk.DeleteDisk(parametros)
	}
}

func analizarFdisk(parametros map[string]string) {

	path, okPath := parametros["path"]
	name, okName := parametros["name"]

	if !okPath || !okName {
		fmt.Println("[ERROR] FDISK requiere obligatoriamente -path y -name.")
		return
	}

	// --- RAMA A: ¿Es un DELETE? ---
	if deleteVal, ok := parametros["delete"]; ok {
		// Llamamos a la lógica de borrado que definimos antes
		addVal := ""
		unit := "k"
		size := int(0)
		filesystem.Fdisk(path, name, deleteVal, addVal, unit, size)
		return
	}

	// --- RAMA B: ¿Es un ADD? ---
	if addVal, ok := parametros["add"]; ok {
		deleteVal := ""
		size := int(0)
		unit := "k"
		if val, okUnit := parametros["unit"]; okUnit {
			unit = strings.ToLower(val)
		}
		filesystem.Fdisk(path, name, deleteVal, addVal, unit, size)
		return
	}

	// --- RAMA C: Es CREACIÓN (Tu lógica original de Fase 1) ---
	// Aquí sí pedimos los parámetros obligatorios de creación
	sizeStr, okSize := parametros["size"]
	if !okSize {
		fmt.Println("[ERROR] Falta -size para crear partición.")
		return
	}

	size, _ := strconv.Atoi(sizeStr)

	// Tus validaciones originales de unit, type, fit...
	unit := "k"
	if val, ok := parametros["unit"]; ok {
		unit = strings.ToLower(val)
	}

	tipo := "p"
	if val, ok := parametros["type"]; ok {
		tipo = strings.ToLower(val)
	}

	fit := "wf"
	if val, ok := parametros["fit"]; ok {
		fit = strings.ToLower(val)
	}

	// Llamada a tu función original de creación
	partition.CreatePartition(int64(size), unit, path, tipo, fit, name)
}

func analizarMount(parametros map[string]string) {
	path, okPath := parametros["path"]
	name, okName := parametros["name"]

	if !okPath || !okName {
		fmt.Println("[ERROR] Faltan parámetros obligatorios para MOUNT.")
		return
	}

	mount.MountPartition(path, name)
}

func analizarMkfs(parametros map[string]string) {
	id, okId := parametros["id"]

	if !okId {
		fmt.Println("[ERROR] Falta el parámetro obligatorio -id para MKFS.")
		return
	}

	tipo := "full"
	if val, ok := parametros["type"]; ok {
		tipo = strings.ToLower(val)
		if tipo != "full" {
			fmt.Println("[ERROR] Tipo no válido.")
			return
		}
	}

	format.Mkfs(id, tipo)
}

func analizarCat(parametros map[string]string) {
	var archivos []string

	if val, ok := parametros["file"]; ok {
		archivos = append(archivos, val)
	}

	i := 1
	for {
		key := fmt.Sprintf("file%d", i)
		if val, ok := parametros[key]; ok {
			archivos = append(archivos, val)
			i++
		} else {
			break
		}
	}

	if len(archivos) == 0 {
		fmt.Println("[ERROR] Falta el parámetro obligatorio -file o -file1 para CAT.")
		return
	}

	for _, archivoRuta := range archivos {
		filesystem.Cat(archivoRuta)
	}
}

func analizarLogin(parametros map[string]string) {
	user, okUser := parametros["user"]
	pass, okPass := parametros["pass"]
	id, okId := parametros["id"]

	if !okUser || !okPass || !okId {
		fmt.Println("[ERROR] Faltan parámetros obligatorios para LOGIN.")
		return
	}

	users.Login(user, pass, id)
}

func analizarLogout(parametros map[string]string) {
	if len(parametros) > 0 {
		fmt.Println("[ERROR] LOGOUT no debe recibir parámetros.")
		return
	}

	users.Logout()
}

func analizarMkgrp(parametros map[string]string) {
	name, okName := parametros["name"]

	if !okName {
		fmt.Println("[ERROR] Falta el parámetro obligatorio -name para MKGRP.")
		return
	}

	users.Mkgrp(name)
}

func analizarRmgrp(parametros map[string]string) {
	name, okName := parametros["name"]

	if !okName {
		fmt.Println("[ERROR] Falta el parámetro obligatorio -name para RMGRP.")
		return
	}

	users.Rmgrp(name)
}

func analizarMkusr(parametros map[string]string) {
	user, okU := parametros["user"]
	pass, okP := parametros["pass"]
	grp, okG := parametros["grp"]

	if !okU || !okP || !okG {
		fmt.Println("[ERROR] Faltan parámetros obligatorios para MKUSR.")
		return
	}

	users.Mkusr(user, pass, grp)
}

func analizarMkdir(parametros map[string]string) {
	path, ok := parametros["path"]
	if !ok {
		fmt.Println("[ERROR] Falta -path.")
		return
	}

	p := false
	if valorP, existe := parametros["p"]; existe {
		if valorP != "" {
			fmt.Println("[ERROR] El parámetro -p no debe recibir ningún valor.")
			return
		}
		p = true
	}

	filesystem.Mkdir(path, p)
}

func analizarMkfile(parametros map[string]string) {
	path, ok := parametros["path"]
	if !ok {
		fmt.Println("[ERROR] Falta -path.")
		return
	}

	r := false
	if valorR, existe := parametros["r"]; existe {
		if valorR != "" {
			fmt.Println("[ERROR] El parámetro -r no debe recibir ningún valor.")
			return
		}
		r = true
	}

	size := 0
	if val, ok := parametros["size"]; ok {
		sizeConv, err := strconv.Atoi(val)
		if err != nil {
			fmt.Println("[ERROR] -size debe ser un número.")
			return
		}
		size = sizeConv
	}

	cont := ""
	if val, ok := parametros["cont"]; ok {
		cont = val
	}

	filesystem.Mkfile(path, r, size, cont)
}

func analizarPause(parametros map[string]string) {
	fmt.Println("\n[PAUSE] Ejecución pausada. Presiona 'Enter' para continuar...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

func analizarRep(parametros map[string]string) {
	name, okName := parametros["name"]
	path, okPath := parametros["path"]
	id, okId := parametros["id"]
	pathFileLs, okPathFileLs := parametros["path_file_ls"]

	if !okName || !okPath || !okId {
		fmt.Println("[ERROR] Faltan parámetros obligatorios para REP.")
		return
	}

	name = strings.ToLower(name)
	nombresValidos := map[string]bool{
		"mbr": true, "disk": true, "inode": true, "block": true,
		"bm_inode": true, "bm_block": true, "sb": true,
		"file": true, "ls": true, "tree": true,
	}

	if !nombresValidos[name] {
		fmt.Println("[ERROR] El nombre del reporte no es válido.")
		return
	}

	if (name == "file" || name == "ls") && !okPathFileLs {
		fmt.Println("[ERROR] Falta el parámetro -path_file_ls para el reporte solicitado.")
		return
	}

	if !okPathFileLs {
		pathFileLs = ""
	}

	report.GenerarReporte(name, path, id, pathFileLs)
}

func analizarExecute(parametros map[string]string) {
	ruta, ok := parametros["path"]
	if !ok {
		fmt.Println("[ERROR] Falta el parámetro -path en el comando execute.")
		return
	}

	archivo, err := os.Open(ruta)
	if err != nil {
		fmt.Println("Error al abrir el script:", err)
		return
	}
	defer archivo.Close()

	scanner := bufio.NewScanner(archivo)
	for scanner.Scan() {
		linea := strings.TrimSpace(scanner.Text())

		if linea == "" {
			continue
		}

		AnalizarComando(linea)
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error al leer el archivo:", err)
	}
}

func analizarUnmount(parametros map[string]string) {
	id, ok := parametros["id"]
	if !ok {
		fmt.Println("[ERROR] Falta el parámetro obligatorio -id para UNMOUNT.")
		return
	}

	mount.Unmount(id)
}

func analizarRename(parametros map[string]string) {
	path, okPath := parametros["path"]
	name, okName := parametros["name"]
	if !okPath || !okName {
		fmt.Println("[ERROR] RENAME requiere obligatoriamente -path y -name.")
		return
	}
	filesystem.Rename(path, name)
}

func analizarEdit(parametros map[string]string) {
	path, okPath := parametros["path"]
	cont, okCont := parametros["contenido"]
	if !okPath || !okCont {
		fmt.Println("[ERROR] EDIT requiere obligatoriamente -path y -contenido.")
		return
	}
	filesystem.Edit(path, cont)
}

func analizarRemove(parametros map[string]string) {
	path, okPath := parametros["path"]
	if !okPath {
		fmt.Println("[ERROR] REMOVE requiere obligatoriamente -path.")
		return
	}
	filesystem.Remove(path) // Lo implementaremos después
	fmt.Printf("[INFO] Función REMOVE en construcción para la ruta: %s\n", path)
}

func analizarCopy(parametros map[string]string) {
	path, okPath := parametros["path"]
	destino, okDest := parametros["destino"]
	if !okPath || !okDest {
		fmt.Println("[ERROR] COPY requiere obligatoriamente -path y -destino.")
		return
	}
	filesystem.Copy(path, destino) // Lo implementaremos después
	fmt.Printf("[INFO] Función COPY en construcción. Origen: %s, Destino: %s\n", path, destino)
}

func analizarMove(parametros map[string]string) {
	path, okPath := parametros["path"]
	destino, okDest := parametros["destino"]
	if !okPath || !okDest {
		fmt.Println("[ERROR] MOVE requiere obligatoriamente -path y -destino.")
		return
	}
	filesystem.Move(path, destino) // Lo implementaremos después
	fmt.Printf("[INFO] Función MOVE en construcción. Origen: %s, Destino: %s\n", path, destino)
}
