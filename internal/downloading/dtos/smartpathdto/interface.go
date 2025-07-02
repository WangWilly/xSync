package smartpathdto

// 路径Plus
type SmartPath interface {
	Path() (string, error)
	Create(name string) error
	Rename(string) error
	Remove() error
	Name() string
	Id() int
	Recorded() bool
}
