package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"kubegraph/config"
	"kubegraph/pkg/logger"
	"kubegraph/pkg/neo4j"

	driverneo4j "github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile     string
	envFile     string
	cfg         *config.Config
	client      *neo4j.Client
	ctx         context.Context
	showQuery   bool
	clusterName string
	showEmojis  bool
	showRelated bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "kubegraph-cli",
	Short: "Neo4j Query CLI for KubeGraph",
	Long: `kubegraph-cli is a command-line tool for querying the Neo4j database
populated by KubeGraph. It provides convenient commands for exploring
Kubernetes resources and their relationships stored in the graph database.

Examples:
  # List all node types
  kubegraph-cli nodes

  # List pods in default namespace
  kubegraph-cli pods default

  # Show recent events
  kubegraph-cli events 50

  # Custom query
  kubegraph-cli query "MATCH (p:Pod)-[:OWNED_BY]->(d:Deployment) RETURN p.name, d.name"

  # Use .env file
  kubegraph-cli --env-file .env nodes`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initializeClient()
	},
}

// nodesCmd represents the nodes command
var nodesCmd = &cobra.Command{
	Use:   "nodes [type] [limit]",
	Short: "List nodes (all types or specific type)",
	Long: `List nodes in the graph database. Without arguments, shows all node types
and their counts. With a node type, lists nodes of that type.

Examples:
  kubegraph-cli nodes                    # Show all node types
  kubegraph-cli nodes Pod 20             # Show 20 Pod nodes
  kubegraph-cli nodes Service            # Show all Service nodes`,
	Args: cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		handleNodes(args)
	},
}

// relationshipsCmd represents the relationships command
var relationshipsCmd = &cobra.Command{
	Use:   "relationships [type] [limit]",
	Short: "List relationships (all types or specific type)",
	Long: `List relationships in the graph database. Without arguments, shows all
relationship types and their counts. With a relationship type, lists
relationships of that type.

Examples:
  kubegraph-cli relationships                    # Show all relationship types
  kubegraph-cli relationships OWNED_BY 20       # Show 20 OWNED_BY relationships`,
	Args: cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		handleRelationships(args)
	},
}

// resourcesCmd represents the resources command
var resourcesCmd = &cobra.Command{
	Use:   "resources",
	Short: "Show resource counts by type",
	Long:  `Show a summary of all resource types and their counts in the database.`,
	Run: func(cmd *cobra.Command, args []string) {
		handleResources()
	},
}

// podsCmd represents the pods command
var podsCmd = &cobra.Command{
	Use:   "pods [namespace]",
	Short: "List pods (optionally filtered by namespace)",
	Long: `List pods in the database. Optionally filter by namespace.

Examples:
  kubegraph-cli pods                    # Show all pods
  kubegraph-cli pods default            # Show pods in default namespace`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		handlePods(args)
	},
}

// servicesCmd represents the services command
var servicesCmd = &cobra.Command{
	Use:   "services [namespace]",
	Short: "List services (optionally filtered by namespace)",
	Long: `List services in the database. Optionally filter by namespace.

Examples:
  kubegraph-cli services                    # Show all services
  kubegraph-cli services default            # Show services in default namespace`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		handleServices(args)
	},
}

// deploymentsCmd represents the deployments command
var deploymentsCmd = &cobra.Command{
	Use:   "deployments [namespace]",
	Short: "List deployments (optionally filtered by namespace)",
	Long: `List deployments in the database. Optionally filter by namespace.

Examples:
  kubegraph-cli deployments                    # Show all deployments
  kubegraph-cli deployments default            # Show deployments in default namespace`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		handleDeployments(args)
	},
}

// eventsCmd represents the events command
var eventsCmd = &cobra.Command{
	Use:   "events [limit]",
	Short: "Show recent events",
	Long: `Show recent events in the database. Optionally specify a limit.

Examples:
  kubegraph-cli events                    # Show 20 recent events
  kubegraph-cli events 50                 # Show 50 recent events`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		handleEvents(args)
	},
}

// dbEventsCmd represents the database events command
var dbEventsCmd = &cobra.Command{
	Use:   "db-events <database-id> [limit]",
	Short: "Show events related to a specific Neo4j database",
	Long: `Show events related to a specific Neo4j database by its ID. 
Optionally specify a limit for the number of events to return.

Examples:
  kubegraph-cli db-events 12345                    # Show events for database ID 12345
  kubegraph-cli db-events 12345 50                 # Show 50 events for database ID 12345`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		handleDbEvents(args)
	},
}

// dbResourcesCmd represents the database resources command
var dbResourcesCmd = &cobra.Command{
	Use:   "db-resources <database-id>",
	Short: "Show all resources related to a specific Neo4j database",
	Long: `Show all Kubernetes resources related to a specific Neo4j database by its ID.
This includes the database itself, its owner (cluster/single instance), 
StatefulSet, ConfigMap, and any associated pods.

Examples:
  kubegraph-cli db-resources 12345                  # Show all resources for database ID 12345`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		handleDbResources(args[0])
	},
}

// clustersCmd represents the clusters command
var clustersCmd = &cobra.Command{
	Use:   "clusters",
	Short: "List all clusters and their hashes",
	Long: `List all unique clusters and their instance hashes found in the database.

Examples:
  kubegraph-cli clusters`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		handleClusters()
	},
}

// queryCmd represents the query command
var queryCmd = &cobra.Command{
	Use:   "query <cypher>",
	Short: "Execute custom Cypher query",
	Long: `Execute a custom Cypher query against the Neo4j database.

Examples:
  kubegraph-cli query "MATCH (p:Pod)-[:OWNED_BY]->(d:Deployment) RETURN p.name, d.name"
  kubegraph-cli query "MATCH (n) RETURN labels(n)[0] as type, count(*) as count"`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		handleCustomQuery(args[0])
	},
}

// statsCmd represents the stats command
var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show database statistics",
	Long:  `Show comprehensive statistics about the database contents.`,
	Run: func(cmd *cobra.Command, args []string) {
		handleStats()
	},
}

// healthCmd represents the health command
var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Show Neo4j health information",
	Long:  `Show Neo4j database health and version information.`,
	Run: func(cmd *cobra.Command, args []string) {
		handleHealth()
	},
}

// k8sNodesCmd represents the k8s-nodes command
var k8sNodesCmd = &cobra.Command{
	Use:   "k8s-nodes",
	Short: "List Kubernetes nodes (computers)",
	Long: `List Kubernetes nodes (computers) in the database with their type, spec, and cluster information.

Examples:
  kubegraph-cli k8s-nodes                    # Show all Kubernetes nodes
  kubegraph-cli k8s-nodes --cluster-name my-cluster  # Show nodes for specific cluster`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		handleK8sNodes()
	},
}

// neo4jDatabasesCmd represents the neo4j-databases command
var neo4jDatabasesCmd = &cobra.Command{
	Use:   "neo4j-databases",
	Short: "List all Neo4j databases",
	Long: `List all Neo4j databases in the database with their details including
ownership, status, and configuration.

Examples:
  kubegraph-cli neo4j-databases                    # Show all Neo4j databases
  kubegraph-cli neo4j-databases --cluster-name my-cluster  # Show databases for specific cluster`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		handleNeo4jDatabases()
	},
}

// rootPodsCmd represents the root-pods command
var rootPodsCmd = &cobra.Command{
	Use:   "root-pods",
	Short: "Find pods running as root",
	Long: `Find pods that are running as root (runAsUser: 0) or have privileged security contexts.
This command helps identify security risks in your Kubernetes cluster.

Examples:
  kubegraph-cli root-pods                    # Show all pods running as root
  kubegraph-cli root-pods --cluster-name my-cluster  # Show root pods for specific cluster`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		handleRootPods()
	},
}

