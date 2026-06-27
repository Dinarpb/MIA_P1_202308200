package main

import (
	// Descomenta la ruta de tu paquete analizador
	"MIAP1/analyzer"
	"MIAP1/filesystem"
	"MIAP1/global"
	"MIAP1/report"
	"MIAP1/types"
	"MIAP1/users"
	"MIAP1/utils"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type NodoVisual struct {
	Nombre string `json:"nombre"`
	Tipo   string `json:"tipo"` // "0" carpeta, "1" archivo
}

// Estructura para recibir el comando desde el frontend
type PeticionComando struct {
	Comando string `json:"comando"`
}

type ParticionEstado struct {
	Nombre  string                      `json:"nombre"`
	Montada bool                        `json:"montada"`
	Id      string                      `json:"id"` // El ID de montaje si existe
	Espacio filesystem.EspacioParticion `json:"espacio"`
}

type DiscoEstado struct {
	Path        string            `json:"path"`
	TamanoTotal int64             `json:"tamanoTotal"`
	Particiones []ParticionEstado `json:"particiones"`
}

type Archivo struct {
	Nombre string `json:"nombre"`
	Tipo   string `json:"tipo"` // "0" para carpeta, "1" para archivo
	Ruta   string `json:"ruta"`
}

func main() {
	// 1. Inicializar el router de Gin
	router := gin.Default()

	// 2. Configurar CORS (Permite que el frontend en React/Angular se conecte)
	router.Use(cors.Default())

	// 3. Endpoint de prueba para verificar que el servidor vive
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"mensaje": "¡El backend de Dina está en línea!",
		})
	})

	router.GET("/sistema", func(c *gin.Context) {
		var sistema []DiscoEstado
		var files []string

		// 1. Ruta base donde empezará a buscar (puedes ajustarla si necesitas buscar desde más atrás)
		rutaBase := "/home/dinaarpb"

		// 2. Escanear recursivamente buscando .dsk y .mia
		filepath.WalkDir(rutaBase, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil // Ignorar carpetas a las que no tengamos permiso y seguir buscando
			}
			// Si no es un directorio, verificamos la extensión
			if !d.IsDir() {
				ext := strings.ToLower(filepath.Ext(path))
				if ext == ".dsk" || ext == ".mia" {
					files = append(files, path)
				}
			}
			return nil
		})

		// 3. Procesar los archivos encontrados
		for _, file := range files {
			// Abrir el disco para leer el MBR
			archivo, err := os.OpenFile(file, os.O_RDONLY, 0644)
			if err != nil {
				continue
			}
			mbr := utils.ObtenerMBR(archivo)
			archivo.Close()

			// Preparar el objeto del disco
			disco := DiscoEstado{Path: file, Particiones: []ParticionEstado{}}

			// Buscar si este disco está en la lista de montados
			var discoMontado *global.DiscoMontado = nil
			for _, d := range global.DiscosMontados {
				if d.Path == file {
					discoMontado = &d
					break
				}
			}

			// Recorrer las particiones del MBR físico
			for i := 0; i < 4; i++ {
				if mbr.Mbr_partitions[i].Part_status == '1' {
					nombre := strings.TrimRight(string(mbr.Mbr_partitions[i].Part_name[:]), "\x00")

					// Verificar si esta partición específica está montada
					estaMontada := false
					idMontaje := ""
					if discoMontado != nil {
						for _, p := range discoMontado.Particiones {
							if p.Nombre == nombre {
								estaMontada = true
								idMontaje = p.ID
								break
							}
						}
					}

					disco.Particiones = append(disco.Particiones, ParticionEstado{
						Nombre:  nombre,
						Montada: estaMontada,
						Id:      idMontaje,
					})
				}
			}
			sistema = append(sistema, disco)
		}

		c.JSON(http.StatusOK, sistema)
	})

	// 4. Endpoint principal que recibirá los comandos del frontend
	router.POST("/ejecutar", func(c *gin.Context) {
		// Estructura para recibir el JSON del frontend
		var peticion struct {
			Comando string `json:"comando"`
		}

		if err := c.ShouldBindJSON(&peticion); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 1. Crear un "tubo" (Pipe) para capturar lo que el programa imprima
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// 2. Ejecutar tu analizador real
		// Todo lo que use fmt.Println dentro de esta función quedará atrapado
		analyzer.AnalizarComando(peticion.Comando)

		// 3. Cerrar el tubo y restaurar la consola normal
		w.Close()
		os.Stdout = oldStdout

		// 4. Leer todo lo que se atrapó en el tubo
		var buf bytes.Buffer
		io.Copy(&buf, r)
		salidaReal := buf.String()

		// Si no imprimió nada, devolvemos un mensaje genérico
		if salidaReal == "" {
			salidaReal = "[INFO] Comando ejecutado, pero no devolvió ningún mensaje."
		}

		// 5. Enviar el texto real al Frontend
		c.JSON(http.StatusOK, gin.H{
			"output": salidaReal,
		})
	})

	router.GET("/explorar", func(c *gin.Context) {
		ruta := c.Query("ruta")
		if ruta == "" {
			ruta = "/" // Por defecto cargamos la raíz
		}

		// 1. Validar sesión
		if !users.SesionActiva {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "No hay sesión activa. Por favor inicie sesión."})
			return
		}

		// 2. Abrir partición
		archivo, sb, _, _, _, err := utils.ObtenerContextoParticion()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo acceder a la partición."})
			return
		}
		defer archivo.Close()

		// 3. Buscar la carpeta que el usuario quiere ver
		_, inodoCarpeta, errRuta := utils.BuscarInodoPorRuta(archivo, sb, ruta)
		if errRuta != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "La ruta no existe."})
			return
		}

		if inodoCarpeta.I_type == '1' {
			c.JSON(http.StatusBadRequest, gin.H{"error": "La ruta seleccionada es un archivo."})
			return
		}

		// 4. Leer los archivos dentro de la carpeta
		var elementos []NodoVisual

		for i := 0; i < 12; i++ {
			if inodoCarpeta.I_block[i] != -1 {
				var bc types.BloqueCarpeta
				archivo.Seek(int64(sb.S_block_start)+int64(inodoCarpeta.I_block[i])*int64(sb.S_block_s), 0)
				binary.Read(archivo, binary.LittleEndian, &bc)

				for j := 0; j < 4; j++ {
					if bc.B_content[j].B_inodo != -1 {
						nombreItem := strings.TrimRight(string(bc.B_content[j].B_name[:]), "\x00")

						// Leer el inodo hijo para saber si dibujamos un ícono de carpeta o de archivo
						var inodoHijo types.Inodo
						archivo.Seek(int64(sb.S_inode_start)+int64(bc.B_content[j].B_inodo)*int64(sb.S_inode_s), 0)
						binary.Read(archivo, binary.LittleEndian, &inodoHijo)

						tipoStr := "0"
						if inodoHijo.I_type == '1' || inodoHijo.I_type == 1 {
							tipoStr = "1"
						}

						elementos = append(elementos, NodoVisual{
							Nombre: nombreItem,
							Tipo:   tipoStr,
						})
					}
				}
			}
		}

		// 5. Enviar la lista de archivos a la página web
		c.JSON(http.StatusOK, gin.H{
			"ruta":     ruta,
			"archivos": elementos,
		})
	})
	// Reemplaza tu explorarHandler y el http.HandleFunc por este bloque dentro de func main()

	router.GET("/explorar-fisico", func(c *gin.Context) {
		ruta := c.Query("ruta")
		if ruta == "" {
			ruta = "/home/dinaarpb"
		}

		entradas, err := os.ReadDir(ruta)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("No se pudo leer la ruta: %v", err)})
			return
		}

		var archivos []Archivo
		for _, entrada := range entradas {
			tipo := "1" // Por defecto es un archivo
			if entrada.IsDir() {
				tipo = "0" // Es una carpeta
			}

			archivos = append(archivos, Archivo{
				Nombre: entrada.Name(),
				Tipo:   tipo,
				Ruta:   filepath.Join(ruta, entrada.Name()),
			})
		}

		c.JSON(http.StatusOK, gin.H{"archivos": archivos})
	})

	// Endpoint para generar reportes
	router.POST("/generar-reporte", func(c *gin.Context) {
		var req struct {
			TipoReporte string `json:"tipo_reporte"` // disk, mbr, inode, block, bm_inode, bm_block, sb, file, ls, tree
			ParticionID string `json:"particion_id"`
			RutaGuardar string `json:"ruta_guardar"`
			RutaArchivo string `json:"ruta_archivo"` // Para reportes de file/ls
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Crear directorio si no existe
		if req.RutaGuardar != "" {
			os.MkdirAll(filepath.Dir(req.RutaGuardar), 0755)
		}

		// Capturar stdout mientras se genera el reporte
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Generar el reporte
		report.GenerarReporte(req.TipoReporte, req.RutaGuardar, req.ParticionID, req.RutaArchivo)

		// Restaurar stdout
		w.Close()
		os.Stdout = oldStdout

		// Leer la salida
		var buf bytes.Buffer
		io.Copy(&buf, r)
		salida := buf.String()

		if salida == "" {
			salida = "[INFO] Reporte generado sin mensajes."
		}

		c.JSON(http.StatusOK, gin.H{
			"output": salida,
			"ruta":   req.RutaGuardar,
		})
	})

	// Endpoint para servir reportes estáticos (imágenes, SVG, etc)
	router.GET("/reporte/*filePath", func(c *gin.Context) {
		filePath := c.Param("filePath")
		// Remover el prefijo "/"
		if len(filePath) > 0 && filePath[0] == '/' {
			filePath = filePath[1:]
		}

		// Sanitizar la ruta para evitar traversal attacks
		safeFilePath := filePath
		if strings.Contains(safeFilePath, "..") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Ruta inválida"})
			return
		}

		// Intentar buscar el archivo en diferentes ubicaciones
		posiblesPaths := []string{
			filepath.Join("/tmp/reportes", safeFilePath),
			filepath.Join("/home/dinaarpb/reportes", safeFilePath),
			filepath.Join("/tmp", safeFilePath),
		}

		var archivoPath string
		for _, p := range posiblesPaths {
			if _, err := os.Stat(p); err == nil {
				archivoPath = p
				break
			}
		}

		if archivoPath == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Reporte no encontrado"})
			return
		}

		c.File(archivoPath)
	})

	// Endpoint para listar reportes disponibles
	router.GET("/listar-reportes", func(c *gin.Context) {
		// Intentar listar desde /tmp/reportes/
		reportDir := "/tmp/reportes/"

		// Si no existe, usar directorio del usuario
		if _, err := os.Stat(reportDir); os.IsNotExist(err) {
			reportDir = "/home/dinaarpb/reportes/"
		}

		entradas, err := os.ReadDir(reportDir)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo leer la carpeta de reportes"})
			return
		}

		var reportes []map[string]interface{}
		for _, entrada := range entradas {
			if !entrada.IsDir() {
				info, _ := entrada.Info()
				reportes = append(reportes, map[string]interface{}{
					"nombre":     entrada.Name(),
					"tamanio":    info.Size(),
					"modificado": info.ModTime().String(),
				})
			}
		}

		c.JSON(http.StatusOK, gin.H{"reportes": reportes})
	})

	fmt.Println("🚀 Servidor corriendo en http://localhost:3000")
	router.Run(":3000")
}
