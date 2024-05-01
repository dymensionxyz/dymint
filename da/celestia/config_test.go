package celestia

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	uretry "github.com/dymensionxyz/dymint/utils/retry"

	"github.com/stretchr/testify/assert"
)

func mustMarshal(v any) []byte {
	bz, _ := json.Marshal(v)
	return bz
}

func Test_createConfig(t *testing.T) {
	type args struct {
		bz []byte
	}
	tests := []struct {
		name    string
		args    args
		wantC   Config
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "empty",
			args: args{
				bz: mustMarshal(Config{
					BaseURL:       TestConfig.BaseURL,
					AppNodeURL:    TestConfig.AppNodeURL,
					Timeout:       TestConfig.Timeout,
					Fee:           0,
					GasPrices:     42,
					GasAdjustment: 42,
					GasLimit:      42,
					Backoff:       uretry.NewBackoffConfig(),
					RetryAttempts: 10,
					RetryDelay:    10 * time.Second,
					,
				}),
			},
			wantC: Config{
				BaseURL:       TestConfig.BaseURL,
				AppNodeURL:    TestConfig.AppNodeURL,
				Timeout:       TestConfig.Timeout,
				Fee:           0,
				GasPrices:     42,
				GasAdjustment: 42,
				GasLimit:      42,
				Backoff:       uretry.NewBackoffConfig(),
				RetryAttempts: 10,
				RetryDelay:    10 * time.Second,
				NamespaceID:   TestConfig.NamespaceID,
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotC, err := createConfig(tt.args.bz)
			if !tt.wantErr(t, err, fmt.Sprintf("createConfig(%v)", tt.args.bz)) {
				return
			}
			assert.Equalf(t, tt.wantC, gotC, "createConfig(%v)", tt.args.bz)
		})
	}
}
