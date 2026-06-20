package types

type MBR struct {
	Mbr_tamano         int64
	Mbr_fecha_creacion [20]byte
	Mbr_dsk_signature  int32
	Dsk_fit            byte
	Mbr_partitions     [4]Partition
}

type Partition struct {
	Part_status      byte
	Part_type        byte
	Part_fit         byte
	Part_start       int64
	Part_s           int64
	Part_name        [16]byte
	Part_correlative int32
	Part_id          [4]byte
}

type EBR struct {
	Part_mount byte
	Part_fit   byte
	Part_start int64
	Part_s     int64
	Part_next  int64
	Part_name  [16]byte
}

type Superblock struct {
	S_filesystem_type   int32
	S_inodes_count      int32
	S_blocks_count      int32
	S_free_blocks_count int32
	S_free_inodes_count int32
	S_mtime             [20]byte
	S_umtime            [20]byte
	S_mnt_count         int32
	S_magic             int32
	S_inode_s           int32
	S_block_s           int32
	S_firts_ino         int32
	S_first_blo         int32
	S_bm_inode_start    int32
	S_bm_block_start    int32
	S_inode_start       int32
	S_block_start       int32
}

type Inodo struct {
	I_uid   int32
	I_gid   int32
	I_s     int64
	I_atime [20]byte
	I_ctime [20]byte
	I_mtime [20]byte
	I_block [15]int32
	I_type  byte
	I_perm  [3]byte
}

type BloqueArchivo struct {
	B_content [64]byte
}

type BloqueApuntadores struct {
	B_pointers [16]int32
}
