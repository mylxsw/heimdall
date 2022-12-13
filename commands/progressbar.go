package commands

import (
	"github.com/schollz/progressbar/v3"
)

type Progressbar struct {
	*progressbar.ProgressBar
	enabled bool
}

func NewProgressbar(enabled bool, description string) *Progressbar {
	p := Progressbar{enabled: enabled}
	if enabled {
		p.ProgressBar = progressbar.Default(-1, description)
	}

	return &p
}

func (p *Progressbar) Add(count int) {
	if p.ProgressBar != nil {
		p.ProgressBar.Add(count)
	}
}

func (p *Progressbar) Describe(description string) {
	if p.ProgressBar != nil {
		p.ProgressBar.Describe(description)
	}
}

func (p *Progressbar) Clear() {
	if p.ProgressBar != nil {
		p.ProgressBar.Clear()
	}
}
