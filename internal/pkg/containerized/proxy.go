package containerized

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
)

var (
	proxydir = "/etc/containerd-proxy"
)

type proxyConfig struct {
	ID        string   `json:"-"`
	Namespace string   `json:"namespace"`
	Image     string   `json:"image"`
	ImagePath string   `json:"imagePath"`
	Args      []string `json:"args"`
	Scope     string   `json:"scope"`
}

func updateConfig(name, newImage string) error {
	cfg, err := loadConfig(name)
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	cfg.Image = newImage
	cfg.ImagePath = ""
	return storeConfig(name, cfg)
}

func loadConfig(name string) (*proxyConfig, error) {
	configFile := filepath.Join(proxydir, name+".json")
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	var cfg proxyConfig
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// storeConfig will write out the config only if it already exists
func storeConfig(name string, cfg *proxyConfig) error {
	configFile := filepath.Join(proxydir, name+".json")
	fd, err := os.OpenFile(configFile, os.O_RDWR, 0644)
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	err = fd.Truncate(0)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(fd)
	return enc.Encode(cfg)
}
