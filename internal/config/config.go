package config

import (
	"io/fs"
	"os"
)

type Config struct {
	dir            string
	dirPermission  fs.FileMode
	filePermission fs.FileMode
}

func NewConfig(homeDir string) (*Config, error) {
	c := &Config{
		dir:            homeDir + "/.config/go-shell",
		dirPermission:  0755,
		filePermission: 0644,
	}
	if err := c.makeDir(); err != nil {
		return c, err
	}
	return c, nil
}

func (c Config) readFile(filename string) ([]byte, error) {
	filePath := c.dir + "/" + filename
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return []byte{}, nil
	}
	return os.ReadFile(filePath)
}

func (c Config) writeFile(filename string, data []byte) error {
	filePath := c.dir + "/" + filename
	return os.WriteFile(filePath, data, c.filePermission)
}

func (c Config) makeDir() error {
	if _, err := os.Stat(c.dir); os.IsNotExist(err) {
		return os.MkdirAll(c.dir, c.dirPermission)
	}
	return nil
}
