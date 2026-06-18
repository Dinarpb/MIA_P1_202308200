package types

type MBR struct {
	Tamanio       int64
	FechaCreacion [20]byte
	Id            int16
	Ajuste        byte
	Particiones   [4]Partition
}

type Partition struct {
	Estado  byte
	Tipo    byte
	Ajuste  byte
	Inicio  int64
	Tamanio int64
	Nombre  [20]byte
}