// privilegedPodsCmd represents the privileged-pods command
var privilegedPodsCmd = &cobra.Command{
	Use:   "privileged-pods",
	Short: "Find privileged pods",
	Long: `Find pods that are running in privileged mode, which is a significant security risk.
Privileged containers have access to host resources and can bypass security controls.

Examples:
  kubegraph-cli privileged-pods                    # Show all privileged pods
  kubegraph-cli privileged-pods --cluster-name my-cluster  # Show privileged pods for specific cluster`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		handlePrivilegedPods()
	},
}

// securityRisksCmd represents the security-risks command
var securityRisksCmd = &cobra.Command{
	Use:   "security-risks",
	Short: "Find pods with security risks",
	Long: `Find pods with various security risks including:
- Running as root (runAsUser: 0)
- Privileged containers
- Containers without read-only root filesystem
- Containers with privilege escalation allowed

Examples:
  kubegraph-cli security-risks                    # Show all pods with security risks
  kubegraph-cli security-risks --cluster-name my-cluster  # Show security risks for specific cluster`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		handleSecurityRisks()
	},
}

// resourcePressureCmd represents the resource-pressure command
var resourcePressureCmd = &cobra.Command{
	Use:   "resource-pressure [cpu-threshold] [memory-threshold] [disk-threshold]",
	Short: "Find nodes with resource pressure",
	Long: `Find nodes that are under resource pressure based on CPU, memory, and disk usage.
Thresholds are percentages (0-100). Default thresholds are 80% for CPU and memory, 85% for disk.

Examples:
  kubegraph-cli resource-pressure                    # Show nodes with default thresholds (80% CPU/memory, 85% disk)
  kubegraph-cli resource-pressure 90 85 90          # Show nodes with 90% CPU, 85% memory, 90% disk thresholds
  kubegraph-cli resource-pressure --cluster-name my-cluster  # Show resource pressure for specific cluster`,
	Args: cobra.MaximumNArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		handleResourcePressure(args)
	},
}

// resourcePressureSummaryCmd represents the resource-pressure-summary command
var resourcePressureSummaryCmd = &cobra.Command{
	Use:   "resource-pressure-summary [cpu-threshold] [memory-threshold] [disk-threshold]",
	Short: "Show resource pressure summary",
	Long: `Show a concise summary of nodes with resource pressure, focusing on percentages and pressure levels.
Thresholds are percentages (0-100). Default thresholds are 80% for CPU and memory, 85% for disk.

Examples:
  kubegraph-cli resource-pressure-summary                    # Show summary with default thresholds
  kubegraph-cli resource-pressure-summary 90 85 90          # Show summary with custom thresholds
  kubegraph-cli resource-pressure-summary --cluster-name my-cluster  # Show summary for specific cluster`,
	Args: cobra.MaximumNArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		handleResourcePressureSummary(args)
	},
}

// debugDiskCmd represents the debug-disk command
var debugDiskCmd = &cobra.Command{
	Use:   "debug-disk",
	Short: "Debug disk storage information",
	Long: `Show debug information about disk storage data available in nodes.
This helps diagnose why disk usage might be showing as 0.0%%.

Examples:
  kubegraph-cli debug-disk                    # Show disk data for all nodes
  kubegraph-cli debug-disk --cluster-name my-cluster  # Show disk data for specific cluster`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		handleDebugDisk()
	},
}

// resourceCmd represents the resource command
var resourceCmd = &cobra.Command{
	Use:   "resource [type] [name]",
	Short: "List resources hierarchically or show resource details",
	Long: `List resources in a hierarchical manner:
- No arguments: List all resource types
- Type only: List all resources of that type
- Type and name: Show details of specific resource
- Use --related flag to show related resources

Examples:
  kubegraph-cli resource                    # Show all resource types
  kubegraph-cli resource Pod                # Show all Pod resources
  kubegraph-cli resource Pod my-pod         # Show details of specific Pod
  kubegraph-cli resource Pod my-pod --related  # Show Pod details with related resources`,
	Args: cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		handleResource(args)
	},
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.kubegraph-cli.yaml)")
	rootCmd.PersistentFlags().StringVar(&envFile, "env-file", "", ".env file (default is $HOME/.kubegraph-cli.env, or set KUBEGRAPH_ENV_FILE env var)")
	rootCmd.PersistentFlags().String("uri", "", "Neo4j database URI (default: from NEO4J_URI env var)")
	rootCmd.PersistentFlags().String("user", "", "Neo4j username (default: from NEO4J_USERNAME env var)")
	rootCmd.PersistentFlags().String("pass", "", "Neo4j password (default: from NEO4J_PASSWORD env var)")
	rootCmd.PersistentFlags().StringVar(&clusterName, "cluster-name", "", "Kubernetes cluster name to filter by")
	rootCmd.PersistentFlags().String("log-level", "INFO", "Log level (DEBUG, INFO, WARN, ERROR)")
	rootCmd.PersistentFlags().String("output", "table", "Output format: table, json, csv")
	rootCmd.PersistentFlags().BoolVar(&showQuery, "show-query", false, "Show the executed Cypher query")
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug mode to show configuration details")
	rootCmd.PersistentFlags().BoolVar(&showEmojis, "show-emojis", true, "Show emojis in output")
	rootCmd.PersistentFlags().BoolVar(&showRelated, "related", false, "Show related resources when displaying resource details")

	// Bind flags to viper
	viper.BindPFlag("neo4j.uri", rootCmd.PersistentFlags().Lookup("uri"))
	viper.BindPFlag("neo4j.user", rootCmd.PersistentFlags().Lookup("user"))
	viper.BindPFlag("neo4j.pass", rootCmd.PersistentFlags().Lookup("pass"))
	viper.BindPFlag("kubernetes.cluster", rootCmd.PersistentFlags().Lookup("cluster-name"))
	viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	viper.BindPFlag("show-emojis", rootCmd.PersistentFlags().Lookup("show-emojis"))
	viper.BindPFlag("show-related", rootCmd.PersistentFlags().Lookup("related"))

	// Add subcommands
	rootCmd.AddCommand(nodesCmd)
	rootCmd.AddCommand(relationshipsCmd)
	rootCmd.AddCommand(resourcesCmd)
	rootCmd.AddCommand(podsCmd)
	rootCmd.AddCommand(servicesCmd)
	rootCmd.AddCommand(deploymentsCmd)
	rootCmd.AddCommand(eventsCmd)
	rootCmd.AddCommand(dbEventsCmd)
	rootCmd.AddCommand(dbResourcesCmd)
	rootCmd.AddCommand(clustersCmd)
	rootCmd.AddCommand(queryCmd)
	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(healthCmd)
	rootCmd.AddCommand(k8sNodesCmd)
	rootCmd.AddCommand(neo4jDatabasesCmd)
	rootCmd.AddCommand(rootPodsCmd)
	rootCmd.AddCommand(privilegedPodsCmd)
	rootCmd.AddCommand(securityRisksCmd)
	rootCmd.AddCommand(resourcePressureCmd)
	rootCmd.AddCommand(resourcePressureSummaryCmd)
	rootCmd.AddCommand(debugDiskCmd)
	rootCmd.AddCommand(resourceCmd)
}

// initConfig reads in config file and ENV variables if set
func initConfig() {
	// Load .env file first if specified or if default exists
	loadEnvFile()

	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".kubegraph-cli" (without extension)
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".kubegraph-cli")
	}

	// Read environment variables
	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("neo4j.uri", "neo4j://localhost:7687")
	viper.SetDefault("neo4j.user", "neo4j")
	viper.SetDefault("neo4j.pass", "password")
	viper.SetDefault("log.level", "INFO")
	viper.SetDefault("output", "table")

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}

	// Override with environment variables if they exist
	if uri := os.Getenv("NEO4J_URI"); uri != "" {
		viper.Set("neo4j.uri", uri)
	}
	if user := os.Getenv("NEO4J_USERNAME"); user != "" {
		viper.Set("neo4j.user", user)
	}
	if pass := os.Getenv("NEO4J_PASSWORD"); pass != "" {
		viper.Set("neo4j.pass", pass)
	}
	if cluster := os.Getenv("KUBEGRAPH_CLUSTER_NAME"); cluster != "" {
		viper.Set("kubernetes.cluster", cluster)
	}
	if logLevel := os.Getenv("KUBEGRAPH_LOG_LEVEL"); logLevel != "" {
		viper.Set("log.level", logLevel)
	}
}

