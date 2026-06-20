package analyzer

import (
	"MIAP1/disk"
	"MIAP1/filesystem"
	"MIAP1/format"
	"MIAP1/mount"
	"MIAP1/partition"
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

		analizarComando(entrada)
	}
}

func analizarComando(entrada string) {
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
		analizarRmdisk(parametros)
	case "fdisk":
		analizarFdisk(parametros)
	case "mount":
		analizarMount(parametros)
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
			fmt.Println("Error: Falta -user")
			return
		}
		users.Rmusr(user)
	case "chgrp":
		user, okU := parametros["user"]
		grp, okG := parametros["grp"]
		if !okU || !okG {
			fmt.Println("Error: Falta -user o -grp")
			return
		}
		users.Chgrp(user, grp)
	case "mkdir":
		analizarMkdir(parametros)
	case "mkfile":
		analizarMkfile(parametros)
	case "pause":
		fmt.Println("Presione ENTER para continuar...")
		bufio.NewReader(os.Stdin).ReadString('\n')
	default:
		if strings.HasPrefix(comando, "#") {
			fmt.Println(entrada)
		} else {
			fmt.Println("Error: Comando no reconocido.")
		}
	}
}

func extraerParametros(entrada string) map[string]string {
	parametros := make(map[string]string)
	re := regexp.MustCompile(`-(\w+)=("[^"]+"|\S+)`)
	coincidencias := re.FindAllStringSubmatch(entrada, -1)

	for _, coincidencia := range coincidencias {
		clave := strings.ToLower(coincidencia[1])
		valor := strings.Trim(coincidencia[2], "\"")
		parametros[clave] = valor
	}

	return parametros
}

func analizarMkdisk(parametros map[string]string) {
	sizeStr, okSize := parametros["size"]
	path, okPath := parametros["path"]

	if !okSize || !okPath {
		fmt.Println("Error: Faltan parámetros obligatorios para MKDISK.")
		return
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil || size <= 0 {
		fmt.Println("Error: El tamaño debe ser un número mayor a cero.")
		return
	}

	fit := "ff"
	if val, ok := parametros["fit"]; ok {
		fit = strings.ToLower(val)
		if fit != "bf" && fit != "ff" && fit != "wf" {
			fmt.Println("Error: Ajuste no válido.")
			return
		}
	}

	unit := "m"
	if val, ok := parametros["unit"]; ok {
		unit = strings.ToLower(val)
		if unit != "k" && unit != "m" {
			fmt.Println("Error: Unidad no válida.")
			return
		}
	}

	disk.CreateDisk(int64(size), fit, unit, path)
}

func analizarRmdisk(parametros map[string]string) {
	path, okPath := parametros["path"]

	if !okPath {
		fmt.Println("Error: Falta el parámetro obligatorio -path para RMDISK.")
		return
	}

	disk.DeleteDisk(path)
}

func analizarFdisk(parametros map[string]string) {
	sizeStr, okSize := parametros["size"]
	path, okPath := parametros["path"]
	name, okName := parametros["name"]

	if !okSize || !okPath || !okName {
		fmt.Println("Error: Faltan parámetros obligatorios para FDISK.")
		return
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil || size <= 0 {
		fmt.Println("Error: El tamaño debe ser un número mayor a cero.")
		return
	}

	unit := "k"
	if val, ok := parametros["unit"]; ok {
		unit = strings.ToLower(val)
		if unit != "b" && unit != "k" && unit != "m" {
			fmt.Println("Error: Unidad no válida.")
			return
		}
	}

	tipo := "p"
	if val, ok := parametros["type"]; ok {
		tipo = strings.ToLower(val)
		if tipo != "p" && tipo != "e" && tipo != "l" {
			fmt.Println("Error: Tipo no válido.")
			return
		}
	}

	fit := "wf"
	if val, ok := parametros["fit"]; ok {
		fit = strings.ToLower(val)
		if fit != "bf" && fit != "ff" && fit != "wf" {
			fmt.Println("Error: Ajuste no válido.")
			return
		}
	}

	partition.CreatePartition(int64(size), unit, path, tipo, fit, name)
}

func analizarMount(parametros map[string]string) {
	path, okPath := parametros["path"]
	name, okName := parametros["name"]

	if !okPath || !okName {
		fmt.Println("Error: Faltan parámetros obligatorios para MOUNT.")
		return
	}

	mount.MountPartition(path, name)
}

func analizarMkfs(parametros map[string]string) {
	id, okId := parametros["id"]

	if !okId {
		fmt.Println("Error: Falta el parámetro obligatorio -id para MKFS.")
		return
	}

	tipo := "full"
	if val, ok := parametros["type"]; ok {
		tipo = strings.ToLower(val)
		if tipo != "full" {
			fmt.Println("Error: Tipo no válido.")
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
		fmt.Println("Error: Falta el parámetro obligatorio -file para CAT.")
		return
	}

	filesystem.Cat(archivos)
}

func analizarLogin(parametros map[string]string) {
	user, okUser := parametros["user"]
	pass, okPass := parametros["pass"]
	id, okId := parametros["id"]

	if !okUser || !okPass || !okId {
		fmt.Println("Error: Faltan parámetros obligatorios para LOGIN.")
		return
	}

	users.Login(user, pass, id)
}

func analizarLogout(parametros map[string]string) {
	if len(parametros) > 0 {
		fmt.Println("Error: LOGOUT no debe recibir parámetros.")
		return
	}

	users.Logout()
}

func analizarMkgrp(parametros map[string]string) {
	name, okName := parametros["name"]

	if !okName {
		fmt.Println("Error: Falta el parámetro obligatorio -name para MKGRP.")
		return
	}

	users.Mkgrp(name)
}

func analizarRmgrp(parametros map[string]string) {
	name, okName := parametros["name"]

	if !okName {
		fmt.Println("Error: Falta el parámetro obligatorio -name para RMGRP.")
		return
	}

	users.Rmgrp(name)
}

func analizarMkusr(parametros map[string]string) {
	user, okU := parametros["user"]
	pass, okP := parametros["pass"]
	grp, okG := parametros["grp"]

	if !okU || !okP || !okG {
		fmt.Println("Error: Faltan parámetros obligatorios para MKUSR.")
		return
	}

	users.Mkusr(user, pass, grp)
}

func analizarMkdir(parametros map[string]string) {
	path, ok := parametros["path"]
	_, p := parametros["p"]

	if !ok {
		fmt.Println("Error: Falta -path.")
		return
	}
	filesystem.Mkdir(path, p)
}

func analizarMkfile(parametros map[string]string) {
	path, ok := parametros["path"]
	size := parametros["size"]
	cont := parametros["cont"]
	_, r := parametros["r"]

	if !ok {
		fmt.Println("Error: Falta -path.")
		return
	}
	filesystem.Mkfile(path, size, cont, r)
}
