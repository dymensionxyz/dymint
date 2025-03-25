package bnb

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dymensionxyz/gerr-cosmos/gerrc"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"
	"golang.org/x/crypto/sha3"
)

// Transaction types.
const (
	LegacyTxType     = 0x00
	AccessListTxType = 0x01
	DynamicFeeTxType = 0x02
	BlobTxType       = 0x03
	SetCodeTxType    = 0x04
)

const (
	// BloomByteLength represents the number of bytes used in a header log bloom.
	BloomByteLength = 256

	// BloomBitLength represents the number of bits used in a header log bloom.
	BloomBitLength = 8 * BloomByteLength
)

type BlockNonce [8]byte

type Bloom [BloomByteLength]byte

type Header struct {
	ParentHash  common.Hash    `json:"parentHash"       gencodec:"required"`
	UncleHash   common.Hash    `json:"sha3Uncles"       gencodec:"required"`
	Coinbase    common.Address `json:"miner"`
	Root        common.Hash    `json:"stateRoot"        gencodec:"required"`
	TxHash      common.Hash    `json:"transactionsRoot" gencodec:"required"`
	ReceiptHash common.Hash    `json:"receiptsRoot"     gencodec:"required"`
	Bloom       Bloom          `json:"logsBloom"        gencodec:"required"`
	Difficulty  *big.Int       `json:"difficulty"       gencodec:"required"`
	Number      *big.Int       `json:"number"           gencodec:"required"`
	GasLimit    uint64         `json:"gasLimit"         gencodec:"required"`
	GasUsed     uint64         `json:"gasUsed"          gencodec:"required"`
	Time        uint64         `json:"timestamp"        gencodec:"required"`
	Extra       []byte         `json:"extraData"        gencodec:"required"`
	MixDigest   common.Hash    `json:"mixHash"`
	Nonce       BlockNonce     `json:"nonce"`

	// BaseFee was added by EIP-1559 and is ignored in legacy headers.
	BaseFee *big.Int `json:"baseFeePerGas" rlp:"optional"`

	// WithdrawalsHash was added by EIP-4895 and is ignored in legacy headers.
	WithdrawalsHash *common.Hash `json:"withdrawalsRoot" rlp:"optional"`

	// BlobGasUsed was added by EIP-4844 and is ignored in legacy headers.
	BlobGasUsed *uint64 `json:"blobGasUsed" rlp:"optional"`

	// ExcessBlobGas was added by EIP-4844 and is ignored in legacy headers.
	ExcessBlobGas *uint64 `json:"excessBlobGas" rlp:"optional"`

	// ParentBeaconRoot was added by EIP-4788 and is ignored in legacy headers.
	ParentBeaconRoot *common.Hash `json:"parentBeaconBlockRoot" rlp:"optional"`
}

type BNBClient interface {
	SubmitBlob(blob []byte) ([]byte, error)
	GetBlob(frameRef []byte) ([]byte, error)
	GetAccountAddress() common.Address
}

// WeaveVM RPC client
type Client struct {
	client    *ethclient.Client
	rpcClient rpc.Client
	cfg       *Config
	ctx       context.Context
}

type Config struct {
	ChainId    *big.Int
	PrivateKey string
	From       common.Address
}

var _ BNBClient = &Client{}

func NewClient(ctx context.Context, config BNBConfig) (BNBClient, error) {

	rpcClient, err := rpc.DialContext(ctx, config.Endpoint)
	if err != nil {
		return nil, err
	}
	cfg := &Config{
		ChainId:    new(big.Int).SetUint64(config.ChainId),
		PrivateKey: config.PrivateKey,
	}
	client := Client{
		client:    ethclient.NewClient(rpcClient),
		rpcClient: *rpcClient,
		ctx:       ctx,
		cfg:       cfg,
	}

	from, err := client.getAddressFromPrivateKey()
	if err != nil {
		return nil, err
	}
	client.cfg.From = from
	return client, nil
}

// GetAccountAddress provides a mock function with no fields
func (c Client) GetAccountAddress() common.Address {
	return c.cfg.From
}

