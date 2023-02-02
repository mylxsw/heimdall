package commands

import (
	"github.com/schollz/progressbar/v3"
)

type Progressbar struct {
	*progressbar.ProgressBar
	enabled bool
}

func NewProgressbarWithMax(enabled bool, description string, max int64) *Progressbar {
	p := Progressbar{enabled: enabled}
	if enabled {
		p.ProgressBar = progressbar.Default(max, description)
	}

	return &p
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

func (p *Progressbar) Reset(max int, desc string) {
	if p.ProgressBar != nil {
		p.ProgressBar.Reset()
		p.ProgressBar.Clear()
		p.ProgressBar.ChangeMax(max)
		p.ProgressBar.Describe(desc)
	}
}

func (p *Progressbar) Close() {
	if p.ProgressBar != nil {
		p.ProgressBar.Finish()
	}
}
