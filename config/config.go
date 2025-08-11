package config

type Config struct {
	Neo4j struct {
		URI                            string
		Username                       string
		Password                       string
		MaxConnectionPoolSize          int
		ConnectionAcquisitionTimeout   int // in seconds
		ConnectionLivenessCheckTimeout int // in seconds
		MaxConnectionLifetime          int // in hours
		MaxTransactionRetryTime        int // in seconds
	}
	Kubernetes struct {
		ConfigPath  string
		ClusterName string // Name to identify the cluster in Neo4j
	}
	HTTP struct {
		Enabled bool
		Port    int
	}
	InstanceHash string // Unique hash for this program instance
	EventTTLDays int    // TTL for events in days (0 disables event handling)
}

func NewConfig() *Config {
	return &Config{
		Neo4j: struct {
			URI                            string
			Username                       string
			Password                       string
			MaxConnectionPoolSize          int
			ConnectionAcquisitionTimeout   int
			ConnectionLivenessCheckTimeout int
			MaxConnectionLifetime          int
			MaxTransactionRetryTime        int
		}{
			URI:                            "neo4j://localhost:7687",
			Username:                       "neo4j",
			Password:                       "password",
			MaxConnectionPoolSize:          50,
			ConnectionAcquisitionTimeout:   30,
			ConnectionLivenessCheckTimeout: 30,
			MaxConnectionLifetime:          1,
			MaxTransactionRetryTime:        15,
		},
		Kubernetes: struct {
			ConfigPath  string
			ClusterName string
		}{
			ConfigPath:  "",        // Will use in-cluster config if empty, or load from specified path
			ClusterName: "default", // Default cluster name if not specified
		},
		HTTP: struct {
			Enabled bool
			Port    int
		}{
			Enabled: true,
			Port:    8080,
		},
		InstanceHash: "",
		EventTTLDays: 7,
	}
}
