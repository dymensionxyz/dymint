package sequencer

import (
	"gopkg.in/yaml.v2"
)

func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}
