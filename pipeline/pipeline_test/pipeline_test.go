package pipeline_test

// Not a real test, a sanity check

import (
	"fmt"
	"testing"

	"github.com/phil-mansfield/symfind_2/pipeline"
)

func TestWrite(t *testing.T) {
	dset := &pipeline.Config{}
	args, data := &pipeline.ConfigArgs{}, &pipeline.ConfigData{10.0, -5, "meow", true}

	err := dset.Write("test_config.h5", args, data)

	if err != nil {
		panic(err.Error())
	}

	args, data = &pipeline.ConfigArgs{}, &pipeline.ConfigData{}

	err = dset.Read("test_config.h5", args, data)
	if err != nil {
		panic(err.Error())
	}

	fmt.Println(data)

}
