package config

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

////////////////////////////////////////////////////////////////////////////////
// Configuration Structures
////////////////////////////////////////////////////////////////////////////////

// Cookie represents authentication cookies for Twitter API
type Cookie struct {
	AuthToken string `yaml:"auth_token"`
	Ct0       string `yaml:"ct0"`
}

// Config represents the main application configuration
type Config struct {
	RootPath           string `yaml:"root_path"`
	Cookie             Cookie `yaml:"cookie"`
	MaxDownloadRoutine int    `yaml:"max_download_routine"`
}

////////////////////////////////////////////////////////////////////////////////
// Configuration Management Functions
////////////////////////////////////////////////////////////////////////////////

// ReadConfig reads configuration from the specified path
func ReadConfig(path string) (*Config, error) {
	file, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var result Config
	err = yaml.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// WriteConfig writes configuration to the specified path
func WriteConfig(path string, conf *Config) error {
	file, err := os.OpenFile(path, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := yaml.Marshal(conf)
	if err != nil {
		return err
	}
	_, err = io.Copy(file, bytes.NewReader(data))
	return err
}

// PromptConfig interactively prompts user for configuration and saves it
func PromptConfig(saveto string) (*Config, error) {
	conf := Config{}
	scan := bufio.NewScanner(os.Stdin)

	print("enter storage dir: ")
	scan.Scan()
	storePath := scan.Text()
	// ensure path is available
	err := os.MkdirAll(storePath, 0755)
	if err != nil {
		return nil, err
	}
	storePath, err = filepath.Abs(storePath)
	if err != nil {
		return nil, err
	}

	conf.RootPath = storePath

	print("enter auth_token: ")
	scan.Scan()
	conf.Cookie.AuthToken = scan.Text()

	print("enter ct0: ")
	scan.Scan()
	conf.Cookie.Ct0 = scan.Text()

	print("enter max download routine: ")
	scan.Scan()
	conf.MaxDownloadRoutine, err = strconv.Atoi(scan.Text())
	if err != nil {
		return nil, err
	}

	return &conf, WriteConfig(saveto, &conf)
}

////////////////////////////////////////////////////////////////////////////////
// Additional Cookie Management
////////////////////////////////////////////////////////////////////////////////

// ReadAdditionalCookies reads additional cookies from the specified path
func ReadAdditionalCookies(path string) ([]*Cookie, error) {
	res := []*Cookie{}
	file, err := os.OpenFile(path, os.O_RDONLY, 0)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return res, yaml.Unmarshal(data, &res)
}
