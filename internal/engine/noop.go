package engine

import "context"

// Noop is the null adapter for a Format. It satisfies the ports convention:
// when a format is disabled the build wires in a Noop instead of a real engine,
// so callers never special-case a missing renderer. Render is a no-op that
// produces nothing.
type Noop struct {
	format Format
}

// NewNoop returns a Noop renderer for the given format.
func NewNoop(format Format) *Noop { return &Noop{format: format} }

func (n *Noop) Format() Format                              { return n.format }
func (n *Noop) Tool() string                                { return "" }
func (n *Noop) Available() bool                             { return true }
func (n *Noop) Render(context.Context, Job) (string, error) { return "", nil }
