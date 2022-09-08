package settlement

import (
	"context"
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/types/query"
	rollapptypes "github.com/dymensionxyz/dymension/x/rollapp/types"

	// cosmospb "github.com/dymensionxyz/dymint/types/pb/cosmos"

	"github.com/ignite/cli/ignite/pkg/cosmosaccount"
	"github.com/ignite/cli/ignite/pkg/cosmosclient"
)

func TestSendTx(t *testing.T) {

	addressPrefix := "dym"

	// Create a Cosmos client instance
	cosmos, err := cosmosclient.New(
		// TODO: What about gas adjustment + gas limit?
		context.Background(),
		cosmosclient.WithAddressPrefix(addressPrefix),
		cosmosclient.WithNodeAddress("http://localhost:26657"),
		cosmosclient.WithKeyringBackend(cosmosaccount.KeyringTest),
		// cosmosclient.WithHome("/Users/omridagan/.dymension"),
	)

	if err != nil {
		t.Log(err)
	}

	// Account `alice` was initialized during `ignite chain serve`
	// accountName := "bob"

	// Get account from the keyring
	// account, err := cosmos.Account(accountName)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// addr := account.Address("dym")

	// Define a message to create a post
	// blockDescriptors := &rollapptypes.BlockDescriptors{[]rollapptypes.BlockDescriptor{{Height: 1}}}
	// msg := rollapptypes.NewMsgUpdateState(addr, "rollapp2", 1, 1, "", 0, blockDescriptors)

	// Broadcast a transaction from account `alice` with the message
	// to create a post store response in txResp
	// txResp, err := cosmos.BroadcastTx(accountName, msg)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// // Print response from broadcasting a transaction
	// fmt.Print("MsgCreatePost:\n\n")
	// fmt.Println(txResp)

	// Instantiate a query client for your `blog` blockchain
	queryClient := rollapptypes.NewQueryClient(cosmos.Context())
	queryResp, _ := queryClient.StateInfoAll(context.Background(), &rollapptypes.QueryAllStateInfoRequest{&query.PageRequest{Limit: 1, Reverse: true}})
	fmt.Println(queryResp)

	// Query the blockchain using the client's `Posts` method
	// to get all posts store all posts in queryResp
	// queryResp, err := queryClient.Posts(context.Background(), &types.QueryPostsRequest{})
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// // Print response from querying all the posts
	// fmt.Print("\n\nAll posts:\n\n")
	// fmt.Println(queryResp)

}
