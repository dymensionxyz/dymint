package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/server"
	"github.com/dymensionxyz/dymint/conv"
	"github.com/libp2p/go-libp2p"
	"github.com/spf13/cobra"
	"github.com/tendermint/tendermint/p2p"
	rpctypes "github.com/tendermint/tendermint/rpc/jsonrpc/types"
)

var ip, port string // used for flags

// ShowNodeIDCmd dumps node's ID to the standard output.
var ShowP2PInfoCmd = &cobra.Command{
	Use:     "show-p2p-info",
	Aliases: []string{"show_p2p_info"},
	Short:   "Show P2P status information",
	RunE:    showP2PInfo,
}

func init() {
	ShowP2PInfoCmd.Flags().StringVar(&ip, "ip", "127.0.0.1", "rpc ip address")
	ShowP2PInfoCmd.Flags().StringVar(&port, "port", "26657", "rpc port")
}

func showP2PInfo(cmd *cobra.Command, args []string) error {

	serverCtx := server.GetServerContextFromCmd(cmd)
	cfg := serverCtx.Config
	nodeKey, err := p2p.LoadNodeKey(cfg.NodeKeyFile())
	if err != nil {
		return err
	}
	signingKey, err := conv.GetNodeKey(nodeKey)
	if err != nil {
		return err
	}
	// convert nodeKey to libp2p key
	host, err := libp2p.New(libp2p.Identity(signingKey))
	if err != nil {
		return err
	}
	data, err := json.Marshal(map[string]interface{}{
		"method":  "net_info",
		"jsonrpc": "2.0",
		"id":      1,
		"params":  []interface{}{},
	})
	if err != nil {
		log.Fatalf("Marshal: %v", err)
	}
	resp, err := http.Post("http://"+ip+":"+port, "application/json", strings.NewReader(string(data)))
	if err != nil {
		fmt.Println("Error connecting to Dymint RPC")
		return nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading data from RPC:", err)
		return nil
	}
	response := rpctypes.RPCResponse{}
	//result := make(map[string]interface{})
	err = json.Unmarshal(body, &response)
	if err != nil {
		fmt.Println("Error unmarshal:", err)
		return nil
	}
	var netinfo map[string]interface{}

	//netinfo := ctypes.ResultNetInfo{}
	err = json.Unmarshal([]byte(response.Result), &netinfo)
	if err != nil {
		fmt.Println("Error unmarshal:", err)
		return nil
	}
	fmt.Println("Host ID:", host.ID())
	fmt.Println("Listening P2P addresses:", netinfo["listeners"])

	if netinfo["peers"] != nil {

		peers := netinfo["peers"].([]interface{})

		for i, p := range peers {

			peer := p.(map[string]interface{})
			info := peer["node_info"].(map[string]interface{})
			status := peer["connection_status"].(map[string]interface{})

			duration, err := strconv.ParseInt(status["Duration"].(string), 10, 64)
			if err != nil {
				fmt.Println("Error parsing connection duration:", err)
			}
			fmt.Printf("Peer %d Id:%s Multiaddress:%s Duration:%s\n", i, info["id"], peer["remote_ip"], time.Duration(duration))
		}
	} else {
		fmt.Println("0 peers connected")
	}

	return nil
}
