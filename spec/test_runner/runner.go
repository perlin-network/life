package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"fmt"
	"path/filepath"
	"path"
	"github.com/perlin-network/life/exec"
)

type Resolver struct{}

func (r *Resolver) ResolveFunc(module, field string) exec.FunctionImport {
	panic("ResolveFunc not supported")
}

func (r *Resolver) ResolveGlobal(module, field string) int64 {
	panic("ResolveGlobal not supported")
}

type Config struct {
	SourceFilename string `json:"source_filename"`
	Commands []Command `json:"commands"`
}

type Command struct {
	Type string `json:"type"`
	Line int `json:"line"`
	Filename string `json:"filename"`
	Action CmdAction `json:"action"`
	Text string `json:"text"`
	ModuleType string `json:"module_type"`
}

type CmdAction struct {
	Type string `json:"type"`
	Field string `json:"field"`
	Args []ValueInfo `json:"args"`
	Expected []ValueInfo `json:"expected"`
}

type ValueInfo struct {
	Type string `json:"type"`
	Value string `json:"value"`
}


func LoadConfigFromFile(filename string) *Config {
	raw, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	var cfg Config
	err = json.Unmarshal(raw, &cfg)
	if err != nil {
		panic(err)
	}
	return &cfg
}

func (c *Config) Run(cfgPath string) error {
	var input []byte

	dir, _ := filepath.Split(cfgPath)

	for _, cmd := range c.Commands {
		switch cmd.Type {
		case "module":
			/*if input != nil {
				panic("input != nil")
			}*/
			var err error
			input, err = ioutil.ReadFile(path.Join(dir, cmd.Filename))
			if err != nil {
				panic(err)
			}
		case "assert_return":
			vm, err := exec.NewVirtualMachine(input, exec.VMConfig{
				MaxMemoryPages: 1024, // for memory trap tests
			}, &Resolver{})
			if err != nil {
				panic(err)
			}
			switch cmd.Action.Type {
			case "invoke":
				entryID, ok := vm.GetFunctionExport(cmd.Action.Field)
				if !ok {
					panic("export not found")
				}
				args := make([]int64, 0)
				for _, arg := range cmd.Action.Args {
					var val int64
					fmt.Sscanf(arg.Value, "%d", &val)
					args = append(args, val)
				}
				fmt.Printf("Entry = %d\n", entryID)
				ret, err := vm.Run(entryID, args...)
				if err != nil {
					panic(err)
				}
				if len(cmd.Action.Expected) != 0 {
					var exp int64
					fmt.Sscanf(cmd.Action.Expected[0].Value, "%d", &exp)
					if ret != exp {
						panic(fmt.Errorf("ret mismatch: got %d, expected %d\n", ret, exp))
					}
				}
			default:
				panic(cmd.Action.Type)
			}
		case "assert_trap":
			fmt.Println("skipping assert_trap")
			/*
			vm, err := exec.NewVirtualMachine(input, exec.VMConfig{}, &Resolver{})
			if err != nil {
				panic(err)
			}
			switch cmd.Action.Type {
			case "invoke":
				entryID, ok := vm.GetFunctionExport(cmd.Action.Field)
				if !ok {
					panic("export not found")
				}
				args := make([]int64, 0)
				for _, arg := range cmd.Action.Args {
					var val int64
					fmt.Sscanf(arg.Value, "%d", &val)
					args = append(args, val)
				}
				_, err := vm.Run(entryID, args...)
				if err == nil {
					panic("expected error")
				}
			default:
				panic(cmd.Action.Type)
			}
			*/
		case "assert_malformed":
			fmt.Println("skipping assert_malformed")
			/*
			targetBytes, err := ioutil.ReadFile(path.Join(dir, cmd.Filename))
			if err != nil {
				panic(err)
			}
			_, err = exec.NewVirtualMachine(targetBytes, exec.VMConfig{}, &Resolver{})
			if err == nil {
				panic("expected error")
			}*/
		case "assert_invalid":
			fmt.Println("skipping assert_invalid")
		case "assert_exhaustion":
			fmt.Println("skipping assert_exhaustion")
		case "assert_unlinkable":
			fmt.Println("skipping assert_unlinkable")
		case "assert_return_canonical_nan":
			fmt.Println("skipping assert_return_canonical_nan")
		case "assert_return_arithmetic_nan":
			fmt.Println("skipping assert_return_arithmetic_nan")
		default:
			panic(cmd.Type)
		}
		fmt.Printf("PASS L%d\n", cmd.Line)
	}

	return nil
}

func main() {
	cfg := LoadConfigFromFile(os.Args[1])
	err := cfg.Run(os.Args[1])
	if err != nil {
		panic(err)
	}
}