// GetBlock retrieves a block from Near chain by block hash
func (c Client) GetBlob(frameRef []byte) ([]byte, error) {
	/*var txSidecars []*BlobTxSidecar
	number := rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(blockID))
	err := c.rpcClient.CallContext(c.ctx, &txSidecars, "eth_getBlobSidecars", number.String())
	if err != nil {
		return nil, err
	}
	if txSidecars == nil {
		return nil, gerrc.ErrNotFound
	}
	idx := 0
	for _, txSidecar := range txSidecars {
		txIndex, err := util.HexToUint64(txSidecar.TxIndex)
		if err != nil {
			return nil, err
		}
		for j := range txSidecar.BlobSidecar.Blobs {
			sidecars = append(sidecars,
				&types2.GeneralSideCar{
					Sidecar: structs.Sidecar{
						Index:         strconv.Itoa(idx),
						Blob:          txSidecar.BlobSidecar.Blobs[j],
						KzgCommitment: txSidecar.BlobSidecar.Commitments[j],
						KzgProof:      txSidecar.BlobSidecar.Proofs[j],
					},
					TxIndex: int64(txIndex),
					TxHash:  txSidecar.TxHash,
				},
			)
			idx++
		}
	}
	return sidecars, err*/
	return nil, nil
}

func (c Client) SubmitBlob(blob []byte) ([]byte, error) {

	fmt.Println("from: ", c.cfg.From)
	balance, err := c.BalanceAt(c.ctx, c.cfg.From)
	if err != nil {
		return nil, err
	}
	fmt.Println("balance:", balance)
	nonce, err := c.client.PendingNonceAt(c.ctx, c.GetAccountAddress())
	if err != nil {
		return nil, err
	}

	gasTipCap := big.NewInt(2 * params.GWei)
	//pendingHeader, err := c.HeaderByNumber(c.ctx)
	pendingHeader, err := c.client.HeaderByNumber(c.ctx, big.NewInt(-1))
	if err != nil {
		return nil, err
	}
	gasFeeCap := new(big.Int).Add(gasTipCap, new(big.Int).Mul(pendingHeader.BaseFee, big.NewInt(2)))

	var txData TxData
	//if s.l2BatcherCfg.DataAvailabilityType == batcherFlags.CalldataType {
	/*rawTx := &types.DynamicFeeTx{
		ChainID:   c.cfg.ChainId,
		Nonce:     nonce,
		To:        &c.cfg.To,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Data:      blob,
	}
	gas, err := core.IntrinsicGas(rawTx.Data, nil, false, true, false)
	if err != nil {
		return nil, err
	}
	rawTx.Gas = gas
	txData = rawTx*/
	//}  else if s.l2BatcherCfg.DataAvailabilityType == batcherFlags.BlobsType {
	var b Blob
	err = b.FromData(blob)
	if err != nil {
		return nil, err
	}
	//require.NoError(t, b.FromData(payload), "must turn data into blob")
	sidecar, blobHashes, err := MakeSidecar([]*Blob{&b})
	if err != nil {
		return nil, err
	}
	/*blobBaseFee := CalcBlobFee(*pendingHeader.ExcessBlobGas)
	blobFeeCap := new(uint256.Int).Mul(uint256.NewInt(2), uint256.MustFromBig(blobBaseFee))
	if blobFeeCap.Lt(uint256.NewInt(params.GWei)) { // ensure we meet 1 gwei geth tx-pool minimum
		blobFeeCap = uint256.NewInt(params.GWei)
	}*/
	blobFeeCap := uint256.NewInt(params.GWei)
	txData = &BlobTx{
		To:         common.HexToAddress(ArchivePoolAddress),
		Data:       nil,
		Gas:        params.TxGas, // intrinsic gas only
		BlobHashes: blobHashes,
		Sidecar:    sidecar,
		ChainID:    uint256.MustFromBig(c.cfg.ChainId),
		GasTipCap:  uint256.MustFromBig(gasTipCap),
		GasFeeCap:  uint256.MustFromBig(gasFeeCap),
		BlobFeeCap: blobFeeCap,
		Value:      uint256.NewInt(0),
		Nonce:      nonce,
	}
	/*} else {
		t.Fatalf("unrecognized DA type: %q", string(s.l2BatcherCfg.DataAvailabilityType))
	}*/

	pKeyBytes, err := hexutil.Decode("0x" + c.cfg.PrivateKey)
	if err != nil {
		return nil, err
	}
	// Convert the private key bytes to an ECDSA private key.
	ecdsaPrivateKey, err := crypto.ToECDSA(pKeyBytes)
	if err != nil {
		return nil, err
	}
	signer := NewEIP4844Signer(c.cfg.ChainId)
	tx, err := SignNewTx(ecdsaPrivateKey, signer, txData)
	if err != nil {
		return nil, err
	}

	sender, err := signer.Sender(tx)
	if err != nil {
		return nil, err
	}
	fmt.Println("sender:", sender)

	err = c.SendTransaction(c.ctx, tx)
	if err != nil {
		return nil, err
	}

	return tx.Hash().Bytes(), nil
}

