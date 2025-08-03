package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HandlerSpec defines the desired state of Handler
type HandlerSpec struct {
	// ResourceType is the type of Kubernetes resource this handler manages
	ResourceType string `json:"resourceType"`

	// GVR is the GroupVersionResource specification
	GVR GroupVersionResource `json:"gvr"`

	// Enabled determines whether this handler is enabled
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// Properties defines the properties to extract and store in Neo4j
	// +optional
	Properties map[string]PropertySpec `json:"properties,omitempty"`

	// Relationships defines the relationships to create in Neo4j
	// +optional
	Relationships []RelationshipSpec `json:"relationships,omitempty"`

	// Filters defines filters to apply to resources
	// +optional
	Filters *FilterSpec `json:"filters,omitempty"`

	// Priority determines the processing order (higher numbers processed first)
	// +optional
	Priority *int32 `json:"priority,omitempty"`

	// RetryPolicy defines retry behavior for failed operations
	// +optional
	RetryPolicy *RetryPolicySpec `json:"retryPolicy,omitempty"`

	// Monitoring defines monitoring configuration
	// +optional
	Monitoring *MonitoringSpec `json:"monitoring,omitempty"`
}

// GroupVersionResource represents a Kubernetes GroupVersionResource
type GroupVersionResource struct {
	// Group is the API group (empty string for core resources)
	Group string `json:"group"`

	// Version is the API version
	Version string `json:"version"`

	// Resource is the resource name (plural)
	Resource string `json:"resource"`
}

// PropertySpec defines how to extract a property from a resource
type PropertySpec struct {
	// Path is the JSONPath to extract the property value
	Path string `json:"path"`

	// Type is the data type for the property
	// +optional
	Type string `json:"type,omitempty"`

	// Required determines whether this property is required
	// +optional
	Required *bool `json:"required,omitempty"`

	// Transform is the transformation function to apply
	// +optional
	Transform string `json:"transform,omitempty"`
}

// RelationshipSpec defines a relationship to create in Neo4j
type RelationshipSpec struct {
	// Type is the relationship type in Neo4j
	Type string `json:"type"`

	// Target is the target resource type
	Target string `json:"target"`

	// Direction is the relationship direction
	// +optional
	Direction string `json:"direction,omitempty"`

	// Selector defines how to find the target resource
	// +optional
	Selector *RelationshipSelector `json:"selector,omitempty"`
}

// RelationshipSelector defines how to find the target resource
type RelationshipSelector struct {
	// ByOwnerReference uses owner references to find target
	// +optional
	ByOwnerReference *bool `json:"byOwnerReference,omitempty"`

	// ByLabelSelector uses label selector to find target
	// +optional
	ByLabelSelector *LabelSelectorSpec `json:"byLabelSelector,omitempty"`

	// ByFieldSelector uses field selector to find target
	// +optional
	ByFieldSelector *FieldSelectorSpec `json:"byFieldSelector,omitempty"`

	// ByCustomQuery uses a custom Cypher query to find target
	// +optional
	ByCustomQuery string `json:"byCustomQuery,omitempty"`
}

// LabelSelectorSpec defines label selector configuration
type LabelSelectorSpec struct {
	// Labels are the label key-value pairs
	Labels map[string]string `json:"labels"`
}

// FieldSelectorSpec defines field selector configuration
type FieldSelectorSpec struct {
	// Fields are the field key-value pairs
	Fields map[string]string `json:"fields"`
}

// FilterSpec defines filters to apply to resources
type FilterSpec struct {
	// Namespaces are namespaces to include (empty for all)
	// +optional
	Namespaces []string `json:"namespaces,omitempty"`

	// ExcludeNamespaces are namespaces to exclude
	// +optional
	ExcludeNamespaces []string `json:"excludeNamespaces,omitempty"`

	// Labels are label selectors to include
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations are annotation selectors to include
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// RetryPolicySpec defines retry behavior for failed operations
type RetryPolicySpec struct {
	// MaxRetries is the maximum number of retries
	// +optional
	MaxRetries *int32 `json:"maxRetries,omitempty"`

	// BackoffDelay is the initial backoff delay
	// +optional
	BackoffDelay string `json:"backoffDelay,omitempty"`

	// MaxDelay is the maximum backoff delay
	// +optional
	MaxDelay string `json:"maxDelay,omitempty"`
}

// MonitoringSpec defines monitoring configuration
type MonitoringSpec struct {
	// Metrics enables metrics collection
	// +optional
	Metrics *bool `json:"metrics,omitempty"`

	// Logging defines logging configuration
	// +optional
	Logging *LoggingSpec `json:"logging,omitempty"`
}

// LoggingSpec defines logging configuration
type LoggingSpec struct {
	// Level is the logging level
	// +optional
	Level string `json:"level,omitempty"`

	// Events enables logging of resource events
	// +optional
	Events *bool `json:"events,omitempty"`

	// Errors enables logging of errors
	// +optional
	Errors *bool `json:"errors,omitempty"`
}

// HandlerStatus defines the observed state of Handler
type HandlerStatus struct {
	// Conditions represent the latest available observations of a handler's current state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration is the most recent generation observed by the controller
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// LastProcessedTime is the last time the handler processed a resource
	// +optional
	LastProcessedTime *metav1.Time `json:"lastProcessedTime,omitempty"`

	// ProcessedCount is the number of resources processed by this handler
	// +optional
	ProcessedCount int64 `json:"processedCount,omitempty"`

	// ErrorCount is the number of errors encountered by this handler
	// +optional
	ErrorCount int64 `json:"errorCount,omitempty"`

	// LastError is the last error encountered by this handler
	// +optional
	LastError string `json:"lastError,omitempty"`
}

// Handler is the Schema for the handlers API
type Handler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HandlerSpec   `json:"spec,omitempty"`
	Status HandlerStatus `json:"status,omitempty"`
}

// HandlerList contains a list of Handler
type HandlerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Handler `json:"items"`
}