// loadEnvFile loads environment variables from a .env file
func loadEnvFile() {
	var envFilePath string

	if envFile != "" {
		// Use env file from the flag (highest priority)
		envFilePath = envFile
	} else if envFileEnv := os.Getenv("KUBEGRAPH_ENV_FILE"); envFileEnv != "" {
		// Use env file from environment variable (second priority)
		envFilePath = envFileEnv
	} else {
		// Try to find default .env file in current directory
		if _, err := os.Stat(".env"); err == nil {
			envFilePath = ".env"
		} else {
			// Try to find default .env file in home directory
			home, err := os.UserHomeDir()
			if err == nil {
				defaultEnvFile := home + "/.kubegraph-cli.env"
				if _, err := os.Stat(defaultEnvFile); err == nil {
					envFilePath = defaultEnvFile
				}
			}
		}
	}

	if envFilePath == "" {
		return // No .env file found
	}

	file, err := os.Open(envFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not open .env file %s: %v\n", envFilePath, err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE format
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				// Remove quotes if present
				if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
					(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
					value = value[1 : len(value)-1]
				}

				// Set environment variable if not already set
				if os.Getenv(key) == "" {
					os.Setenv(key, value)
				}
			} else {
				fmt.Fprintf(os.Stderr, "Warning: Invalid line %d in .env file %s: %s\n", lineNum, envFilePath, line)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Error reading .env file %s: %v\n", envFilePath, err)
		return
	}

	fmt.Fprintf(os.Stderr, "Loaded environment variables from: %s\n", envFilePath)
}

func initializeClient() error {
	// Initialize logger
	logLevel := viper.GetString("log.level")
	logger.Init(logger.ParseLogLevel(logLevel))

	// Create configuration
	cfg = config.NewConfig()
	cfg.Neo4j.URI = viper.GetString("neo4j.uri")
	cfg.Neo4j.Username = viper.GetString("neo4j.user")
	cfg.Neo4j.Password = viper.GetString("neo4j.pass")
	cfg.Kubernetes.ClusterName = viper.GetString("kubernetes.cluster")

	// Debug: Print the configuration being used
	if viper.GetBool("debug") {
		fmt.Fprintf(os.Stderr, "Debug: Using Neo4j URI: %s\n", cfg.Neo4j.URI)
		fmt.Fprintf(os.Stderr, "Debug: Using Neo4j Username: %s\n", cfg.Neo4j.Username)
		fmt.Fprintf(os.Stderr, "Debug: Using Cluster Name: %s\n", cfg.Kubernetes.ClusterName)
	}

	// Initialize context
	ctx = context.Background()

	// Initialize Neo4j client
	var err error
	client, err = neo4j.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create Neo4j client: %w", err)
	}

	return nil
}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func handleNodes(args []string) {
	if len(args) == 0 {
		// Show all node types
		query := `
			MATCH (n)
			WITH labels(n)[0] as label
			RETURN label, count(*) as count
			ORDER BY count DESC`
		executeQuery(query, "Node Types")
		return
	}

	nodeType := args[0]
	limit := 10
	if len(args) > 1 {
		if l, err := fmt.Sscanf(args[1], "%d", &limit); err != nil || l == 0 {
			limit = 10
		}
	}

	query := fmt.Sprintf(`
		MATCH (n:%s)
		%s
		RETURN n.name as name, n.namespace as namespace, n.clusterName as cluster
		ORDER BY n.name
		LIMIT %d`, nodeType, getClusterFilterWithVar("n"), limit)

	executeQuery(query, fmt.Sprintf("%s Nodes", nodeType))
}

func handleRelationships(args []string) {
	if len(args) == 0 {
		// Show all relationship types
		query := fmt.Sprintf(`
			MATCH ()-[r]->()
			%s
			WITH type(r) as relationshipType, count(*) as count
			RETURN relationshipType, count
			ORDER BY count DESC`,
			getClusterFilterForRelationships())
		executeQuery(query, "Relationship Types")
		return
	}

	relType := args[0]
	limit := 10
	if len(args) > 1 {
		if l, err := fmt.Sscanf(args[1], "%d", &limit); err != nil || l == 0 {
			limit = 10
		}
	}

	query := fmt.Sprintf(`
		MATCH (a)-[r:%s]->(b)
		%s
		RETURN labels(a)[0] as from_type, a.name as from_name, 
		       labels(b)[0] as to_type, b.name as to_name,
		       a.clusterName as cluster
		ORDER BY a.name, b.name
		LIMIT %d`, relType, getClusterFilterForRelationships(), limit)

	executeQuery(query, fmt.Sprintf("%s Relationships", relType))
}

func handleResources() {
	query := fmt.Sprintf(`
		MATCH (n)
		%s
		WITH labels(n)[0] as type, n.clusterName as cluster, count(*) as count
		RETURN type, cluster, count
		ORDER BY type, cluster, count DESC`, getClusterFilterWithVar("n"))

	executeQuery(query, "Resource Counts")
}

func handlePods(args []string) {
	namespace := ""
	if len(args) > 0 {
		namespace = args[0]
	}

	query := fmt.Sprintf(`
		MATCH (p:Pod)
		%s
		%s
		RETURN p.name as name, p.namespace as namespace, p.status as status, p.clusterName as cluster
		ORDER BY p.namespace, p.name`,
		getClusterFilterWithVar("p"),
		getNamespaceFilter(namespace, "p"))

	executeQuery(query, "Pods")
}

func handleServices(args []string) {
	namespace := ""
	if len(args) > 0 {
		namespace = args[0]
	}

	query := fmt.Sprintf(`
		MATCH (s:Service)
		%s
		%s
		RETURN s.name as name, s.namespace as namespace, s.type as type, s.clusterName as cluster
		ORDER BY s.namespace, s.name`,
		getClusterFilterWithVar("s"),
		getNamespaceFilter(namespace, "s"))

	executeQuery(query, "Services")
}

func handleDeployments(args []string) {
	namespace := ""
	if len(args) > 0 {
		namespace = args[0]
	}

	query := fmt.Sprintf(`
		MATCH (d:Deployment)
		%s
		%s
		RETURN d.name as name, d.namespace as namespace, d.replicas as replicas, d.clusterName as cluster
		ORDER BY d.namespace, d.name`,
		getClusterFilterWithVar("d"),
		getNamespaceFilter(namespace, "d"))

	executeQuery(query, "Deployments")
}

func handleEvents(args []string) {
	limit := 20
	if len(args) > 0 {
		if l, err := fmt.Sscanf(args[0], "%d", &limit); err != nil || l == 0 {
			limit = 20
		}
	}

	query := fmt.Sprintf(`
		MATCH (e:Event)
		%s
		RETURN e.name as name, e.namespace as namespace, e.type as type, e.reason as reason, e.message as message, e.clusterName as cluster
		ORDER BY e.lastTimestamp DESC
		LIMIT %d`,
		getClusterFilterWithVar("e"), limit)

	executeQuery(query, "Recent Events")
}

