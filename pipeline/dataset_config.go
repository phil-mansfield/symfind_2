package pipeline

import (
	"fmt"
	"reflect"

	"github.com/phil-mansfield/chimera/config"
	"github.com/phil-mansfield/chimera/hdf5"
)

// Config is a Dataset
type Config struct {
	BaseDataset
}

var _ Dataset = &Config{}

type ConfigArgs struct {
}

type ConfigData struct {
	L float64 // cMpc/h

	TreeFileName string
	TreeFileN    int64

	TreeConfigFileName string
	TreeConfigText     string // Derived - TreeConfigFileName

}

func (dset *Config) Init(pipe *Pipeline) {
	dset.BaseDataset = BaseDataset{
		Pipeline: pipe,
		DsetName: "Config",
		Dep:      "Configure",
	}
}

func (dset *Config) Configure(cfg *ConfigData) error {
	// Config.Configure() is a no-op, all the initialisation is done in
	// Configure.Run().
	return nil
}

func (dset *Config) Read(fname string, genArgs, genData any) error {
	args, data, err := convertConfigTypes(genArgs, genData)
	if err != nil {
		return err
	}

	_ = args // args is not used by Read()

	file, err := hdf5.Open(fname, "r")
	if err != nil {
		return err
	}
	defer file.Close()

	return config.LoopByField(func(name string, field reflect.Value, ptr any) error {
		switch val := ptr.(type) {
		case *int64, *float64, *bool:
			return file.GetAttr("Config", name, ptr)
		case *[]int64:
			return genericGetAttr(file, "Config", name, val)
		case *[]float64:
			return genericGetAttr(file, "Config", name, val)
		case *string:
			var b []byte
			err := genericGetAttr(file, "Config", name, &b)
			if err != nil {
				return err
			}
			*val = string(b)
		default:
			return fmt.Errorf("Internal error: ConfigData has field %s had unsupported type %T", name, ptr)
		}

		return nil
	}, data)
}

func genericGetAttr[T any](file *hdf5.File, grp, name string, out *[]T) error {
	dim, err := file.AttrDim(grp, name)
	if err != nil {
		return nil
	}

	data := make([]T, dim[0])
	file.GetAttr(grp, name, data)
	*out = data

	return nil
}

func (dset *Config) Write(fname string, genArgs, genData any) error {
	args, data, err := convertConfigTypes(genArgs, genData)
	if err != nil {
		return err
	}

	_ = args // args not used by Config.Write

	// Config file already exists
	file, err := hdf5.Open(fname, "w")
	if err != nil {
		return err
	}
	defer file.Close()

	return config.LoopByField(func(name string, field reflect.Value, ptr any) error {
		switch val := ptr.(type) {
		case *float64, *int64, *bool:
			return file.Attr("Config", name, ptr)
		case *[]float64:
			return file.Attr("Config", name, *val)
		case *[]int64:
			return file.Attr("Config", name, *val)
		case *string:
			return file.Attr("Config", name, []byte(*val))
		default:
			return fmt.Errorf("Internal error: ConfigData has field %s had unsupported type %T", name, ptr)
		}
	}, data)
}

func convertConfigTypes(genArgs, genData any) (*ConfigArgs, *ConfigData, error) {
	var (
		args *ConfigArgs
		data *ConfigData
		ok   bool
	)

	if args, ok = genArgs.(*ConfigArgs); !ok {
		return nil, nil, fmt.Errorf("Internal error: Config arguments must be of type *ConfigDatasetArgs, but got type %T instead.", args)
	}
	if data, ok = genData.(*ConfigData); !ok {
		return nil, nil, fmt.Errorf("Internal error: Config data must be of type *ConfigDatasetData, but got type %T instead.", args)
	}

	return args, data, nil
}
