package pipeline

var Symfind2 = &Pipeline{
	Stages: []Stage{
		&BuildInfo{},
		&Configure{},
		&Data{},
		&AnnotateTree{},
		// TreeGrid (needed?)
		// TagParticles
		// FindCores
		// FindSubhaloes
		// PostProcess (needed?)
	},

	Datasets: []Dataset{
		&Config{},
		&InputTree{},
		&Tree{},
	},
}
