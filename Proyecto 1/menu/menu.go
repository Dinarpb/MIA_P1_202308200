package menu

import "fmt"

func Pause() {
	fmt.Println("Presione cualquier tecla para continuar")
	key := ""
	fmt.Scanln(&key)
}
