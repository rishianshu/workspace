package adapters

import "github.com/antigravity/go-agent-service/internal/agentengine"

// AllowAllPolicy allows every tool.
type AllowAllPolicy struct{}

// NewAllowAllPolicy returns a permissive policy.
func NewAllowAllPolicy() *AllowAllPolicy {
	return &AllowAllPolicy{}
}

// AllowTool implements agentengine.Policy.
func (p *AllowAllPolicy) AllowTool(name string) bool {
	_ = name
	return true
}

var _ agentengine.Policy = (*AllowAllPolicy)(nil)
