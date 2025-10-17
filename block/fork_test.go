package block

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/pubsub"
	tmstate "github.com/tendermint/tendermint/proto/tendermint/state"
	"github.com/tendermint/tendermint/proto/tendermint/version"
	tmversion "github.com/tendermint/tendermint/proto/tendermint/version"

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
		runMode  uint
		expected bool
	}{
		{
			name: "should stop - current height greater than revision start height and lower revision",
			rollapp: &types.Rollapp{
				Revisions: []types.Revision{{
					Revision:    tmstate.Version{Consensus: tmversion.Consensus{App: 2}},
					StartHeight: 100,
				}},
			},
			block: &types.Block{
				Header: types.Header{
					Version: types.Version{
						App: 1,
					},
				},
			},
			height:   150,
			runMode:  RunModeFullNode,
			expected: true,
		},
		{
			name: "should not stop - current height less than revision start height",
			rollapp: &types.Rollapp{
				Revisions: []types.Revision{{
					Revision:    tmstate.Version{Consensus: tmversion.Consensus{App: 2}},
					StartHeight: 100,
				}},
			},
			block: &types.Block{
				Header: types.Header{
					Version: types.Version{
						App: 1,
					},
				},
			},
			height:   50,
			runMode:  RunModeFullNode,
			expected: false,
		},
		{
			name: "should not stop - same revision",
			rollapp: &types.Rollapp{
				Revisions: []types.Revision{{
					Revision:    tmstate.Version{Consensus: tmversion.Consensus{App: 2}},
					StartHeight: 100,
				}},
			},
			block: &types.Block{
				Header: types.Header{
					Version: types.Version{
						App: 2,
					},
				},
			},
			height:   150,
			runMode:  RunModeFullNode,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedRevision := tt.rollapp.GetRevisionForHeight(tt.height)
			result := shouldStopNode(expectedRevision, tt.height, tt.block.Header.Version.App)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCheckForkUpdate(t *testing.T) {
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
					Revisions: []types.Revision{
						{
							Revision:    tmstate.Version{Consensus: tmversion.Consensus{App: 2}},
							StartHeight: 100,
						},
					},
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
			mockState := &types.State{
				Revisions: []types.Revision{
					{Revision: tmstate.Version{Consensus: version.Consensus{App: 1, Block: 11}}, StartHeight: 0},
					{Revision: tmstate.Version{Consensus: version.Consensus{App: 2, Block: 11}}, StartHeight: 101},
				},
			}

			tt.setupMocks(mockSL, mockStore, mockState)

			manager := &Manager{
				SLClient: mockSL,
				Store:    mockStore,
				State:    mockState,
			}

			err := manager.checkForkUpdate("")
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
		Revisions: []types.Revision{{
			Revision:    tmstate.Version{Consensus: tmversion.Consensus{App: 2}},
			StartHeight: 100,
		}},
	}, nil)

	mockStore.On("LoadBlock", uint64(100)).Return(&types.Block{
		Header: types.Header{
			Height: 100,
			Version: types.Version{
				App: 1,
			},
		},
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
		errCh <- manager.MonitorForkUpdateLoop(ctx)
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
		expectedError bool
	}{
		{
			name: "successful instruction creation",
			rollapp: &types.Rollapp{
				Revisions: []types.Revision{{
					Revision:    tmstate.Version{Consensus: tmversion.Consensus{App: 2}},
					StartHeight: 100,
				}},
			},
			block: &types.Block{
				Header: types.Header{
					Height: 150,
				},
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &Manager{
				RootDir: t.TempDir(), // Use temporary directory for testing
			}
			mockSL := new(settlement.MockClientI)
			mockSL.On("GetObsoleteDrs").Return([]uint32{}, nil)
			mockSL.On("GetRollapp").Return(&types.Rollapp{
				Revisions: []types.Revision{{
					Revision:    tmstate.Version{Consensus: tmversion.Consensus{App: 2}},
					StartHeight: 100,
				}},
			}, nil)

			manager.SLClient = mockSL
			expectedRevision := tt.rollapp.GetRevisionForHeight(tt.block.Header.Height)
			_, err := manager.createInstruction(expectedRevision)
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
