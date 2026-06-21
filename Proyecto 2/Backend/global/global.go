package global

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

var SesionActiva bool = false
var UsuarioActual string = ""
var IdParticionActual string = ""