// MakeSidecar builds & returns the BlobTxSidecar and corresponding blob hashes from the raw blob
// data.
func MakeSidecar(blobs []*Blob) (*BlobTxSidecar, []common.Hash, error) {
	sidecar := &BlobTxSidecar{}
	blobHashes := make([]common.Hash, 0, len(blobs))
	for i, blob := range blobs {
		rawBlob := *blob.KZGBlob()
		sidecar.Blobs = append(sidecar.Blobs, rawBlob)
		commitment, err := BlobToCommitment(rawBlob)
		if err != nil {
			return nil, nil, fmt.Errorf("cannot compute KZG commitment of blob %d in tx candidate: %w", i, err)
		}
		sidecar.Commitments = append(sidecar.Commitments, commitment)
		proof, err := ComputeBlobProof(rawBlob, commitment)
		if err != nil {
			return nil, nil, fmt.Errorf("cannot compute KZG proof for fast commitment verification of blob %d in tx candidate: %w", i, err)
		}
		sidecar.Proofs = append(sidecar.Proofs, proof)
		blobHashes = append(blobHashes, KZGToVersionedHash(commitment))
	}
	return sidecar, blobHashes, nil
}

// Transaction is an Ethereum transaction.
type Transaction struct {
	inner TxData    // Consensus contents of a transaction
	time  time.Time // Time first seen locally (spam avoidance)

	// caches
	hash atomic.Value
	size atomic.Value
	from atomic.Value

	// cache of details to compute the data availability fee
	rollupCostData atomic.Value
}

// SignNewTx creates a transaction and signs it.
func SignNewTx(prv *ecdsa.PrivateKey, s Signer, txdata TxData) (*Transaction, error) {
	tx := NewTx(txdata)
	h := s.Hash(tx)
	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}
	return tx.WithSignature(s, sig)
}

// NewTx creates a new transaction.
func NewTx(inner TxData) *Transaction {
	tx := new(Transaction)
	tx.setDecoded(inner.copy(), 0)
	return tx
}

// WithSignature returns a new transaction with the given signature.
// This signature needs to be in the [R || S || V] format where V is 0 or 1.
func (tx *Transaction) WithSignature(signer Signer, sig []byte) (*Transaction, error) {
	r, s, v, err := signer.SignatureValues(tx, sig)
	if err != nil {
		return nil, err
	}
	if r == nil || s == nil || v == nil {
		return nil, fmt.Errorf("%w: r: %s, s: %s, v: %s", errors.New("invalid transaction v, r, s values"), r, s, v)
	}
	cpy := tx.inner.copy()
	cpy.setSignatureValues(signer.ChainID(), v, r, s)
	return &Transaction{inner: cpy, time: tx.time}, nil
}

// RawSignatureValues returns the V, R, S signature values of the transaction.
// The return values should not be modified by the caller.
// The return values may be nil or zero, if the transaction is unsigned.
func (tx *Transaction) RawSignatureValues() (v, r, s *big.Int) {
	return tx.inner.rawSignatureValues()
}

// setDecoded sets the inner transaction and size after decoding.
func (tx *Transaction) setDecoded(inner TxData, size uint64) {
	tx.inner = inner
	tx.time = time.Now()
	if size > 0 {
		tx.size.Store(size)
	}
}

// MarshalBinary returns the canonical encoding of the transaction.
// For legacy transactions, it returns the RLP encoding. For EIP-2718 typed
// transactions, it returns the type and payload.
func (tx *Transaction) MarshalBinary() ([]byte, error) {

	var buf bytes.Buffer
	err := tx.encodeTyped(&buf)
	return buf.Bytes(), err
}

// encodeTyped writes the canonical encoding of a typed transaction to w.
func (tx *Transaction) encodeTyped(w *bytes.Buffer) error {
	w.WriteByte(tx.Type())
	return tx.inner.encode(w)
}

// Type returns the transaction type.
func (tx *Transaction) Type() uint8 {
	return tx.inner.txType()
}

// Hash returns the transaction hash.
func (tx *Transaction) Hash() common.Hash {
	if hash := tx.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}

	var h common.Hash
	h = prefixedRlpHash(tx.Type(), tx.inner)
	tx.hash.Store(h)
	return h
}

// ChainId returns the EIP155 chain ID of the transaction. The return value will always be
// non-nil. For legacy transactions which are not replay-protected, the return value is
// zero.
func (tx *Transaction) ChainId() *big.Int {
	return tx.inner.chainID()
}

// Data returns the input data of the transaction.
func (tx *Transaction) Data() []byte { return tx.inner.data() }

// AccessList returns the access list of the transaction.
func (tx *Transaction) AccessList() AccessList { return tx.inner.accessList() }

