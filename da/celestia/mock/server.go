package mock

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"time"

	mux2 "github.com/gorilla/mux"
	"github.com/tendermint/tendermint/libs/pubsub"

	"github.com/celestiaorg/go-cnc"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/local"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
)

// Server mocks celestia-node HTTP API.
type Server struct {
	da        *local.DataAvailabilityLayerClient
	blockTime time.Duration
	server    *http.Server
	logger    types.Logger
}

// NewServer creates new instance of Server.
func NewServer(blockTime time.Duration, logger types.Logger) *Server {
	return &Server{
		da:        new(local.DataAvailabilityLayerClient),
		blockTime: blockTime,
		logger:    logger,
	}
}

// Start starts HTTP server with given listener.
func (s *Server) Start(listener net.Listener) error {
	err := s.da.Init([]byte(s.blockTime.String()), pubsub.NewServer(), store.NewDefaultInMemoryKVStore(), s.logger)
	if err != nil {
		return err
	}
	err = s.da.Start()
	if err != nil {
		return err
	}
	s.server = new(http.Server)
	s.server.Handler = s.getHandler()
	go func() {
		err := s.server.Serve(listener)
		s.logger.Debug("http server exited with", "error", err)
	}()
	return nil
}

// Stop shuts down the Server.
func (s *Server) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_ = s.server.Shutdown(ctx)
}

func (s *Server) getHandler() http.Handler {
	mux := mux2.NewRouter()
	mux.HandleFunc("/submit_pfb", s.submit).Methods(http.MethodPost)
	mux.HandleFunc("/namespaced_shares/{namespace}/height/{height}", s.shares).Methods(http.MethodGet)
	mux.HandleFunc("/namespaced_data/{namespace}/height/{height}", s.data).Methods(http.MethodGet)

	return mux
}

func (s *Server) submit(w http.ResponseWriter, r *http.Request) {
	req := cnc.SubmitPFBRequest{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		s.writeError(w, err)
		return
	}

	batch := types.Batch{}
	batchData, err := hex.DecodeString(req.Data)
	if err != nil {
		s.writeError(w, err)
		return
	}
	err = batch.UnmarshalBinary(batchData)
	if err != nil {
		s.writeError(w, err)
		return
	}

	res := s.da.SubmitBatch(&batch)
	code := 0
	if res.Code != da.StatusSuccess {
		code = 3
	}

	resp, err := json.Marshal(cnc.TxResponse{
		Height: int64(res.SubmitMetaData.Height),
		Code:   uint32(code),
		RawLog: res.Message,
	})
	if err != nil {
		s.writeError(w, err)
		return
	}

	s.writeResponse(w, resp)
}

func (s *Server) shares(w http.ResponseWriter, r *http.Request) {
	height, err := parseHeight(r)
	if err != nil {
		s.writeError(w, err)
		return
	}

	daMetaData := &da.DASubmitMetaData{
		Height: height,
	}
	res := s.da.RetrieveBatches(daMetaData)
	if res.Code != da.StatusSuccess {
		s.writeError(w, errors.New(res.Message))
		return
	}

	var nShares []NamespacedShare
	for _, batch := range res.Batches {
		blob, err := json.Marshal(batch)
		if err != nil {
			s.writeError(w, err)
			return
		}
		delimited, err := marshalDelimited(blob)
		if err != nil {
			s.writeError(w, err)
		}
		nShares = appendToShares(nShares, []byte{1, 2, 3, 4, 5, 6, 7, 8}, delimited)
	}
	shares := make([]Share, len(nShares))
	for i := range nShares {
		shares[i] = nShares[i].Share
	}

	resp, err := json.Marshal(namespacedSharesResponse{
		Shares: shares,
		Height: res.CheckMetaData.Height,
	})
	if err != nil {
		s.writeError(w, err)
		return
	}

	s.writeResponse(w, resp)
}

func (s *Server) data(w http.ResponseWriter, r *http.Request) {
	height, err := parseHeight(r)
	if err != nil {
		s.writeError(w, err)
		return
	}
	daMetaData := &da.DASubmitMetaData{
		Height: height,
	}
	res := s.da.RetrieveBatches(daMetaData)
	if res.Code != da.StatusSuccess {
		s.writeError(w, errors.New(res.Message))
		return
	}

	data := make([][]byte, len(res.Batches))
	for i := range res.Batches {
		data[i], err = res.Batches[i].MarshalBinary()
		if err != nil {
			s.writeError(w, err)
			return
		}
	}

	resp, err := json.Marshal(namespacedDataResponse{
		Data:   data,
		Height: res.CheckMetaData.Height,
	})
	if err != nil {
		s.writeError(w, err)
		return
	}

	s.writeResponse(w, resp)
}

func parseHeight(r *http.Request) (uint64, error) {
	vars := mux2.Vars(r)

	height, err := strconv.ParseUint(vars["height"], 10, 64)
	if err != nil {
		return 0, err
	}
	return height, nil
}

func (s *Server) writeResponse(w http.ResponseWriter, payload []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write(payload)
	if err != nil {
		s.logger.Error("failed to write response", "error", err)
	}
}

func (s *Server) writeError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	resp, jerr := json.Marshal(err.Error())
	if jerr != nil {
		s.logger.Error("failed to serialize error message", "error", jerr)
	}
	_, werr := w.Write(resp)
	if werr != nil {
		s.logger.Error("failed to write response", "error", werr)
	}
}