func handleDbEvents(args []string) {
	databaseID := args[0]
	limit := 20
	if len(args) > 1 {
		if l, err := fmt.Sscanf(args[1], "%d", &limit); err != nil || l == 0 {
			limit = 20
		}
	}

	query := fmt.Sprintf(`
		MATCH (db:Neo4jDatabase {name: '%s'})
		%s
		OPTIONAL MATCH (db)-[:MANAGED_BY]->(ss:StatefulSet)
		OPTIONAL MATCH (ss)-[:MANAGES]->(pod:Pod)
		OPTIONAL MATCH (ss_event:Event)-[:INVOLVES]->(ss)
		OPTIONAL MATCH (pod_event:Event)-[:INVOLVES]->(pod)
		OPTIONAL MATCH (ipac:IPAccessControl)-[:OWNED_BY]->(db)
		OPTIONAL MATCH (ipac_event:Event)-[:INVOLVES]->(ipac)
		WITH db, ss, pod, ss_event, pod_event, ipac, ipac_event
		UNWIND [ss_event, pod_event, ipac_event] as event
		WITH event
		WHERE event IS NOT NULL
		RETURN DISTINCT event.name as event_name,
		       event.namespace as namespace,
		       event.type as event_type,
		       event.reason as reason,
		       event.message as message,
		       event.lastTimestamp as timestamp,
		       event.clusterName as cluster
		ORDER BY event.lastTimestamp DESC
		LIMIT %d`, databaseID, getClusterFilterWithVar("db"), limit)

	executeQuery(query, fmt.Sprintf("Events for Database ID %s", databaseID))
}

func handleDbResources(databaseID string) {
	query := fmt.Sprintf(`
		MATCH (db:Neo4jDatabase {name: '%s'})
		%s
		OPTIONAL MATCH (db)-[:MANAGED_BY]->(ss:StatefulSet)
		OPTIONAL MATCH (db)-[:OWNS]->(cluster:Neo4jCluster)
		OPTIONAL MATCH (db)-[:USES]->(cm:ConfigMap)
		OPTIONAL MATCH (ss)-[:MANAGES]->(pod:Pod)
		OPTIONAL MATCH (pod)-[:SCHEDULED_ON]->(node:Node)
		OPTIONAL MATCH (pod)-[:USES]->(pvc:PersistentVolumeClaim)
		OPTIONAL MATCH (pod)-[:USES]->(secret:Secret)
		OPTIONAL MATCH (pvc)-[:BOUND_TO]->(pv:PersistentVolume)
		OPTIONAL MATCH (db)<-[:OWNED_BY]-(owned_resource)
		OPTIONAL MATCH (protecting_resource)-[:PROTECTS]->(db)
		WITH db, ss, cluster, cm, pod, node, pvc, secret, pv, owned_resource, protecting_resource
		UNWIND [
			{type: 'Neo4jDatabase', name: db.name, namespace: db.namespace, status: replace(db.phase, '\"', ''), database_id: replace(db.dbid, '\"', ''), cluster: db.clusterName},
			{type: 'StatefulSet', name: ss.name, namespace: ss.namespace, status: ss.replicas, database_id: '', cluster: ss.clusterName},
			{type: 'Neo4jCluster', name: cluster.name, namespace: cluster.namespace, status: replace(cluster.phase, '\"', ''), database_id: '', cluster: cluster.clusterName},
			{type: 'ConfigMap', name: cm.name, namespace: cm.namespace, status: 'Config', database_id: '', cluster: cm.clusterName},
			{type: 'Pod', name: pod.name, namespace: pod.namespace, status: replace(pod.status, '\"', ''), database_id: '', cluster: pod.clusterName},
			{type: 'Node', name: node.name, namespace: '', status: replace(node.status, '\"', ''), database_id: '', cluster: node.clusterName},
			{type: 'PersistentVolumeClaim', name: pvc.name, namespace: pvc.namespace, status: replace(pvc.status, '\"', ''), database_id: '', cluster: pvc.clusterName},
			{type: 'PersistentVolume', name: pv.name, namespace: '', status: replace(pv.status, '\"', ''), database_id: '', cluster: pv.clusterName},
			{type: 'Secret', name: secret.name, namespace: secret.namespace, status: 'Secret', database_id: '', cluster: secret.clusterName},
			{type: labels(owned_resource)[0], name: owned_resource.name, namespace: owned_resource.namespace, status: replace(owned_resource.status, '\"', ''), database_id: replace(owned_resource.dbid, '\"', ''), cluster: owned_resource.clusterName},
			{type: labels(protecting_resource)[0], name: protecting_resource.name, namespace: protecting_resource.namespace, status: replace(protecting_resource.status, '\"', ''), database_id: replace(protecting_resource.dbid, '\"', ''), cluster: protecting_resource.clusterName}
		] as resource
		WITH resource
		WHERE resource.name IS NOT NULL
		RETURN resource.type as resource_type, 
		       resource.name as name, 
		       resource.namespace as namespace, 
		       resource.status as status, 
		       resource.database_id as database_id,
		       resource.cluster as cluster
		UNION
		MATCH (db:Neo4jDatabase {name: '%s'})
		%s
		OPTIONAL MATCH (db)-[:MANAGED_BY]->(ss:StatefulSet)
		OPTIONAL MATCH (ss)-[:MANAGES]->(pod:Pod)
		OPTIONAL MATCH (svc:Service)-[:SELECTS]->(pod)
		WHERE svc IS NOT NULL
		RETURN 'Service' as resource_type,
		       svc.name as name,
		       svc.namespace as namespace,
		       svc.type as status,
		       '' as database_id,
		       svc.clusterName as cluster
		ORDER BY resource_type, name`,
		databaseID, getClusterFilterWithVar("db"), databaseID, getClusterFilterWithVar("db"))

	executeQuery(query, fmt.Sprintf("Resources for Database ID %s", databaseID))
}

func handleCustomQuery(query string) {
	executeQuery(query, "Custom Query")
}

func handleStats() {
	query := fmt.Sprintf(`
		MATCH (n)
		%s
		WITH labels(n)[0] as type, n.clusterName as cluster, count(*) as count
		RETURN type, cluster, count
		ORDER BY type, cluster, count DESC`, getClusterFilterWithVar("n"))

	executeQuery(query, "Database Statistics")
}

func handleHealth() {
	query := `
		CALL dbms.components() YIELD name, versions, edition
		RETURN name, versions[0] as version, edition`

	executeQuery(query, "Neo4j Health")
}

func handleClusters() {
	query := `
		MATCH (n)
		WHERE n.clusterName IS NOT NULL AND n.instanceHash IS NOT NULL AND NOT n:Event
		WITH n.clusterName as cluster, n.instanceHash as hash, n.creationTimestamp as timestamp
		ORDER BY cluster, timestamp DESC
		WITH cluster, collect({hash: hash, timestamp: timestamp})[0] as latest
		WITH cluster, latest.hash as hash, latest.timestamp as last_updated,
		     CASE 
		       WHEN latest.timestamp > datetime() - duration('PT1H') THEN '🟢 ACTIVE'
		       WHEN latest.timestamp > datetime() - duration('PT24H') THEN '🟡 RECENT'
		       ELSE '⚪ OLD'
		     END as status
		RETURN cluster, hash, last_updated, status
		ORDER BY cluster`

	executeQuery(query, "Active Clusters")
}

func handleK8sNodes() {
	query := fmt.Sprintf(`
		MATCH (n:Node)
		%s
		RETURN n.name as name, n.phase as status, n.clusterName as cluster, 
		       n.architecture as architecture, n.operatingSystem as os, 
		       n.kernelVersion as kernel, n.kubeletVersion as kubelet,
		       n.capacityCPU as cpu, n.capacityMemory as memory, n.capacityPods as max_pods,
		       n.unschedulable as unschedulable, n.creationTimestamp as created
		ORDER BY n.name`,
		getClusterFilterWithVar("n"))

	executeQuery(query, "Kubernetes Nodes")
}

func handleNeo4jDatabases() {
	query := fmt.Sprintf(`
		MATCH (db:Neo4jDatabase)
		%s
		OPTIONAL MATCH (db)-[:OWNS]->(owner)
		RETURN db.name as name, 
		       db.namespace as namespace, 
		       db.phase as status, 
		       db.dbid as database_id, 
		       db.clusterName as cluster,
		       labels(owner)[0] as owner_type,
		       owner.name as owner_name,
		       db.singleInstance as is_single_instance,
		       db.coreCount as core_count,
		       db.primariesCount as primaries_count,
		       db.creationTimestamp as created
		ORDER BY db.name`,
		getClusterFilterWithVar("db"))

	executeQuery(query, "Neo4j Databases")
}

