package shell

import (
	"io/fs"
	"os"
)

type config struct {
	dir            string
	dirPermission  fs.FileMode
	filePermission fs.FileMode
}

func newConfig(homeDir string) (*config, error) {
	return &config{
		dir:            homeDir + "/.config/go-shell",
		dirPermission:  0755,
		filePermission: 0644,
	}, nil
}

func (c config) readFile(filename string) ([]byte, error) {
	filePath := c.dir + "/" + filename
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return []byte{}, nil
	}
	return os.ReadFile(filePath)
}

func (c config) writeFile(filename string, data []byte) error {
	filePath := c.dir + "/" + filename
	return os.WriteFile(filePath, data, c.filePermission)
}

func (c config) makeDir() error {
	if _, err := os.Stat(c.dir); os.IsNotExist(err) {
		return os.MkdirAll(c.dir, c.dirPermission)
	}
	return nil
}
