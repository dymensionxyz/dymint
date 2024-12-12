package store

var (
	_ KV      = &PrefixKV{}
	_ KVBatch = &PrefixKVBatch{}
)

type PrefixKV struct {
	kv     KV
	prefix []byte
}

func (p *PrefixKV) Close() error {
	return p.kv.Close()
}

func NewPrefixKV(kv KV, prefix []byte) *PrefixKV {
	return &PrefixKV{
		kv:     kv,
		prefix: prefix,
	}
}

func (p *PrefixKV) Get(key []byte) ([]byte, error) {
	return p.kv.Get(append(p.prefix, key...))
}

func (p *PrefixKV) Set(key []byte, value []byte) error {
	return p.kv.Set(append(p.prefix, key...), value)
}

func (p *PrefixKV) Delete(key []byte) error {
	return p.kv.Delete(append(p.prefix, key...))
}

func (p *PrefixKV) NewBatch() KVBatch {
	return &PrefixKVBatch{
		b:      p.kv.NewBatch(),
		prefix: p.prefix,
	}
}

func (p *PrefixKV) PrefixIterator(prefix []byte) KVIterator {
	return p.kv.PrefixIterator(append(p.prefix, prefix...))
}

type PrefixKVBatch struct {
	b      KVBatch
	prefix []byte
}

func (pb *PrefixKVBatch) Set(key, value []byte) error {
	return pb.b.Set(append(pb.prefix, key...), value)
}

func (pb *PrefixKVBatch) Delete(key []byte) error {
	return pb.b.Delete(append(pb.prefix, key...))
}

func (pb *PrefixKVBatch) Commit() error {
	return pb.b.Commit()
}

func (pb *PrefixKVBatch) Discard() {
	pb.b.Discard()
}