func handleRootPods() {
	query := fmt.Sprintf(`
		MATCH (p:Pod)
		%s
		WHERE (
			// Check pod-level security context for runAsUser: 0
			(p.podSecurityContext IS NOT NULL AND p.podSecurityContext CONTAINS '"runAsUser":0') OR
			
			// Check container-level security contexts for runAsUser: 0
			(p.containerSecurityContexts IS NOT NULL AND ANY(ctx IN p.containerSecurityContexts WHERE ctx CONTAINS '"runAsUser":0')) OR
			
			// Check for privileged containers
			(p.containerSecurityContexts IS NOT NULL AND ANY(ctx IN p.containerSecurityContexts WHERE ctx CONTAINS '"privileged":true')) OR
			
			// Check for specific security annotations
			(p.annotations IS NOT NULL AND (
				p.annotations CONTAINS '"runAsUser":"0"' OR 
				p.annotations CONTAINS '"runAsUser":0' OR
				p.annotations CONTAINS '"security.alpha.kubernetes.io/runAsUser":"0"'
			)) OR
			
			// Check for security-related labels
			(p.labels IS NOT NULL AND (
				p.labels CONTAINS 'runAsUser=0' OR
				p.labels CONTAINS 'security-context=root' OR
				p.labels CONTAINS 'privileged=true'
			))
		)
		RETURN p.name as pod_name, 
		       p.namespace as namespace, 
		       p.status as status,
		       p.podSecurityContext as pod_security_context,
		       p.containerSecurityContexts as container_security_contexts,
		       p.annotations as annotations,
		       p.labels as labels,
		       p.clusterName as cluster
		ORDER BY p.namespace, p.name`,
		getClusterFilterWithVar("p"))

	executeQuery(query, "Pods Running as Root")
}

func handlePrivilegedPods() {
	query := fmt.Sprintf(`
		MATCH (p:Pod)
		%s
		WHERE (
			// Check for privileged containers
			(p.containerSecurityContexts IS NOT NULL AND ANY(ctx IN p.containerSecurityContexts WHERE ctx CONTAINS '"privileged":true')) OR
			
			// Check for specific security annotations
			(p.annotations IS NOT NULL AND (
				p.annotations CONTAINS '"privileged":"true"' OR 
				p.annotations CONTAINS '"privileged":true' OR
				p.annotations CONTAINS '"security.alpha.kubernetes.io/runAsUser":"0"'
			)) OR
			
			// Check for security-related labels
			(p.labels IS NOT NULL AND (
				p.labels CONTAINS 'privileged=true' OR
				p.labels CONTAINS 'security-context=privileged' OR
				p.labels CONTAINS 'privileged=true'
			))
		)
		RETURN p.name as pod_name, 
		       p.namespace as namespace, 
		       p.status as status,
		       p.podSecurityContext as pod_security_context,
		       p.containerSecurityContexts as container_security_contexts,
		       p.annotations as annotations,
		       p.labels as labels,
		       p.clusterName as cluster
		ORDER BY p.namespace, p.name`,
		getClusterFilterWithVar("p"))

	executeQuery(query, "Pods Running in Privileged Mode")
}

func handleSecurityRisks() {
	query := fmt.Sprintf(`
		MATCH (p:Pod)
		%s
		WHERE (
			// Check for running as root (runAsUser: 0)
			(p.podSecurityContext IS NOT NULL AND p.podSecurityContext CONTAINS '"runAsUser":0') OR
			
			// Check for privileged containers
			(p.containerSecurityContexts IS NOT NULL AND ANY(ctx IN p.containerSecurityContexts WHERE ctx CONTAINS '"privileged":true')) OR
			
			// Check for containers without read-only root filesystem
			(p.containerSecurityContexts IS NOT NULL AND ANY(ctx IN p.containerSecurityContexts WHERE ctx CONTAINS '"security.alpha.kubernetes.io/runAsUser":"0"')) OR
			
			// Check for containers with privilege escalation allowed
			(p.containerSecurityContexts IS NOT NULL AND ANY(ctx IN p.containerSecurityContexts WHERE ctx CONTAINS '"security.alpha.kubernetes.io/runAsUser":"0"')) OR
			
			// Check for security-related labels
			(p.labels IS NOT NULL AND (
				p.labels CONTAINS 'runAsUser=0' OR
				p.labels CONTAINS 'security-context=root' OR
				p.labels CONTAINS 'privileged=true'
			))
		)
		RETURN p.name as pod_name, 
		       p.namespace as namespace, 
		       p.status as status,
		       p.podSecurityContext as pod_security_context,
		       p.containerSecurityContexts as container_security_contexts,
		       p.annotations as annotations,
		       p.labels as labels,
		       p.clusterName as cluster
		ORDER BY p.namespace, p.name`,
		getClusterFilterWithVar("p"))

	executeQuery(query, "Pods with Security Risks")
}

