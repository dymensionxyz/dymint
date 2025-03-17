package near

import (
	near "github.com/near/rollup-data-availability/gopkg/da-rpc"
)

type NearClient interface {
	SubmitData(data []byte) ([]byte, error)
	GetBlob(frameRef []byte) ([]byte, error)
	GetAccountAddress() string
}

type NearConfig struct {
	Enable    bool
	Account   string
	Network   string
	Contract  string
	Key       string
	Namespace uint32
}

type Client struct {
	near near.Config
}

var _ NearClient = &Client{}

// NewClient returns a DA avail client
func NewClient(config NearConfig) (NearClient, error) {

	near, err := near.NewConfig(config.Account, config.Contract, config.Key, config.Network, config.Namespace)
	if err != nil {
		return nil, err
	}

	client := Client{
		near: *near,
	}
	return client, nil
}

// SubmitData sends blob data to Avail DA
func (c Client) SubmitData(data []byte) ([]byte, error) {
	frameRefBytes, err := c.near.ForceSubmit(data)
	if err != nil {
		return nil, err
	}

	return frameRefBytes, nil
}

// GetBlock retrieves a block from Near chain by block hash
func (c Client) GetBlob(frameRef []byte) ([]byte, error) {
	blob, err := c.near.Get(frameRef, 0)
	if err != nil {
		return nil, err
	}

	return blob, nil
}

// GetAccountAddress returns configured account address
func (c Client) GetAccountAddress() string {
	return ""
}
