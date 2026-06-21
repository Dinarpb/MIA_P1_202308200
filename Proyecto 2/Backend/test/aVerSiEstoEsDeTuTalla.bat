#Calificacion MIA 2026 VACAS VAQUERAS

#CREACION DE DISCOS

#error
MKdesk -size=75 -unit=M -path=/tmp/d1.dsk

#Disco con primer ajuste
mkdisk -size=75 -unit=M -path=/tmp/d1.dsk

#Tamaño de 50mb
mkdisk -unit=m -path=/tmp/d2.dsk -fit=BF -size=50


#Debe crear discos en MB
mkdisk -size=101 -path=/tmp/d3.dsk -fit=WF            	 
mkdisk -size=1 -path="/tmp/eliminar_1.dsk"
mkdisk -size=1 -path="/tmp/eliminar_2.dsk"

#CREACION DE PARTICIONES PRIMARIAS Y EXTENDIDAS

#Crear particiones d1.dsk
fdisk -type=P -unit=M -name=Part1 -size=25 -path=/tmp/d1.dsk
fdisk -type=P -unit=M -name=Part2 -size=25 -path=/tmp/d1.dsk
fdisk -type=P -unit=M -name=Part3 -size=20 -path=/tmp/d1.dsk

#ERROR
efedisk -type=P -unit=M -name=Part3 -size=20 -path=/tmp/d3.dsk

#Crear particiones d2.dsk
#Error, no existe extendida
fdisk -type=L -unit=M -name=Part6 -size=25 -path=/tmp/d2.dsk
PAUSE
#Ocupa los 10MB del disco
fdisk -type=E -unit=M -name=Part1 -size=10 -path=/tmp/d2.dsk -fit=FirstFit
#Error, ya existe una extendida
fdisk -type=E -unit=M -name=Part7 -size=25 -path=/tmp/d2.dsk -fit=WorstFit
PAUSE
#fdisk -type=L -unit=k -name=Part2 -size=1024 -path=/tmp/d2.dsk
#fdisk -type=L -unit=k -name=Part3 -size=1024 -path=/tmp/d2.dsk
#fdisk -type=L -unit=k -name=Part4 -size=1024 -path=/tmp/d2.dsk

#Crear particiones d3.dsk
fdisk -type=E -unit=M -name=Part1 -size=25 -path=/tmp/d3.dsk -fit=BestFit
fdisk -type=P -unit=M -name=Part2 -size=25 -path=/tmp/d3.dsk -fit=BestFit
#error
fdisk -tipe=P -unyt=M -name=Part4 -size=25 -ruta=/tmp/d3.dsk -fit=BestFit

fdisk -type=P -unit=M -name=Part3 -size=25 -path=/tmp/d3.dsk -fit=BestFit
fdisk -type=P -unit=M -name=Part4 -size=25 -path=/tmp/d3.dsk -fit=BestFit
#error, ya existen 4 particiones
fdisk -type=P -unit=M -name=Part1 -size=25 -path=/tmp/d3.dsk -fit=BestFit
PAUSE
#fdisk -type=L -unit=K -name=Part5 -size=1024 -path=/tmp/d3.dsk -fit=BestFit
#fdisk -type=L -unit=K -name=Part6 -size=1024 -path=/tmp/d3.dsk -fit=BestFit

#MOUNT
mount -path=/tmp/d1.dsk -name=Part1
mount -path=/tmp/d2.dsk -name=Part1
mount -path=/tmp/d3.dsk -name=Part1

########reporte disk del estado inicial de las particiones
rep -id=XXXX -Path=/home/parte1/particiones/d1.jpg -name=disk
rep -id=XXXX -Path=/home/parte1/particiones/d2.jpg -name=disk
rep -id=XXXX -Path=/home/parte1/particiones/d3.jpg -name=disk

PAUSE
#ELIMINACION DE DISCOS

#Debe de mostrar error por no existir
rmdisk -path="/home/a_eliminar_disco/no_existo.dsk"
PAUSE
rmdisk -path="/tmp/eliminar_1.dsk"
rmdisk -path="/tmp/eliminar_2.dsk"

#REPORTES MBR
rep -id=XXXX -Path=/home/parte1/mbr1.jpg -name=mbr
rep -id=XXXX -Path=/home/parte1/mbr2.jpg -name=mbr
rep -id=XXXX -Path=/home/parte1/mbr3.jpg -name=mbr

#UNMOUNT
unmount -id=XXXX

#Debe dar error porque ya no esta montada la particion
rep -id=XXXX -Path=/home/parte1/mbr3.jpg -name=mbr

#Cerrar el programa para validar
#Debe dar error porque no deberia estar montado nada
pause
rep -id=XXXX -Path=/home/parte1/mbr3.jpg -name=mbr


#MKFS A PARTICION
mkfs -type-=full -id-=XXXX #MKFS a Part1 del discon con nombre d2.dsk

#REPORTES INICIALES
rep -id=XXXX -Path="/home/parte2/inicial/ext2_sb_1.jpg" -name=sb
rep -id=XXXX -Path="/home/parte2/inicial/ext2_tree_1.jpg" -name=tree

