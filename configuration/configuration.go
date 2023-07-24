package configuration

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"

	"github.com/bartossh/Computantis/bookkeeping"
	"github.com/bartossh/Computantis/dataprovider"
	"github.com/bartossh/Computantis/emulator"
	"github.com/bartossh/Computantis/fileoperations"
	"github.com/bartossh/Computantis/repository"
	"github.com/bartossh/Computantis/server"
	"github.com/bartossh/Computantis/validator"
	"github.com/bartossh/Computantis/walletapi"
)

// Configuration is the main configuration of the application that corresponds to the *.yaml file
// that holds the configuration.
type Configuration struct {
	Server       server.Config         `yaml:"server"`
	Database     repository.DBConfig   `yaml:"database"`
	Client       walletapi.Config      `yaml:"client"`
	FileOperator fileoperations.Config `yaml:"file_operator"`
	Validator    validator.Config      `yaml:"validator"`
	Emulator     emulator.Config       `yaml:"emulator"`
	DataProvider dataprovider.Config   `yaml:"data_provider"`
	Bookkeeper   bookkeeping.Config    `yaml:"bookkeeper"`
}

// Read reads the configuration from the file and returns the Configuration with set fields according to the yaml setup.
func Read(path string) (Configuration, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		return Configuration{}, err
	}

	var main Configuration
	err = yaml.Unmarshal(buf, &main)
	if err != nil {
		return Configuration{}, fmt.Errorf("in file %q: %w", path, err)
	}

	return main, err
}
