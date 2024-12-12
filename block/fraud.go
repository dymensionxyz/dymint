package block

import (
	"context"
)



type FraudHandler interface {
	
	
	HandleFault(ctx context.Context, fault error)
}



type FreezeHandler struct {
	m *Manager
}

func (f FreezeHandler) HandleFault(ctx context.Context, fault error) {
	f.m.freezeNode(fault)
}

func NewFreezeHandler(manager *Manager) *FreezeHandler {
	return &FreezeHandler{
		m: manager,
	}
}
