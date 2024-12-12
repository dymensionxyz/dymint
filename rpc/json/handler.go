package json

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	tmjson "github.com/tendermint/tendermint/libs/json"

	"github.com/gorilla/rpc/v2"
	"github.com/gorilla/rpc/v2/json2"

	"github.com/dymensionxyz/dymint/types"
)

type handler struct {
	srv    *service
	mux    *http.ServeMux
	codec  rpc.Codec
	logger types.Logger
}

func newHandler(s *service, codec rpc.Codec, logger types.Logger) *handler {
	mux := http.NewServeMux()
	h := &handler{
		srv:    s,
		mux:    mux,
		codec:  codec,
		logger: logger,
	}

	mux.HandleFunc("/", h.serveJSONRPC)
	mux.HandleFunc("/websocket", h.wsHandler)
	for name, method := range s.methods {
		logger.Debug("registering method", "name", name)
		mux.HandleFunc("/"+name, h.newHandler(method))
	}

	return h
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}


func (h *handler) serveJSONRPC(w http.ResponseWriter, r *http.Request) {
	h.serveJSONRPCforWS(w, r, nil)
}



func (h *handler) serveJSONRPCforWS(w http.ResponseWriter, r *http.Request, wsConn *wsConn) {
	
	codecReq := h.codec.NewRequest(r)
	
	method, err := codecReq.Method()
	if err != nil {
		if e, ok := err.(*json2.Error); method == "" && ok && e.Message == "EOF" {
			
			return
		}
		codecReq.WriteError(w, http.StatusBadRequest, err)
		return
	}
	methodSpec, ok := h.srv.methods[method]
	if !ok {
		err := fmt.Errorf("method not found: %s", method)
		codecReq.WriteError(w, int(json2.E_NO_METHOD), err)
		return
	}

	
	args := reflect.New(methodSpec.argsType)
	if errRead := codecReq.ReadRequest(args.Interface()); errRead != nil {
		codecReq.WriteError(w, http.StatusBadRequest, errRead)
		return
	}

	callArgs := []reflect.Value{
		reflect.ValueOf(r),
		args,
	}
	if methodSpec.ws {
		callArgs = append(callArgs, reflect.ValueOf(wsConn))
		rpcID, err := codecReq.ID()
		if err != nil {
			codecReq.WriteError(w, http.StatusBadRequest, err)
			return
		}
		callArgs = append(callArgs, reflect.ValueOf(rpcID))
	}
	rets := methodSpec.m.Call(callArgs)

	
	var errResult error
	statusCode := http.StatusOK
	errInter := rets[1].Interface()
	if errInter != nil {
		statusCode = http.StatusBadRequest
		errResult, _ = errInter.(error)
	}

	
	
	w.Header().Set("x-content-type-options", "nosniff")

	
	if errResult == nil {
		var raw json.RawMessage
		raw, err = tmjson.Marshal(rets[0].Interface())
		if err != nil {
			codecReq.WriteError(w, http.StatusInternalServerError, err)
			return
		}
		codecReq.WriteResponse(w, raw)
	} else {
		codecReq.WriteError(w, statusCode, errResult)
	}
}

func (h *handler) newHandler(methodSpec *method) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		args := reflect.New(methodSpec.argsType)
		values, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			h.encodeAndWriteResponse(w, nil, err, int(json2.E_PARSE))
			return
		}
		for i := 0; i < methodSpec.argsType.NumField(); i++ {
			field := methodSpec.argsType.Field(i)
			nameTag := field.Tag.Get("json")
			name := strings.Split(nameTag, ",")[0]
			isRequired := !strings.Contains(nameTag, "omitempty")
			if isRequired && !values.Has(name) {
				h.encodeAndWriteResponse(w, nil, fmt.Errorf("missing param '%s'", name), int(json2.E_INVALID_REQ))
				return
			}
			if isRequired || values.Has(name) {
				rawVal := values.Get(name)
				var err error
				switch field.Type.Kind() {
				case reflect.Bool:
					err = setBoolParam(rawVal, &args, i)
				case reflect.Int, reflect.Int64:
					err = setIntParam(rawVal, &args, i)
				case reflect.String:
					args.Elem().Field(i).SetString(rawVal)
				case reflect.Slice:
					
					if field.Type.Elem().Kind() == reflect.Uint8 {
						err = setByteSliceParam(rawVal, &args, i)
					}
				default:
					err = errors.New("unknown type")
				}
				if err != nil {
					err = fmt.Errorf("parse param '%s': %w", name, err)
					h.encodeAndWriteResponse(w, nil, err, int(json2.E_PARSE))
					return
				}
			}
		}
		rets := methodSpec.m.Call([]reflect.Value{
			reflect.ValueOf(r),
			args,
		})

		
		statusCode := http.StatusOK
		errInter := rets[1].Interface()
		if errInter != nil {
			statusCode = int(json2.E_INTERNAL)
			err, _ = errInter.(error)
		}

		h.encodeAndWriteResponse(w, rets[0].Interface(), err, statusCode)
	}
}

func (h *handler) encodeAndWriteResponse(w http.ResponseWriter, result interface{}, errResult error, statusCode int) {
	
	
	w.Header().Set("x-content-type-options", "nosniff")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	resp := response{
		Version: "2.0",
		ID:      []byte("-1"),
	}

	if errResult != nil {
		resp.Error = &json2.Error{Code: json2.ErrorCode(statusCode), Data: errResult.Error()}
	} else {
		bytes, err := tmjson.Marshal(result)
		if err != nil {
			resp.Error = &json2.Error{Code: json2.E_INTERNAL, Data: err.Error()}
		} else {
			resp.Result = bytes
		}
	}

	encoder := json.NewEncoder(w)
	err := encoder.Encode(resp)
	if err != nil {
		h.logger.Error("encode RPC response", "error", err)
	}
}

func setBoolParam(rawVal string, args *reflect.Value, i int) error {
	v, err := strconv.ParseBool(rawVal)
	if err != nil {
		return err
	}
	args.Elem().Field(i).SetBool(v)
	return nil
}

func setIntParam(rawVal string, args *reflect.Value, i int) error {
	v, err := strconv.ParseInt(rawVal, 10, 64)
	if err != nil {
		return err
	}
	args.Elem().Field(i).SetInt(v)
	return nil
}

func setByteSliceParam(rawVal string, args *reflect.Value, i int) error {
	b, err := hex.DecodeString(rawVal)
	if err != nil {
		return err
	}
	args.Elem().Field(i).SetBytes(b)
	return nil
}
