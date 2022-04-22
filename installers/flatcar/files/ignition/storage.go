package ignition

type Disk struct {
	Device    string       `json:"device"`
	WipeTable bool         `json:"wipeTable,omitempty"`
	Paritions []*Partition `json:"partitions,omitempty"`
}

type File struct {
	Path     string `json:"path"`
	Contents string `json:"contents,omitempty"`
	Mode     int    `json:"mode,omitempty"`
	UID      int    `json:"uid,omitempty"`
	GID      int    `json:"gid,omitempty"`
}

type Filesystem struct {
	Device string             `json:"device"`
	Format string             `json:"format"`
	Files  []*File            `json:"files,omitempty"`
	Create *FilesystemOptions `json:"create,omitempty"`
}

type FilesystemOptions struct {
	Force   bool     `json:"force,omitempty"`
	Options []string `json:"options,omitempty"`
}

type Partition struct {
	Label    string `json:"label,omitempty"`
	Number   int    `json:"number,omitempty"`
	Size     int    `json:"size,omitempty"`
	Start    int    `json:"start,omitempty"`
	TypeGUID string `json:"typeGuid,omitempty"`
}

type RAID struct {
	Name    string   `json:"name"`
	Level   string   `json:"level"`
	Devices []string `json:"devices"`
	Spares  int      `json:"spares,omitempty"`
}

type Storage struct {
	Disks       []*Disk       `json:"disks,omitempty"`
	RAID        []*RAID       `json:"raid,omitempty"`
	Filesystems []*Filesystem `json:"filesystems,omitempty"`
}