PAUSE

#debe dar error pq no hay comando llamado asi:
Loogin -pass=567 -usr=roca -id=XXXX

#Debe dar error porque no existe el usuario roca
Login -pass=567 -usr=roca -id=XXXX
PAUSE
#Debe dar error porque no existe nada activo
logout
PAUSE
Login -pass=123 -usr=root -id=XXXX

pause
#CREACION DE GRUPOS
mkgrp -naMe=Archivos
mkgrp -naMe=elAuxEsBuenaOnda

#Validar cambios en el archivo
Cat -file=/users.txt
pause

#ELIMINACION DE GRUPOS
rmgrp -name=Arqui
rmgrp -nam=Archivos

#Validar cambios en el archivo
Cat -file=/users.txt
pause

#CREACION DE USUARIOS
Mkusr -usr="user1" -grp=root -pass=user1
Mkuser -usr="user2" -grp=root -pass=user2
Mkuser -usr="user3" -grp=root -pass=user3
Mkuser -usr="user4" -grp==root -pass=user4

#Validar cambios en el archivo
Cat -file=/users.txt
pause

#ELIMINACION DE USUARIOS
rmusr -usr="user3"
rmusrr -usr="user1"

#Validar cambios en el archivo
Cat -file=/users.txt
pause

#CAMBIAR USUARIO DE GRUPO
chgrp -usr="user3" -grp=elAuxEsBuenaOnda
chgrp -usr="user4" -grupo=elAuxEsBuenaOnda

#Validar cambios en el archivo
Cat -file=/users.txt
pause

#CREACION DE CARPETAS
Mkdir -P -path=/home/archivos/mia/fase2
Mkdir -P -path=/home/archivos/mia/carpeta2
Mkdir -P -path=/home/archivos/mia/z
Mkdir -P -path=/home/archivos/mia/carpeta3/carpeta7/carpeta8/carpeta9/carpeta10/carpeta11
Mkdir -P -path=/home/archivos/mia/carpeta4/carpeta7/carpeta8/carpeta9/carpeta10/carpeta11/carpeta7/carpeta8/carpeta9/carpeta10/carpeta11
Mkdir -path=/home/archivos/mia/carpeta2/a1
Mkdir -path=/home/archivos/mia/carpeta2/a2
Mkdir -path=/home/archivos/mia/carpeta2/a3
Mkdir -path=/home/archivos/mia/carpeta2/a4
Mkdir -path=/home/archivos/mia/carpeta2/a5
Mkdir -path=/home/archivos/mia/carpeta2/a6
Mkdir -path=/home/archivos/mia/carpeta2/a7
Mkdir -path=/home/archivos/mia/carpeta2/a8
Mkdir -path=/home/archivos/mia/carpeta2/a9
Mkdir -path=/home/archivos/mia/carpeta2/a10

rep -id=XXXX -Path="/home/parte2/avanzado/ext2_sb_2.jpg" -name=sb
rep -id=XXXX -Path="/home/parte2/avanzado/ext2_tree_2.jpg" -name=tree
rep -id=XXXX -Path="/home/parte2/avanzado/ext2_tree_2.jpg" -name=tree
rep -id=XXXX -Path="/home/parte2/avanzado/reporte_inode2.pdf" -name=inode
rep -id=XXXX -Path="/home/parte2/avanzado/reporte_block2.pdf" -name=block
rep -id=XXXX -Path="/home/parte2/avanzado/reporte_bm_inode2.pdf" -name=bm_inode
rep -id=XXXX -Path="/home/parte2/avanzado/reporte_bm_block2.pdf" -name=bm_block

#CREACION DE ARCHIVOS
mkfile -path="/home/b1.txt" -s=75
#Debe dar error ruta no existe
mkfile -path="/home/Noexiste/b2.txt" -s=75
mkfile -r -path="/home/AhoraSiExiste/b2.txt" -s=175

PAUSE

#CAMBIAR PARAMETRO -cont POR UN ARCHIVO QUE SI EXISTA EN SU COMPUTADORA
mkfile -path="/home/b3.txt" -cont=/home/usr/documents/algo.txt

PAUSE

rep -id=XXXX -Path="/home/parte2/CAMBIOS/ext2_tree_3.jpg" -name=tree
rep -id=XXXX -Path="/home/parte2/CAMBIOS/reporte_inode3.pdf" -name=inode
rep -id=XXXX -Path="/home/parte2/CAMBIOS/reporte_block3.pdf" -name=block
rep -id=XXXX -Path="/home/parte2/CAMBIOS/reporte_bm_inode3.pdf" -name=bm_inode
rep -id=XXXX -Path="/home/parte2/CAMBIOS/reporte_bm_block3.pdf" -name=bm_block

#CON ESTE ARCHIVO IGUAL NO SALE
#ARCHIVOS > ARQUI1
# ZZZZZ ARQUI1 ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ

#ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ
#ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ
#ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ

#ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ

#ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