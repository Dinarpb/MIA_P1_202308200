package main

import (
	"MIAP1/disk"
	"MIAP1/menu"
	"MIAP1/partition"
	"MIAP1/report"
)

func main() {
	disk.CreateDisk(10, "ff", "k", "/home/dialjub/mia/disk.mia")
	disk.CreateDisk(1, "ff", "m", "/home/dialjub/mia/uno/dos/tres/diez/disk1.mia")
	disk.CreateDisk(20, "ff", "k", "/home/dialjub/mia/uno/dos/tres/diez/disk2.mia")
	partition.CreatePartition(1, "k", "/home/dialjub/mia/disk.mia", "P", "ff", "particion1")
	partition.CreatePartition(4, "k", "/home/dialjub/mia/disk.mia", "P", "ff", "particion2")
	report.MBR("/home/dialjub/mia/reports/mbr.png", "/home/dialjub/mia/disk.mia")
	menu.Pause()
}