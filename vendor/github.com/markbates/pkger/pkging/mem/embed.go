package mem

import (
	"encoding/json"

	"github.com/markbates/pkger/here"
	"github.com/markbates/pkger/internal/maps"
	"github.com/markbates/pkger/pkging"
	"github.com/markbates/pkger/pkging/embed"
)

// MarshalJSON creates a fully re-hydratable JSON representation of *Pkger
func (p *Pkger) MarshalJSON() ([]byte, error) {
	files := map[string]embed.File{}

	p.files.Range(func(key here.Path, file pkging.File) bool {
		f, ok := file.(*File)
		if !ok {
			return true
		}
		ef := embed.File{
			Info:   f.info,
			Here:   f.Here,
			Path:   f.path,
			Parent: f.parent,
			Data:   f.data,
		}
		files[key.String()] = ef
		return true
	})

	infos := map[string]here.Info{}
	p.infos.Range(func(key string, info here.Info) bool {
		infos[key] = info
		return true
	})
	ed := embed.Data{
		Infos: infos,
		Files: files,
		Here:  p.Here,
	}
	return json.Marshal(ed)
}

// UnmarshalJSON re-hydrates the *Pkger
func (p *Pkger) UnmarshalJSON(b []byte) error {
	y := &embed.Data{
		Infos: map[string]here.Info{},
		Files: map[string]embed.File{},
	}

	if err := json.Unmarshal(b, &y); err != nil {
		return err
	}

	p.Here = y.Here
	p.infos = &maps.Infos{}
	for k, v := range y.Infos {
		p.infos.Store(k, v)
	}

	p.files = &maps.Files{}
	for k, v := range y.Files {
		pt, err := p.Parse(k)
		if err != nil {
			return err
		}

		f := &File{
			Here:   v.Here,
			info:   v.Info,
			path:   v.Path,
			data:   v.Data,
			parent: v.Parent,
		}
		p.files.Store(pt, f)
	}
	return nil
}

func UnmarshalEmbed(in []byte) (*Pkger, error) {
	b, err := embed.Decode(in)
	if err != nil {
		return nil, err
	}

	p := &Pkger{}
	if err := json.Unmarshal(b, p); err != nil {
		return nil, err
	}
	return p, nil
}
