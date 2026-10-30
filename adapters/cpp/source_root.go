package cpp

import (
	"fmt"
	"os"
	"path/filepath"
)

func (d *document) supplementSources(path string) error {
	if path == "" {
		return nil
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return fmt.Errorf("open C++ source root: %w", err)
	}
	defer root.Close()
	for _, method := range d.Methods {
		if err := d.supplementMethod(root, method); err != nil {
			return err
		}
	}
	return nil
}

func (d *document) supplementMethod(root *os.Root, method *vertex) error {
	name := method.text("FILENAME")
	file := d.Files[name]
	if file.embedded() {
		return nil
	}
	file, err := d.supplementFile(root, name)
	if err != nil {
		return err
	}
	p := projection{method: method, file: file, code: file.content}
	if _, ok := p.span(method); !ok {
		return fmt.Errorf("C++ source %s does not match exported method %s", name, method.text("FULL_NAME"))
	}
	return nil
}

func (d *document) supplementFile(root *os.Root, name string) (*sourceFile, error) {
	if file := d.Files[name]; file != nil && file.content != "" {
		return file, nil
	}
	file, err := sourceFromRoot(root, name)
	if err != nil {
		return nil, err
	}
	d.Files[name] = file
	return file, nil
}

func (f *sourceFile) embedded() bool {
	return f != nil && f.content != "" && !f.supplemental
}

func sourceFromRoot(root *os.Root, name string) (*sourceFile, error) {
	if !filepath.IsLocal(name) {
		return nil, fmt.Errorf("C++ source path must stay inside --cpg-source: %q", name)
	}
	content, err := root.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("read C++ source %s: %w", name, err)
	}
	file := newSourceFile(string(content))
	file.supplemental = true
	return file, nil
}
