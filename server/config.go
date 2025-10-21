package server

import (
	"fmt"
	"github.com/spf13/viper"
	"os"
)

var CONFIG = ".config.json"

func LoadConfig() error {
	if len(os.Args) == 1 {
		fmt.Println("Using Default Config")
	} else {
		CONFIG = os.Args[1]
	}
	viper.SetConfigType("json")
	viper.SetConfigFile(CONFIG)
	fmt.Printf("Using config: %s\n", viper.ConfigFileUsed())
	viper.ReadInConfig()
	if viper.IsSet("addr") {
		fmt.Println("Address: ", viper.Get("addr"))
	} else {
		fmt.Println("Address not set")
	}
	var t ServerOpts
	err := viper.Unmarshal(&t)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}
