package dymension

import (
	"fmt"
	"testing"

	"github.com/dymensionxyz/cosmosclient/cosmosclient"
	"github.com/ignite/cli/ignite/pkg/cosmosaccount"
	"github.com/stretchr/testify/assert"
)

func Test_cosmosClient_GetAccount(t *testing.T) {
	type fields struct {
		Client cosmosclient.Client
	}
	type args struct {
		accountName string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    cosmosaccount.Account
		wantErr assert.ErrorAssertionFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &cosmosClient{
				Client: tt.fields.Client,
			}
			got, err := c.GetAccount(tt.args.accountName)
			if !tt.wantErr(t, err, fmt.Sprintf("GetAccount(%v)", tt.args.accountName)) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetAccount(%v)", tt.args.accountName)
		})
	}
}
