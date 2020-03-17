package unit

type Unit struct {
	Name     string  `json:"name"`
	Contents Builder `json:"contents,omitempty"`
}

func New(name string) *Unit {
	return &Unit{Name: name}
}

func (u *Unit) AddSection(name string, lines ...string) *SectionBuilder {
	return u.Contents.AddSection(name, lines...)
}

func (u *Unit) Bytes() []byte {
	return u.Contents.Bytes()
}

func (u *Unit) String() string {
	return string(u.Bytes())
}
