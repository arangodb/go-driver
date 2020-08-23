package arangodb

// DatabaseInfo contains information about a database
type DatabaseInfo struct {
	// The identifier of the database.
	ID string `json:"id,omitempty"`
	// The name of the database.
	Name string `json:"name,omitempty"`
	// The filesystem path of the database.
	Path string `json:"path,omitempty"`
	// If true then the database is the _system database.
	IsSystem bool `json:"isSystem,omitempty"`
	// Default replication factor for collections in database
	ReplicationFactor int `json:"replicationFactor,omitempty"`
	// Default write concern for collections in database
	WriteConcern int `json:"writeConcern,omitempty"`
	// Default sharding for collections in database
	Sharding DatabaseSharding `json:"sharding,omitempty"`
}

// EngineType indicates type of database engine being used.
type EngineType string

const (
	EngineTypeMMFiles = EngineType("mmfiles")
	EngineTypeRocksDB = EngineType("rocksdb")
)

func (t EngineType) String() string {
	return string(t)
}

// EngineInfo contains information about the database engine being used.
type EngineInfo struct {
	Type EngineType `json:"name"`
}