// Gas returns the gas limit of the transaction.
func (tx *Transaction) Gas() uint64 { return tx.inner.gas() }

// GasPrice returns the gas price of the transaction.
func (tx *Transaction) GasPrice() *big.Int { return new(big.Int).Set(tx.inner.gasPrice()) }

// GasTipCap returns the gasTipCap per gas of the transaction.
func (tx *Transaction) GasTipCap() *big.Int { return new(big.Int).Set(tx.inner.gasTipCap()) }

// GasFeeCap returns the fee cap per gas of the transaction.
func (tx *Transaction) GasFeeCap() *big.Int { return new(big.Int).Set(tx.inner.gasFeeCap()) }

// Value returns the ether amount of the transaction.
func (tx *Transaction) Value() *big.Int { return new(big.Int).Set(tx.inner.value()) }

// Nonce returns the sender account nonce of the transaction.
func (tx *Transaction) Nonce() uint64 { return tx.inner.nonce() }

// To returns the recipient address of the transaction.
// For contract-creation transactions, To returns nil.
func (tx *Transaction) To() *common.Address {
	return copyAddressPtr(tx.inner.to())
}

// SetCodeAuthorizations returns the authorizations list of the transaction.
func (tx *Transaction) SetCodeAuthorizations() []SetCodeAuthorization {

	return nil

}

type SetCodeAuthorization struct {
	ChainID uint256.Int    `json:"chainId" gencodec:"required"`
	Address common.Address `json:"address" gencodec:"required"`
	Nonce   uint64         `json:"nonce" gencodec:"required"`
	V       uint8          `json:"yParity" gencodec:"required"`
	R       uint256.Int    `json:"r" gencodec:"required"`
	S       uint256.Int    `json:"s" gencodec:"required"`
}

// copyAddressPtr copies an address.
func copyAddressPtr(a *common.Address) *common.Address {
	if a == nil {
		return nil
	}
	cpy := *a
	return &cpy
}

// hasherPool holds LegacyKeccak256 hashers for rlpHash.
var hasherPool = sync.Pool{
	New: func() interface{} { return sha3.NewLegacyKeccak256() },
}

// prefixedRlpHash writes the prefix into the hasher before rlp-encoding x.
// It's used for typed transactions.
func prefixedRlpHash(prefix byte, x interface{}) (h common.Hash) {
	sha := hasherPool.Get().(crypto.KeccakState)
	defer hasherPool.Put(sha)
	sha.Reset()
	sha.Write([]byte{prefix})
	rlp.Encode(sha, x)
	sha.Read(h[:])
	return h
}

type Signer interface {
	// Sender returns the sender address of the transaction.
	Sender(tx *Transaction) (common.Address, error)

	// SignatureValues returns the raw R, S, V values corresponding to the
	// given signature.
	SignatureValues(tx *Transaction, sig []byte) (r, s, v *big.Int, err error)
	ChainID() *big.Int

	// Hash returns 'signature hash', i.e. the transaction hash that is signed by the
	// private key. This hash does not uniquely identify the transaction.
	Hash(tx *Transaction) common.Hash

	// Equal returns true if the given signer is the same as the receiver.
	Equal(Signer) bool
}

type EIP4844Signer struct {
	chainId, chainIdMul *big.Int
}

// ChainID implements Signer.
func (e EIP4844Signer) ChainID() *big.Int {
	return e.chainId
}

// Equal implements Signer.
func (e EIP4844Signer) Equal(s2 Signer) bool {
	x, ok := s2.(EIP4844Signer)
	return ok && x.chainId.Cmp(e.chainId) == 0
}

// Hash implements Signer.
func (e EIP4844Signer) Hash(tx *Transaction) common.Hash {
	return prefixedRlpHash(
		tx.Type(),
		[]interface{}{
			e.chainId,
			tx.Nonce(),
			tx.GasTipCap(),
			tx.GasFeeCap(),
			tx.Gas(),
			tx.To(),
			tx.Value(),
			tx.Data(),
			tx.AccessList(),
			tx.SetCodeAuthorizations(),
		})
}

func (e EIP4844Signer) Sender(tx *Transaction) (common.Address, error) {

	V, R, S := tx.RawSignatureValues()
	// Blob txs are defined to use 0 and 1 as their recovery
	// id, add 27 to become equivalent to unprotected Homestead signatures.
	V = new(big.Int).Add(V, big.NewInt(27))
	if tx.ChainId().Cmp(e.chainId) != 0 {
		return common.Address{}, fmt.Errorf("%s: have %d want %d", "invalid chain id", tx.ChainId(), e.chainId)
	}
	return recoverPlain(e.Hash(tx), R, S, V, true)
}

