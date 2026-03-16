package files

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type ResultFile struct {
	Name         string    `json:"name"`
	RelPath      string    `json:"relPath"`
	AbsPath      string    `json:"absPath"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"lastModified"`
}

func ReadDir(dirname string, recursive bool, toolBasePath string) ([]ResultFile, error) {
	files, err := os.ReadDir(dirname)
	if err != nil {
		return nil, fmt.Errorf("cannot read directory %s: %v", dirname, err)
	}

	var result []ResultFile
	for _, file := range files {
		if file.IsDir() && recursive {
			subResults, err := ReadDir(path.Join(dirname, file.Name()), recursive, toolBasePath)
			if err != nil {
				return nil, err
			}
			result = append(result, subResults...)
		} else {
			info, _ := file.Info()
			abs := path.Join(dirname, file.Name())
			rel, _ := filepath.Rel(toolBasePath, abs)
			result = append(result, ResultFile{
				Name:         filepath.Base(file.Name()),
				RelPath:      rel,
				AbsPath:      abs,
				Size:         info.Size(),
				LastModified: info.ModTime(),
			})
		}
	}

	return result, nil
}

type Target string

const (
	TargetIn   Target = "in"
	TargetOut  Target = "out"
	TargetBoth Target = "both"
	TargetAll  Target = "all"
)

func (t Target) Validate() error {
	switch t {
	case TargetIn, TargetOut, TargetBoth, TargetAll:
		return nil
	default:
		return fmt.Errorf("invalid target: %s. Has to be one of 'in', 'out', 'both' or 'all'", t)
	}
}

func Find(pattern, mountBasePath string, target Target) ([]ResultFile, error) {
	if err := target.Validate(); err != nil {
		return nil, err
	}

	matches := make([]ResultFile, 0)
	err := filepath.WalkDir(mountBasePath, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		// match the pattern
		match, err := filepath.Match(pattern, filepath.Base(p))
		if err != nil {
			return err
		}

		if !match {
			return nil
		}
		add := false
		switch target {
		case TargetAll:
			add = true
		case TargetIn:
			add = strings.Contains(p, "/in/")
		case TargetOut:
			add = strings.Contains(p, "/out/")
		case TargetBoth:
			add = strings.Contains(p, "/in/") || strings.Contains(p, "/out/")
		}

		if add {
			info, err := d.Info()
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(mountBasePath, p)
			matches = append(matches, ResultFile{
				Name:         d.Name(),
				AbsPath:      p,
				RelPath:      rel,
				Size:         info.Size(),
				LastModified: info.ModTime(),
			})

		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return matches, nil
}
