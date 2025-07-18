package main

import (
	"fmt"
	"os"

	"github.com/Layr-Labs/eigenda/test/v2/client"
	"github.com/Layr-Labs/eigenda/test/v2/load"
)

func main() {
	if len(os.Args) != 3 {
		panic(fmt.Sprintf("Expected 3 args, got %d. Usage: %s <env_file> <load_file>.\n"+
			"If '-' is passed in lieu of a config file, the config file path is read from the environment variable "+
			"$GENERATOR_ENV or $GENERATOR_LOAD, respectively.\n",
			len(os.Args), os.Args[0]))
	}

	envFile := os.Args[1]
	if envFile == "-" {
		envFile = os.Getenv("GENERATOR_ENV")
		if envFile == "" {
			panic("$GENERATOR_ENV not set")
		}
	}

	loadFile := os.Args[2]
	if loadFile == "-" {
		loadFile = os.Getenv("GENERATOR_LOAD")
		if loadFile == "" {
			panic("$GENERATOR_LOAD not set")
		}
	}

	c, err := client.GetClient(envFile)
	if err != nil {
		panic(fmt.Errorf("failed to get client: %w", err))
	}

	config, err := load.ReadConfigFile(loadFile)
	if err != nil {
		panic(fmt.Errorf("failed to read config file %s: %w", loadFile, err))
	}

	generator, err := load.NewLoadGenerator(config, c)
	if err != nil {
		panic(fmt.Errorf("failed to create load generator: %w", err))
	}

	signals := make(chan os.Signal)
	go func() {
		<-signals
		generator.Stop()
	}()

	generator.Start(true)
}