func handleResourcePressure(args []string) {
	// Parse thresholds with defaults
	cpuThreshold := 80.0
	memoryThreshold := 80.0
	diskThreshold := 85.0

	if len(args) > 0 {
		if val, err := fmt.Sscanf(args[0], "%f", &cpuThreshold); err != nil || val == 0 {
			cpuThreshold = 80.0
		}
	}
	if len(args) > 1 {
		if val, err := fmt.Sscanf(args[1], "%f", &memoryThreshold); err != nil || val == 0 {
			memoryThreshold = 80.0
		}
	}
	if len(args) > 2 {
		if val, err := fmt.Sscanf(args[2], "%f", &diskThreshold); err != nil || val == 0 {
			diskThreshold = 85.0
		}
	}

	// Build emoji prefix based on flag
	emojiPrefix := ""
	if showEmojis {
		emojiPrefix = "CASE WHEN cpu_usage_percent >= 90 THEN '🔴' WHEN cpu_usage_percent >= 80 THEN '🟡' ELSE '🟢' END + ' '"
	}

	query := fmt.Sprintf(`
		MATCH (n:Node)
		%s
		WHERE n.capacityCPU IS NOT NULL AND n.capacityMemory IS NOT NULL
		WITH n, 
		     n.capacityCPU as cpu_capacity_str,
		     n.capacityMemory as memory_capacity_str,
		     n.allocatableCPU as cpu_allocatable_str,
		     n.allocatableMemory as memory_allocatable_str,
		     n.capacityEphemeralStorage as disk_capacity_str,
		     n.allocatableEphemeralStorage as disk_allocatable_str
		WHERE cpu_capacity_str IS NOT NULL AND memory_capacity_str IS NOT NULL
		WITH n, cpu_capacity_str, memory_capacity_str, cpu_allocatable_str, memory_allocatable_str, disk_capacity_str, disk_allocatable_str,
		     // Extract numeric values from resource strings (e.g., "4" from "4", "8Gi" from "8Gi")
		     CASE 
		       WHEN cpu_capacity_str CONTAINS 'm' THEN toFloat(replace(cpu_capacity_str, 'm', '')) / 1000
		       ELSE toFloat(cpu_capacity_str)
		     END as cpu_capacity,
		     CASE 
		       WHEN memory_capacity_str CONTAINS 'Ki' THEN toFloat(replace(memory_capacity_str, 'Ki', '')) / 1024
		       WHEN memory_capacity_str CONTAINS 'Mi' THEN toFloat(replace(memory_capacity_str, 'Mi', ''))
		       WHEN memory_capacity_str CONTAINS 'Gi' THEN toFloat(replace(memory_capacity_str, 'Gi', '')) * 1024
		       WHEN memory_capacity_str CONTAINS 'Ti' THEN toFloat(replace(memory_capacity_str, 'Ti', '')) * 1024 * 1024
		       ELSE toFloat(memory_capacity_str)
		     END as memory_capacity_mb,
		     CASE 
		       WHEN cpu_allocatable_str CONTAINS 'm' THEN toFloat(replace(cpu_allocatable_str, 'm', '')) / 1000
		       ELSE toFloat(cpu_allocatable_str)
		     END as cpu_allocatable,
		     CASE 
		       WHEN memory_allocatable_str CONTAINS 'Ki' THEN toFloat(replace(memory_allocatable_str, 'Ki', '')) / 1024
		       WHEN memory_allocatable_str CONTAINS 'Mi' THEN toFloat(replace(memory_allocatable_str, 'Mi', ''))
		       WHEN memory_allocatable_str CONTAINS 'Gi' THEN toFloat(replace(memory_allocatable_str, 'Gi', '')) * 1024
		       WHEN memory_allocatable_str CONTAINS 'Ti' THEN toFloat(replace(memory_allocatable_str, 'Ti', '')) * 1024 * 1024
		       ELSE toFloat(memory_allocatable_str)
		     END as memory_allocatable_mb,
		     CASE 
		       WHEN disk_capacity_str IS NOT NULL AND disk_capacity_str CONTAINS 'Ki' THEN toFloat(replace(disk_capacity_str, 'Ki', '')) / 1024
		       WHEN disk_capacity_str IS NOT NULL AND disk_capacity_str CONTAINS 'Mi' THEN toFloat(replace(disk_capacity_str, 'Mi', ''))
		       WHEN disk_capacity_str IS NOT NULL AND disk_capacity_str CONTAINS 'Gi' THEN toFloat(replace(disk_capacity_str, 'Gi', '')) * 1024
		       WHEN disk_capacity_str IS NOT NULL AND disk_capacity_str CONTAINS 'Ti' THEN toFloat(replace(disk_capacity_str, 'Ti', '')) * 1024 * 1024
		       WHEN disk_capacity_str IS NOT NULL THEN toFloat(disk_capacity_str)
		       ELSE 0
		     END as disk_capacity_mb,
		     CASE 
		       WHEN disk_allocatable_str IS NOT NULL AND disk_allocatable_str CONTAINS 'Ki' THEN toFloat(replace(disk_allocatable_str, 'Ki', '')) / 1024
		       WHEN disk_allocatable_str IS NOT NULL AND disk_allocatable_str CONTAINS 'Mi' THEN toFloat(replace(disk_allocatable_str, 'Mi', ''))
		       WHEN disk_allocatable_str IS NOT NULL AND disk_allocatable_str CONTAINS 'Gi' THEN toFloat(replace(disk_allocatable_str, 'Gi', '')) * 1024
		       WHEN disk_allocatable_str IS NOT NULL AND disk_allocatable_str CONTAINS 'Ti' THEN toFloat(replace(disk_allocatable_str, 'Ti', '')) * 1024 * 1024
		       WHEN disk_allocatable_str IS NOT NULL THEN toFloat(disk_allocatable_str) / (1024 * 1024)
		       ELSE 0
		     END as disk_allocatable_mb
		WHERE cpu_capacity > 0 AND memory_capacity_mb > 0
		WITH n, cpu_capacity, memory_capacity_mb, cpu_allocatable, memory_allocatable_mb, disk_capacity_mb, disk_allocatable_mb,
		     // Calculate usage percentages - fixed disk calculation
		     CASE 
		       WHEN cpu_allocatable IS NOT NULL AND cpu_capacity > 0 THEN ((cpu_capacity - cpu_allocatable) / cpu_capacity) * 100
		       ELSE 0 
		     END as cpu_usage_percent,
		     CASE 
		       WHEN memory_allocatable_mb IS NOT NULL AND memory_capacity_mb > 0 THEN ((memory_capacity_mb - memory_allocatable_mb) / memory_capacity_mb) * 100
		       ELSE 0 
		     END as memory_usage_percent,
		     CASE 
		       WHEN disk_allocatable_mb > 0 AND disk_capacity_mb > 0 THEN 
		         CASE 
		           WHEN disk_allocatable_mb <= disk_capacity_mb THEN ((disk_capacity_mb - disk_allocatable_mb) / disk_capacity_mb) * 100
		           ELSE 0
		         END
		       ELSE 0 
		     END as disk_usage_percent
		WHERE cpu_usage_percent >= %f OR memory_usage_percent >= %f OR disk_usage_percent >= %f
		RETURN n.name as node_name,
		       n.phase as status,
		       n.architecture as architecture,
		       n.operatingSystem as os,
		       n.capacityCPU as cpu_capacity,
		       n.allocatableCPU as cpu_allocatable,
		       %s + ROUND(cpu_usage_percent, 1) + '%%' as cpu_usage,
		       n.capacityMemory as memory_capacity,
		       n.allocatableMemory as memory_allocatable,
		       %s + ROUND(memory_usage_percent, 1) + '%%' as memory_usage,
		       n.capacityEphemeralStorage as disk_capacity,
		       n.allocatableEphemeralStorage as disk_allocatable,
		       %s + ROUND(disk_usage_percent, 1) + '%%' as disk_usage,
		       n.unschedulable as unschedulable,
		       n.clusterName as cluster
		ORDER BY (cpu_usage_percent + memory_usage_percent + disk_usage_percent) DESC, n.name`,
		getClusterFilterWithVar("n"), cpuThreshold, memoryThreshold, diskThreshold, emojiPrefix, emojiPrefix, emojiPrefix)

	executeQuery(query, fmt.Sprintf("Nodes with Resource Pressure (CPU≥%.0f%%, Memory≥%.0f%%, Disk≥%.0f%%)", cpuThreshold, memoryThreshold, diskThreshold))
}

