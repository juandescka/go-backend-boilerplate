package health

import "sync/atomic"

// LifecycleState records whether the process is draining and should refuse readiness.
type LifecycleState struct {
	shuttingDown atomic.Bool
}

func NewLifecycleState() *LifecycleState { return &LifecycleState{} }

func (s *LifecycleState) BeginShutdown() {
	if s != nil {
		s.shuttingDown.Store(true)
	}
}

func (s *LifecycleState) IsShuttingDown() bool {
	return s != nil && s.shuttingDown.Load()
}
