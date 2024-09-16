package block

import (
	"context"
	"github.com/dymensionxyz/dymint/node/events"
	uevent "github.com/dymensionxyz/dymint/utils/event"
)

type FreezeHandler struct {
	m *Manager
}

func (f FreezeHandler) HandleFault(ctx context.Context, fault error) {
	uevent.MustPublish(context.TODO(), f.m.Pubsub, &events.DataHealthStatus{Error: fault}, events.HealthStatusList)
}

func NewFreezeHandler(manager *Manager) *FreezeHandler {
	return &FreezeHandler{
		m: manager,
	}
}
