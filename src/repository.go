package src

import (
	data "mygit/src/database"
	"path/filepath"
)

type Repository struct {
	w *WorkSpace
	d *data.Database
	r *data.Refs
	i *data.Index
}

func GenerateRepository(rootPath, gitPath, dbPath string) *Repository {
	wk := &WorkSpace{
		Path: rootPath,
	}

	r := &data.Refs{
		Path: gitPath,
	}

	d := &data.Database{
		Path: dbPath,
	}

	i := data.GenerateIndex(filepath.Join(gitPath, "index"))

	return &Repository{
		w: wk,
		d: d,
		r: r,
		i: i,
	}
}
