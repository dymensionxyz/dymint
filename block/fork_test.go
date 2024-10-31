package block

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/pubsub"

	"github.com/dymensionxyz/dymint/mocks/github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/mocks/github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/node/events"
	"github.com/dymensionxyz/dymint/types"
	uevent "github.com/dymensionxyz/dymint/utils/event"
)

func TestShouldStopNode(t *testing.T) {
	tests := []struct {
		name     string
		rollapp  *types.Rollapp
		block    *types.Block
		height   uint64
		expected bool
	}{
		{
			name: "should stop - current height greater than revision start height and lower revision",
			rollapp: &types.Rollapp{
				Revision:            2,
				RevisionStartHeight: 100,
			},
			block: &types.Block{
				Header: types.Header{
					Version: types.Version{
						App: 1,
					},
				},
			},
			height:   150,
			expected: true,
		},
		{
			name: "should not stop - current height less than revision start height",
			rollapp: &types.Rollapp{
				Revision:            2,
				RevisionStartHeight: 100,
			},
			block: &types.Block{
				Header: types.Header{
					Version: types.Version{
						App: 1,
					},
				},
			},
			height:   50,
			expected: false,
		},
		{
			name: "should not stop - same revision",
			rollapp: &types.Rollapp{
				Revision:            2,
				RevisionStartHeight: 100,
			},
			block: &types.Block{
				Header: types.Header{
					Version: types.Version{
						App: 2,
					},
				},
			},
			height:   150,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &types.State{}
			state.LastBlockHeight.Store(tt.height)

			logger := log.NewNopLogger()

			manager := &Manager{
				State:  state,
				logger: logger,
			}

			result := manager.shouldStopNode(tt.rollapp, tt.block)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCheckForkUpdate(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		setupMocks    func(*settlement.MockClientI, *store.MockStore, *types.State)
		expectedError bool
	}{
		{
			name: "successful check - no fork update needed",
			setupMocks: func(mockSL *settlement.MockClientI, mockStore *store.MockStore, mockState *types.State) {
				mockState.LastBlockHeight.Store(uint64(100))

				mockSL.On("GetRollapp").Return(&types.Rollapp{
					Revision:            1,
					RevisionStartHeight: 200,
				}, nil)

				mockStore.On("LoadBlock", uint64(100)).Return(&types.Block{
					Header: types.Header{
						Version: types.Version{
							App: 1,
						},
					},
				}, nil)
			},
			expectedError: false,
		},
		{
			name: "error getting rollapp",
			setupMocks: func(mockSL *settlement.MockClientI, mockStore *store.MockStore, mockState *types.State) {
				mockSL.On("GetRollapp").Return((*types.Rollapp)(nil), assert.AnError)
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSL := new(settlement.MockClientI)
			mockStore := new(store.MockStore)
			mockState := &types.State{}

			tt.setupMocks(mockSL, mockStore, mockState)

			manager := &Manager{
				SLClient: mockSL,
				Store:    mockStore,
				State:    mockState,
			}

			err := manager.checkForkUpdate(ctx)
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMonitorForkUpdate(t *testing.T) {
	t.Skip()
	ctx, cancel := context.WithTimeout(context.Background(), 50000*time.Millisecond)
	defer cancel()

	mockSL := new(settlement.MockClientI)
	mockStore := new(store.MockStore)
	state := &types.State{}

	// Setup basic mocks for a successful check
	state.LastBlockHeight.Store(uint64(100))
	mockSL.On("GetRollapp").Return(&types.Rollapp{
		Revision:            2,
		RevisionStartHeight: 100,
	}, nil)

	mockStore.On("LoadBlock", uint64(100)).Return(&types.Block{
		Header: types.Header{
			Height: 100,
			Version: types.Version{
				App: 1,
			},
		},
	}, nil)

	mockSL.On("GetStateInfo", uint64(100)).Return(&types.StateInfo{
		NextProposer: "sequencer1",
	}, nil)

	logger := log.NewNopLogger()

	pubsubServer := pubsub.NewServer()
	go uevent.MustSubscribe(ctx, pubsubServer, "RPCNodeHealthStatusHandler", events.QueryHealthStatus, func(event pubsub.Message) {
		println("event")
	}, logger)

	manager := &Manager{
		SLClient: mockSL,
		Store:    mockStore,
		State:    state,
		logger:   logger,
		RootDir:  t.TempDir(),
		Pubsub:   pubsubServer,
	}

	// Run MonitorForkUpdate in a goroutine since it's a blocking operation
	errCh := make(chan error)
	go func() {
		time.Sleep(1 * time.Second) // Wait for the pubsub server to start
		errCh <- manager.MonitorForkUpdate(ctx)
	}()

	// Wait for context cancellation or error
	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-ctx.Done():
		t.Fatal("MonitorForkUpdate did not return within the context deadline")
	}
}

func TestCreateInstruction(t *testing.T) {
	tests := []struct {
		name          string
		rollapp       *types.Rollapp
		block         *types.Block
		setupMocks    func(*settlement.MockClientI)
		expectedError bool
	}{
		{
			name: "successful instruction creation",
			rollapp: &types.Rollapp{
				Revision:            2,
				RevisionStartHeight: 100,
			},
			block: &types.Block{
				Header: types.Header{
					Height: 150,
				},
			},
			setupMocks: func(mockSL *settlement.MockClientI) {
				mockSL.On("GetStateInfo", uint64(150)).Return(&types.StateInfo{
					NextProposer: "sequencer1",
				}, nil)
			},
			expectedError: false,
		},
		{
			name: "error getting state info",
			rollapp: &types.Rollapp{
				Revision:            2,
				RevisionStartHeight: 100,
			},
			block: &types.Block{
				Header: types.Header{
					Height: 150,
				},
			},
			setupMocks: func(mockSL *settlement.MockClientI) {
				mockSL.On("GetStateInfo", uint64(150)).Return((*types.StateInfo)(nil), assert.AnError)
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSL := new(settlement.MockClientI)
			tt.setupMocks(mockSL)

			manager := &Manager{
				SLClient: mockSL,
				RootDir:  t.TempDir(), // Use temporary directory for testing
			}

			err := manager.createInstruction(tt.rollapp, tt.block)
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