// SignatureValues implements Signer.
func (e EIP4844Signer) SignatureValues(tx *Transaction, sig []byte) (*big.Int, *big.Int, *big.Int, error) {
	txdata, ok := tx.inner.(*BlobTx)
	if !ok {
		return nil, nil, nil, fmt.Errorf("wrong tx type")
	}
	// Check that chain ID of tx matches the signer. We also accept ID zero here,
	// because it indicates that the chain ID was not specified in the tx.
	if txdata.ChainID.Sign() != 0 && txdata.ChainID.ToBig().Cmp(e.chainId) != 0 {
		return nil, nil, nil, fmt.Errorf("%s: have %d want %d", "invalid chain id", txdata.ChainID, e.chainId)
	}
	r, s, _ := decodeSignature(sig)
	v := big.NewInt(int64(sig[64]))
	return r, s, v, nil
}

func decodeSignature(sig []byte) (r, s, v *big.Int) {
	if len(sig) != crypto.SignatureLength {
		panic(fmt.Sprintf("wrong size for signature: got %d, want %d", len(sig), crypto.SignatureLength))
	}
	r = new(big.Int).SetBytes(sig[:32])
	s = new(big.Int).SetBytes(sig[32:64])
	v = new(big.Int).SetBytes([]byte{sig[64] + 27})
	return r, s, v
}

func recoverPlain(sighash common.Hash, R, S, Vb *big.Int, homestead bool) (common.Address, error) {
	if Vb.BitLen() > 8 {
		return common.Address{}, errors.New("invalid transaction v, r, s values")
	}
	V := byte(Vb.Uint64() - 27)
	if !crypto.ValidateSignatureValues(V, R, S, homestead) {
		return common.Address{}, errors.New("invalid transaction v, r, s values")
	}
	// encode the signature in uncompressed format
	r, s := R.Bytes(), S.Bytes()
	sig := make([]byte, crypto.SignatureLength)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = V
	// recover the public key from the signature
	pub, err := crypto.Ecrecover(sighash[:], sig)
	if err != nil {
		return common.Address{}, err
	}
	if len(pub) == 0 || pub[0] != 4 {
		return common.Address{}, errors.New("invalid public key")
	}
	var addr common.Address
	copy(addr[:], crypto.Keccak256(pub[1:])[12:])
	return addr, nil
}

func NewEIP4844Signer(chainId *big.Int) EIP4844Signer {
	if chainId == nil {
		chainId = new(big.Int)
	}
	return EIP4844Signer{
		chainId:    chainId,
		chainIdMul: new(big.Int).Mul(chainId, big.NewInt(2)),
	}
}

// SendTransaction injects a signed transaction into the pending pool for execution.
//
// If the transaction was a contract creation use the TransactionReceipt method to get the
// contract address after the transaction has been mined.
func (c Client) SendTransaction(ctx context.Context, tx *Transaction) error {
	data, err := tx.MarshalBinary()
	if err != nil {
		return err
	}
	return c.rpcClient.CallContext(ctx, nil, "eth_sendRawTransaction", hexutil.Encode(data))
}

// HeaderByNumber returns a block header from the current canonical chain. If number is
// nil, the latest known header is returned.
func (c Client) HeaderByNumber(ctx context.Context) (*Header, error) {
	var head *Header
	err := c.rpcClient.CallContext(ctx, &head, "eth_getBlockByNumber", "latest", false)
	if err == nil && head == nil {
		err = gerrc.ErrNotFound
	}
	return head, err
}

// BalanceAt returns the wei balance of the given account.
// The block number can be nil, in which case the balance is taken from the latest known block.
func (c Client) BalanceAt(ctx context.Context, account common.Address) (*big.Int, error) {
	var result hexutil.Big
	err := c.rpcClient.CallContext(ctx, &result, "eth_getBalance", account, "latest")
	return (*big.Int)(&result), err
}

func (c Client) getAddressFromPrivateKey() (common.Address, error) {
	// Getting public address from private key
	pKeyBytes, err := hexutil.Decode("0x" + c.cfg.PrivateKey)
	if err != nil {
		return common.Address{}, err
	}
	// Convert the private key bytes to an ECDSA private key.
	ecdsaPrivateKey, err := crypto.ToECDSA(pKeyBytes)
	if err != nil {
		return common.Address{}, err
	}
	// Extract the public key from the ECDSA private key.
	publicKey := ecdsaPrivateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return common.Address{}, fmt.Errorf("error casting public key to ECDSA")
	}

	// Compute the Ethereum address of the signer from the public key.
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	return fromAddress, nil
}
