package filesystem

import (
	"MIAP1/types"
	"encoding/binary"
	"os"
)

type EspacioParticion struct {
	Formateada bool    `json:"formateada"`
	Total      int64   `json:"tamanoTotal"`
	Usado      int64   `json:"usado"`
	Libre      int64   `json:"libre"`
	Porcentaje float64 `json:"porcentajeUso"`
}

func ObtenerEspacioParticion(diskPath string, part types.Partition) EspacioParticion {
	res := EspacioParticion{Total: part.Part_s, Libre: part.Part_s}

	// Si la partición no está activa o no tiene tamaño, no hay nada que leer.
	if part.Part_status != '1' || part.Part_s <= 0 {
		return res
	}

	archivo, err := os.OpenFile(diskPath, os.O_RDONLY, 0644)
	if err != nil {
		return res
	}
	defer archivo.Close()

	var sb types.SuperBloque
	if _, err := archivo.Seek(part.Part_start, 0); err != nil {
		return res
	}
	if err := binary.Read(archivo, binary.LittleEndian, &sb); err != nil {
		return res
	}

	// 0xEF53 es el "magic number" típico de EXT2. Ajusta este valor si en tu
	// Mkfs usas otro.
	if sb.S_magic != 0xEF53 {
		return res // no formateada (o el magic no coincide)
	}

	totalBlocks := int64(sb.S_blocks_count)
	freeBlocks := int64(sb.S_free_blocks_count)
	usedBlocks := totalBlocks - freeBlocks
	if usedBlocks < 0 {
		usedBlocks = 0
	}

	res.Formateada = true
	res.Usado = usedBlocks * int64(sb.S_block_s)
	if res.Usado > res.Total {
		res.Usado = res.Total
	}
	res.Libre = res.Total - res.Usado
	if res.Total > 0 {
		res.Porcentaje = float64(res.Usado) / float64(res.Total) * 100
	}
	return res
}
