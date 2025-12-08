package workflow

import (
	"bytes"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"io/ioutil"
)

// LoadWorkflows loads one or more workflows from a YAML file. Supports files
// containing multiple YAML documents separated by `---`. Empty documents are
// ignored. Returns
// parsed workflows.
func LoadWorkflows(path string) ([]Workflow, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	dec := yaml.NewDecoder(bytes.NewReader(data))
	var wfs []Workflow
	for {
		var wf Workflow
		if err := dec.Decode(&wf); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		// skip completely empty docs
		if wf.Name == "" && len(wf.Steps) == 0 {
			continue
		}
		wfs = append(wfs, wf)
	}

	if len(wfs) == 0 {
		return nil, fmt.Errorf("no workflows found in %s", path)
	}
	return wfs, nil
}
