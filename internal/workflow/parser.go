package workflow

import (
    "io/ioutil"
    "gopkg.in/yaml.v3"
)

func LoadWorkflow(path string) (*Workflow, error) {
    data, err := ioutil.ReadFile(path)
    if err != nil {
        return nil, err
    }

    var wf Workflow
    if err := yaml.Unmarshal(data, &wf); err != nil {
        return nil, err
    }

    return &wf, nil
}
