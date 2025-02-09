package loki

import "gopkg.in/yaml.v3"

type Config struct {
	AuthEnabled   bool          `yaml:"auth_enabled"`
	Server        ServerConfig  `yaml:"server"`
	Common        CommonConfig  `yaml:"common"`
	SchemaConfig  SchemaConfig  `yaml:"schema_config"`
	StorageConfig StorageConfig `yaml:"storage_config"`
}

type ServerConfig struct {
	HTTPListenPort int `yaml:"http_listen_port"`
}

type CommonConfig struct {
	Ring              RingConfig `yaml:"ring"`
	ReplicationFactor int        `yaml:"replication_factor"`
	PathPrefix        string     `yaml:"path_prefix"`
}

type RingConfig struct {
	InstanceAddr string        `yaml:"instance_addr"`
	KVStore      KVStoreConfig `yaml:"kvstore"`
}

type KVStoreConfig struct {
	Store string `yaml:"store"`
}

type SchemaConfig struct {
	Configs []SchemaConfigItem `yaml:"configs"`
}

type SchemaConfigItem struct {
	From        string      `yaml:"from"`
	Store       string      `yaml:"store"`
	ObjectStore string      `yaml:"object_store"`
	Schema      string      `yaml:"schema"`
	Index       IndexConfig `yaml:"index"`
}

type IndexConfig struct {
	Prefix string `yaml:"prefix"`
	Period string `yaml:"period"`
}

type StorageConfig struct {
	TSDBShipper TSDBShipperConfig `yaml:"tsdb_shipper"`
	AWS         *AWSConfig        `yaml:"aws,omitempty"`
	FileSystem  *FileSystemConfig `yaml:"filesystem,omitempty"`
}

type TSDBShipperConfig struct {
	ActiveIndexDirectory string `yaml:"active_index_directory"`
	CacheLocation        string `yaml:"cache_location"`
}

type AWSConfig struct {
	S3               string `yaml:"s3"`
	S3ForcePathStyle bool   `yaml:"s3forcepathstyle"`
}

type FileSystemConfig struct {
	Directory string `json:"directory"`
}

func DefaultConfig() *Config {
	return &Config{
		AuthEnabled: false,
		Server: ServerConfig{
			HTTPListenPort: 3100,
		},
		Common: CommonConfig{
			Ring: RingConfig{
				InstanceAddr: "127.0.0.1",
				KVStore: KVStoreConfig{
					Store: "inmemory",
				},
			},
			ReplicationFactor: 1,
			PathPrefix:        "/var/loki",
		},
		SchemaConfig: SchemaConfig{
			Configs: []SchemaConfigItem{
				{
					From:        "2020-05-15",
					Store:       "tsdb",
					ObjectStore: "s3",
					Schema:      "v13",
					Index: IndexConfig{
						Prefix: "index_",
						Period: "24h",
					},
				},
			},
		},
		StorageConfig: StorageConfig{
			TSDBShipper: TSDBShipperConfig{
				ActiveIndexDirectory: "/var/loki/index",
				CacheLocation:        "/var/loki/cache",
			},
			AWS: &AWSConfig{
				S3ForcePathStyle: true,
			},
		},
	}
}

func DefaultConfigYaml() ([]byte, error) {
	return yaml.Marshal(DefaultConfig())
}

type Payload struct {
	Streams []Stream
}

type Stream struct {
	Stream map[string]string `json:"stream"`
	Values [][2]string       `json:"values"`
}

type QueryResult struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Stream map[string]string `json:"stream"`
			Values [][]string        `json:"values"`
		} `json:"result"`
	} `json:"data"`
}