func handleResourcePressureSummary(args []string) {
	// Parse thresholds with defaults
	cpuThreshold := 80.0
	memoryThreshold := 80.0
	diskThreshold := 85.0

	if len(args) > 0 {
		if val, err := fmt.Sscanf(args[0], "%f", &cpuThreshold); err != nil || val == 0 {
			cpuThreshold = 80.0
		}
	}
	if len(args) > 1 {
		if val, err := fmt.Sscanf(args[1], "%f", &memoryThreshold); err != nil || val == 0 {
			memoryThreshold = 80.0
		}
	}
	if len(args) > 2 {
		if val, err := fmt.Sscanf(args[2], "%f", &diskThreshold); err != nil || val == 0 {
			diskThreshold = 85.0
		}
	}

	// Build separate emoji prefixes for different resources
	cpuEmojiPrefix := ""
	memoryEmojiPrefix := ""
	diskEmojiPrefix := ""

	if showEmojis {
		cpuEmojiPrefix = "CASE WHEN cpu_usage_percent >= 90 THEN '🔴 CRITICAL' WHEN cpu_usage_percent >= 80 THEN '🟡 WARNING' ELSE '🟢 OK' END"
		memoryEmojiPrefix = "CASE WHEN memory_usage_percent >= 90 THEN '🔴 CRITICAL' WHEN memory_usage_percent >= 80 THEN '🟡 WARNING' ELSE '🟢 OK' END"
		diskEmojiPrefix = "CASE WHEN disk_usage_percent >= 90 THEN '🔴 CRITICAL' WHEN disk_usage_percent >= 85 THEN '🟡 WARNING' ELSE '🟢 OK' END"
	}

	query := fmt.Sprintf(`
		MATCH (n:Node)
		%s
		WHERE n.capacityCPU IS NOT NULL AND n.capacityMemory IS NOT NULL
		WITH n, 
		     n.capacityCPU as cpu_capacity_str,
		     n.capacityMemory as memory_capacity_str,
		     n.allocatableCPU as cpu_allocatable_str,
		     n.allocatableMemory as memory_allocatable_str,
		     n.capacityEphemeralStorage as disk_capacity_str,
		     n.allocatableEphemeralStorage as disk_allocatable_str
		WHERE cpu_capacity_str IS NOT NULL AND memory_capacity_str IS NOT NULL
		WITH n, cpu_capacity_str, memory_capacity_str, cpu_allocatable_str, memory_allocatable_str, disk_capacity_str, disk_allocatable_str,
		     // Extract numeric values from resource strings
		     CASE 
		       WHEN cpu_capacity_str CONTAINS 'm' THEN toFloat(replace(cpu_capacity_str, 'm', '')) / 1000
		       ELSE toFloat(cpu_capacity_str)
		     END as cpu_capacity,
		     CASE 
		       WHEN memory_capacity_str CONTAINS 'Ki' THEN toFloat(replace(memory_capacity_str, 'Ki', '')) / 1024
		       WHEN memory_capacity_str CONTAINS 'Mi' THEN toFloat(replace(memory_capacity_str, 'Mi', ''))
		       WHEN memory_capacity_str CONTAINS 'Gi' THEN toFloat(replace(memory_capacity_str, 'Gi', '')) * 1024
		       WHEN memory_capacity_str CONTAINS 'Ti' THEN toFloat(replace(memory_capacity_str, 'Ti', '')) * 1024 * 1024
		       ELSE toFloat(memory_capacity_str)
		     END as memory_capacity_mb,
		     CASE 
		       WHEN cpu_allocatable_str CONTAINS 'm' THEN toFloat(replace(cpu_allocatable_str, 'm', '')) / 1000
		       ELSE toFloat(cpu_allocatable_str)
		     END as cpu_allocatable,
		     CASE 
		       WHEN memory_allocatable_str CONTAINS 'Ki' THEN toFloat(replace(memory_allocatable_str, 'Ki', '')) / 1024
		       WHEN memory_allocatable_str CONTAINS 'Mi' THEN toFloat(replace(memory_allocatable_str, 'Mi', ''))
		       WHEN memory_allocatable_str CONTAINS 'Gi' THEN toFloat(replace(memory_allocatable_str, 'Gi', '')) * 1024
		       WHEN memory_allocatable_str CONTAINS 'Ti' THEN toFloat(replace(memory_allocatable_str, 'Ti', '')) * 1024 * 1024
		       ELSE toFloat(memory_allocatable_str)
		     END as memory_allocatable_mb,
		     CASE 
		       WHEN disk_capacity_str CONTAINS 'Ki' THEN toFloat(replace(disk_capacity_str, 'Ki', '')) / 1024
		       WHEN disk_capacity_str CONTAINS 'Mi' THEN toFloat(replace(disk_capacity_str, 'Mi', ''))
		       WHEN disk_capacity_str CONTAINS 'Gi' THEN toFloat(replace(disk_capacity_str, 'Gi', '')) * 1024
		       WHEN disk_capacity_str CONTAINS 'Ti' THEN toFloat(replace(disk_capacity_str, 'Ti', '')) * 1024 * 1024
		       ELSE toFloat(disk_capacity_str)
		     END as disk_capacity_mb,
		     CASE 
		       WHEN disk_allocatable_str IS NOT NULL AND disk_allocatable_str CONTAINS 'Ki' THEN toFloat(replace(disk_allocatable_str, 'Ki', '')) / 1024
		       WHEN disk_allocatable_str IS NOT NULL AND disk_allocatable_str CONTAINS 'Mi' THEN toFloat(replace(disk_allocatable_str, 'Mi', ''))
		       WHEN disk_allocatable_str IS NOT NULL AND disk_allocatable_str CONTAINS 'Gi' THEN toFloat(replace(disk_allocatable_str, 'Gi', '')) * 1024
		       WHEN disk_allocatable_str IS NOT NULL AND disk_allocatable_str CONTAINS 'Ti' THEN toFloat(replace(disk_allocatable_str, 'Ti', '')) * 1024 * 1024
		       WHEN disk_allocatable_str IS NOT NULL THEN toFloat(disk_allocatable_str) / (1024 * 1024)
		       ELSE 0
		     END as disk_allocatable_mb
		WHERE cpu_capacity > 0 AND memory_capacity_mb > 0
		WITH n, cpu_capacity, memory_capacity_mb, cpu_allocatable, memory_allocatable_mb, disk_capacity_mb, disk_allocatable_mb,
		     // Calculate usage percentages - fixed disk calculation
		     CASE 
		       WHEN cpu_allocatable IS NOT NULL AND cpu_capacity > 0 THEN ((cpu_capacity - cpu_allocatable) / cpu_capacity) * 100
		       ELSE 0 
		     END as cpu_usage_percent,
		     CASE 
		       WHEN memory_allocatable_mb IS NOT NULL AND memory_capacity_mb > 0 THEN ((memory_capacity_mb - memory_allocatable_mb) / memory_capacity_mb) * 100
		       ELSE 0 
		     END as memory_usage_percent,
		     CASE 
		       WHEN disk_allocatable_mb > 0 AND disk_capacity_mb > 0 THEN 
		         CASE 
		           WHEN disk_allocatable_mb <= disk_capacity_mb THEN ((disk_capacity_mb - disk_allocatable_mb) / disk_capacity_mb) * 100
		           ELSE 0
		         END
		       ELSE 0 
		     END as disk_usage_percent
		WHERE cpu_usage_percent >= %f OR memory_usage_percent >= %f OR disk_usage_percent >= %f
		WITH n, cpu_usage_percent, memory_usage_percent, disk_usage_percent,
		     // Calculate overall pressure score
		     (cpu_usage_percent + memory_usage_percent + disk_usage_percent) / 3 as overall_pressure
		RETURN n.name as node_name,
		       n.phase as status,
		       n.clusterName as cluster,
		       ROUND(cpu_usage_percent, 1) + '%%' as cpu_usage,
		       %s as cpu_status,
		       ROUND(memory_usage_percent, 1) + '%%' as memory_usage,
		       %s as memory_status,
		       ROUND(disk_usage_percent, 1) + '%%' as disk_usage,
		       %s as disk_status,
		       ROUND(overall_pressure, 1) + '%%' as overall_pressure,
		       n.unschedulable as unschedulable
		ORDER BY overall_pressure DESC, n.name`,
		getClusterFilterWithVar("n"), cpuThreshold, memoryThreshold, diskThreshold, cpuEmojiPrefix, memoryEmojiPrefix, diskEmojiPrefix)

	executeQuery(query, fmt.Sprintf("Resource Pressure Summary (CPU≥%.0f%%, Memory≥%.0f%%, Disk≥%.0f%%)", cpuThreshold, memoryThreshold, diskThreshold))
}

func handleDebugDisk() {
	query := fmt.Sprintf(`
		MATCH (n:Node)
		%s
		WITH n, 
		     n.capacityEphemeralStorage as disk_capacity_str,
		     n.allocatableEphemeralStorage as disk_allocatable_str
		WHERE disk_capacity_str IS NOT NULL OR disk_allocatable_str IS NOT NULL
		WITH n, disk_capacity_str, disk_allocatable_str,
		     // Parse the values to see what we're getting
		     CASE 
		       WHEN disk_capacity_str IS NOT NULL AND disk_capacity_str CONTAINS 'Ki' THEN toFloat(replace(disk_capacity_str, 'Ki', '')) / 1024
		       WHEN disk_capacity_str IS NOT NULL AND disk_capacity_str CONTAINS 'Mi' THEN toFloat(replace(disk_capacity_str, 'Mi', ''))
		       WHEN disk_capacity_str IS NOT NULL AND disk_capacity_str CONTAINS 'Gi' THEN toFloat(replace(disk_capacity_str, 'Gi', '')) * 1024
		       WHEN disk_capacity_str IS NOT NULL AND disk_capacity_str CONTAINS 'Ti' THEN toFloat(replace(disk_capacity_str, 'Ti', '')) * 1024 * 1024
		       WHEN disk_capacity_str IS NOT NULL THEN toFloat(disk_capacity_str)
		       ELSE 0
		     END as disk_capacity_mb,
		     CASE 
		       WHEN disk_allocatable_str IS NOT NULL AND disk_allocatable_str CONTAINS 'Ki' THEN toFloat(replace(disk_allocatable_str, 'Ki', '')) / 1024
		       WHEN disk_allocatable_str IS NOT NULL AND disk_allocatable_str CONTAINS 'Mi' THEN toFloat(replace(disk_allocatable_str, 'Mi', ''))
		       WHEN disk_allocatable_str IS NOT NULL AND disk_allocatable_str CONTAINS 'Gi' THEN toFloat(replace(disk_allocatable_str, 'Gi', '')) * 1024
		       WHEN disk_allocatable_str IS NOT NULL AND disk_allocatable_str CONTAINS 'Ti' THEN toFloat(replace(disk_allocatable_str, 'Ti', '')) * 1024 * 1024
		       WHEN disk_allocatable_str IS NOT NULL THEN toFloat(disk_allocatable_str) / (1024 * 1024)
		       ELSE 0
		     END as disk_allocatable_mb
		RETURN n.name as name, 
		       disk_capacity_str as raw_capacity, 
		       disk_allocatable_str as raw_allocatable,
		       ROUND(disk_capacity_mb, 2) as capacity_mb,
		       ROUND(disk_allocatable_mb, 2) as allocatable_mb,
		       CASE 
		         WHEN disk_allocatable_mb > 0 AND disk_capacity_mb > 0 AND disk_allocatable_mb <= disk_capacity_mb 
		         THEN ROUND(((disk_capacity_mb - disk_allocatable_mb) / disk_capacity_mb) * 100, 2)
		         ELSE 0 
		       END as usage_percent
		ORDER BY n.name`,
		getClusterFilterWithVar("n"))

	executeQuery(query, "Detailed Disk Data for All Nodes")
}

func handleResource(args []string) {
	if len(args) == 0 {
		// Show all resource types
		emojiPrefix := ""
		if showEmojis {
			emojiPrefix = "CASE WHEN count > 100 THEN '🔴' WHEN count > 50 THEN '🟡' ELSE '🟢' END + ' '"
		}
		query := fmt.Sprintf(`
			MATCH (n)
			%s
			WITH labels(n)[0] as type, count(*) as count
			RETURN %s + type as type, count
			ORDER BY count DESC`, getClusterFilterWithVar("n"), emojiPrefix)
		executeQuery(query, "Resource Types")
		return
	}

	resourceType := args[0]

	if len(args) == 1 {
		// Show all resources of the specified type
		emojiPrefix := ""
		if showEmojis {
			emojiPrefix = "CASE WHEN n.status = 'Running' THEN '🟢' WHEN n.status = 'Pending' THEN '🟡' WHEN n.status = 'Failed' THEN '🔴' ELSE '⚪' END + ' '"
		}
		query := fmt.Sprintf(`
			MATCH (n:%s)
			%s
			RETURN %s + n.name as name, n.namespace as namespace, n.clusterName as cluster
			ORDER BY n.namespace, n.name`, resourceType, getClusterFilterWithVar("n"), emojiPrefix)
		executeQuery(query, fmt.Sprintf("%s Resources", resourceType))
		return
	}

	resourceName := args[1]

	if showRelated {
		// Show resource details with cascading related resources
		emojiPrefix := ""
		if showEmojis {
			emojiPrefix = "CASE WHEN rel.direction = 'outgoing' THEN '➡️' ELSE '⬅️' END + ' '"
		}
		query := fmt.Sprintf(`
			MATCH (resource:%s {name: '%s'})
			%s
			// Direct relationships (one hop)
			OPTIONAL MATCH (resource)-[r]->(related)
			OPTIONAL MATCH (incoming)-[r2]->(resource)
			// Cascading relationships (multiple hops) - for Deployments, find ReplicaSets and their Pods
			OPTIONAL MATCH path = (resource)-[:OWNED_BY*]->(cascading)
			WITH resource, 
			     collect(DISTINCT {type: type(r), target: labels(related)[0], name: related.name, direction: 'outgoing', hops: 1}) as outgoing,
			     collect(DISTINCT {type: type(r2), source: labels(incoming)[0], name: incoming.name, direction: 'incoming', hops: 1}) as incoming,
			     collect(DISTINCT {type: 'CASCADING', target: labels(cascading)[0], name: cascading.name, direction: 'outgoing', hops: length(path)}) as cascading
			UNWIND (outgoing + incoming + cascading) as rel
			WITH resource, rel
			WHERE rel.name IS NOT NULL
			RETURN resource.name as resource_name,
			       resource.namespace as namespace,
			       resource.clusterName as cluster,
			       %s + rel.direction as direction,
			       rel.type as relationship_type,
			       CASE 
			         WHEN rel.direction = 'outgoing' THEN rel.target
			         ELSE rel.source
			       END as related_type,
			       rel.name as related_name,
			       rel.hops as hops
			ORDER BY rel.hops, rel.direction, rel.type, rel.name`, resourceType, resourceName, getClusterFilterWithVar("resource"), emojiPrefix)
		executeQuery(query, fmt.Sprintf("%s Details with Related Resources", resourceType))
	} else {
		// Show basic resource details
		emojiPrefix := ""
		if showEmojis {
			emojiPrefix = "CASE WHEN n.phase = 'Running' OR n.status = 'Running' THEN '🟢' WHEN n.phase = 'Pending' OR n.status = 'Pending' THEN '🟡' WHEN n.phase = 'Failed' OR n.status = 'Failed' THEN '🔴' ELSE '⚪' END + ' '"
		}
		query := fmt.Sprintf(`
			MATCH (n:%s {name: '%s'})
			%s
			RETURN %s + n.name as name, 
			       n.namespace as namespace, 
			       n.clusterName as cluster,
			       keys(n) as properties
			ORDER BY n.name`, resourceType, resourceName, getClusterFilterWithVar("n"), emojiPrefix)
		executeQuery(query, fmt.Sprintf("%s Details", resourceType))
	}
}

func executeQuery(query, title string) {
	// Show the query if the flag is enabled
	if showQuery {
		fmt.Printf("\n=== Cypher Query ===\n%s\n", query)
	}

	session := client.Driver().NewSession(ctx, driverneo4j.SessionConfig{AccessMode: driverneo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.Run(ctx, query, nil)
	if err != nil {
		logger.Error("Failed to execute query: %v", err)
		return
	}

	records, err := driverneo4j.CollectWithContext(ctx, result, err)
	if err != nil {
		logger.Error("Failed to collect results: %v", err)
		os.Exit(1)
	}

	if len(records) == 0 {
		fmt.Printf("No results found for: %s\n", title)
		return
	}

	// Get column names from first record
	keys := records[0].Keys
	values := make([][]string, len(records))

	// Extract values
	for i, record := range records {
		row := make([]string, len(keys))
		for j := range keys {
			value := record.Values[j]
			if value == nil {
				row[j] = "null"
			} else {
				row[j] = fmt.Sprintf("%v", value)
			}
		}
		values[i] = row
	}

	// Print results
	fmt.Printf("\n=== %s ===\n", title)
	fmt.Printf("Found %d results\n\n", len(records))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Print header
	header := strings.Join(keys, "\t")
	fmt.Fprintln(w, header)

	// Print separator
	separator := strings.Repeat("-\t", len(keys)-1) + "-"
	fmt.Fprintln(w, separator)

	// Print data
	for _, row := range values {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}

	w.Flush()
	fmt.Println()
}

func getClusterFilter() string {
	return getClusterFilterWithVar("n")
}

func getClusterFilterWithVar(varName string) string {
	// Priority: 1. --cluster-name flag, 2. config cluster name
	cluster := clusterName
	if cluster == "" {
		cluster = cfg.Kubernetes.ClusterName
	}

	if cluster == "" {
		return ""
	}
	return fmt.Sprintf("WHERE %s.clusterName = '%s'", varName, cluster)
}

func getClusterFilterForRelationships() string {
	// Priority: 1. --cluster-name flag, 2. config cluster name
	cluster := clusterName
	if cluster == "" {
		cluster = cfg.Kubernetes.ClusterName
	}

	if cluster == "" {
		return ""
	}
	return fmt.Sprintf("WHERE a.clusterName = '%s' AND b.clusterName = '%s'", cluster, cluster)
}

func getNamespaceFilter(namespace string, varName string) string {
	if namespace == "" {
		return ""
	}
	return fmt.Sprintf("WHERE %s.namespace = '%s'", varName, namespace)
}
