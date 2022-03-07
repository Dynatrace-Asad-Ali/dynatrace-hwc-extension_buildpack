/*
Copyright 2020 Dynatrace LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package hooks

import (
	"fmt"
	"os"

	"github.com/cloudfoundry/libbuildpack"
)

type hooks1 struct {
	libbuildpack.DefaultHook
}

type hooks2 struct {
	libbuildpack.DefaultHook
}

func init() {
	if os.Getenv("BP_DEBUG") != "" {
		libbuildpack.AddHook(hooks1{})
		libbuildpack.AddHook(hooks2{})
	}
}

func (h hooks1) BeforeCompile(compiler *libbuildpack.Stager) error {
	fmt.Println("HOOKS 1: BeforeCompile")
	return nil
}

func (h hooks2) AfterCompile(compiler *libbuildpack.Stager) error {
	fmt.Println("HOOKS 2: AfterCompile")
	return nil
}
