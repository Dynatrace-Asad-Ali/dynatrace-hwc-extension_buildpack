package packit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/cloudfoundry/packit/internal"
)

type BuildFunc func(BuildContext) (BuildResult, error)

type BuildContext struct {
	CNBPath    string
	Stack      string
	WorkingDir string
	Plan       BuildpackPlan
	Layers     Layers
}

type BuildResult struct {
	Plan      BuildpackPlan
	Layers    []Layer
	Processes []Process
}

type Process struct {
	Type    string   `toml:"type"`
	Command string   `toml:"command"`
	Args    []string `toml:"args"`
	Direct  bool     `toml:"direct"`
}

type BuildpackPlanEntry struct {
	Name     string                 `toml:"name"`
	Version  string                 `toml:"version"`
	Metadata map[string]interface{} `toml:"metadata"`
}

type BuildpackPlan struct {
	Entries []BuildpackPlanEntry `toml:"entries"`
}

func Build(f BuildFunc, options ...Option) {
	config := Config{
		exitHandler: internal.NewExitHandler(),
		args:        os.Args,
		tomlWriter:  internal.NewTOMLWriter(),
		envWriter:   internal.NewEnvironmentWriter(),
	}

	for _, option := range options {
		config = option(config)
	}

	var (
		layersPath = config.args[1]
		planPath   = config.args[3]
	)

	pwd, err := os.Getwd()
	if err != nil {
		config.exitHandler.Error(err)
		return
	}

	var plan BuildpackPlan
	_, err = toml.DecodeFile(planPath, &plan)
	if err != nil {
		config.exitHandler.Error(err)
		return
	}

	result, err := f(BuildContext{
		CNBPath:    filepath.Clean(strings.TrimSuffix(config.args[0], filepath.Join("bin", "build"))),
		Stack:      os.Getenv("CNB_STACK_ID"),
		WorkingDir: pwd,
		Plan:       plan,
		Layers: Layers{
			Path: layersPath,
		},
	})
	if err != nil {
		config.exitHandler.Error(err)
		return
	}

	err = config.tomlWriter.Write(planPath, result.Plan)
	if err != nil {
		config.exitHandler.Error(err)
		return
	}

	for _, layer := range result.Layers {
		err = config.tomlWriter.Write(filepath.Join(layersPath, fmt.Sprintf("%s.toml", layer.Name)), layer)
		if err != nil {
			config.exitHandler.Error(err)
			return
		}

		err = config.envWriter.Write(filepath.Join(layer.Path, "env"), layer.SharedEnv)
		if err != nil {
			config.exitHandler.Error(err)
			return
		}

		err = config.envWriter.Write(filepath.Join(layer.Path, "env.launch"), layer.LaunchEnv)
		if err != nil {
			config.exitHandler.Error(err)
			return
		}

		err = config.envWriter.Write(filepath.Join(layer.Path, "env.build"), layer.BuildEnv)
		if err != nil {
			config.exitHandler.Error(err)
			return
		}
	}

	if len(result.Processes) > 0 {
		var launch struct {
			Processes []Process `toml:"processes"`
		}
		launch.Processes = result.Processes

		err = config.tomlWriter.Write(filepath.Join(layersPath, "launch.toml"), launch)
		if err != nil {
			config.exitHandler.Error(err)
			return
		}
	}
}
