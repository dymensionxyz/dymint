package rollapp

const (
	
	ModuleName = "rollapp"

	
	StoreKey = ModuleName

	
	RouterKey = ModuleName

	
	QuerierRoute = ModuleName

	
	MemStoreKey = "mem_rollapp"
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
