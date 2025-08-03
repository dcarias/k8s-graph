package config

type Config struct {
	Neo4j struct {
		URI      string
		Username string
		Password string
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
			URI      string
			Username string
			Password string
		}{
			URI:      "neo4j://localhost:7687",
			Username: "neo4j",
			Password: "password",
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