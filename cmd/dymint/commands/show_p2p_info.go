package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	rpctypes "github.com/tendermint/tendermint/rpc/jsonrpc/types"
)

var ip, port string // used for flags

type peerInfo struct {
	peerId             string
	multiAddress       string
	connectionDuration time.Duration
}

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
	err = json.Unmarshal(body, &response)
	if err != nil {
		fmt.Println("Error unmarshal:", err)
		return nil
	}
	var netinfo map[string]interface{}

	err = json.Unmarshal([]byte(response.Result), &netinfo)
	if err != nil {
		fmt.Println("Error unmarshal:", err)
		return nil
	}
	listeners := netinfo["listeners"].([]interface{})
	fmt.Println("Host ID:", listeners[0])
	fmt.Println("Listening P2P addresses:", listeners[1:])

	var peers []peerInfo

	//peer info is stored from the rpc response
	if netinfo["peers"] != nil {

		ps := netinfo["peers"].([]interface{})

		for _, p := range ps {

			peer := p.(map[string]interface{})
			info := peer["node_info"].(map[string]interface{})
			status := peer["connection_status"].(map[string]interface{})

			//duration is in int64 nanoseconds
			duration, err := strconv.ParseInt(status["Duration"].(string), 10, 64)
			if err != nil {
				fmt.Println("Error parsing connection duration:", err)
			}

			newPeer := peerInfo{
				peerId:             info["id"].(string),
				multiAddress:       peer["remote_ip"].(string),
				connectionDuration: time.Duration(duration),
			}
			peers = append(peers, newPeer)
		}
	} else {
		fmt.Println("0 peers connected")
	}

	//Peers ordered by oldest connection
	sort.Slice(peers[:], func(i, j int) bool {
		return peers[i].connectionDuration > peers[j].connectionDuration
	})

	//Info displayed: Libp2p Peeer ID, Multiaddress (connection info) and time pasted since connection
	for i, p := range peers {
		fmt.Printf("Peer %d Id:%s Multiaddress:%s Connection duration:%s\n", i, p.peerId, p.multiAddress, p.connectionDuration)
	}

	return nil
}
