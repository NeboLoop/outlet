package events

import (
	"time"
)

// These define the public API contract for what topics external consumers can subscribe to.
const (
	// System lifecycle events
	TopicSystemInitializing = "system.initializing" // System startup beginning
	TopicSystemConfigured   = "system.configured"   // Configuration loaded and validated
	TopicSystemReady        = "system.ready"        // All prerequisites met, ready for processing
	TopicSystemStarted      = "system.started"      // All services running, ready for traffic
	TopicSystemShutdown     = "system.shutdown"     // Graceful shutdown initiated

	// Database configuration events
	TopicDatabaseConfigUpdated   = "database.config_updated"   // Database configuration updated
	TopicPostgresConfigUpdated   = "postgres.config_updated"   // PostgreSQL configuration updated
	TopicClickHouseConfigUpdated = "clickhouse.config_updated" // ClickHouse configuration updated

	// Organization and Brand management events
	TopicOrgCreated      = "org.created"       // New organization created
	TopicOrgUpdated      = "org.updated"       // Organization settings updated
	TopicOrgDeleted      = "org.deleted"       // Organization deleted
	TopicBrandCreated    = "brand.created"     // New brand created
	TopicBrandUpdated    = "brand.updated"     // Brand settings updated
	TopicBrandDeleted    = "brand.deleted"     // Brand deleted
	TopicUserInvited     = "user.invited"      // User invited to organization
	TopicUserJoined      = "user.joined"       // User joined organization
	TopicUserLeft        = "user.left"         // User left organization
	TopicUserRoleChanged = "user.role_changed" // User role changed

	// API Key management events
	TopicAPIKeyCreated = "apikey.created" // New API key created
	TopicAPIKeyUpdated = "apikey.updated" // API key updated
	TopicAPIKeyDeleted = "apikey.deleted" // API key deleted
	TopicAPIKeyToggled = "apikey.toggled" // API key enabled/disabled
	TopicAPIKeyUsed    = "apikey.used"    // API key used for request

	// Data ingestion events
	TopicDataIngested            = "data.ingested"            // New data ingested from external source
	TopicCostImportStarted       = "cost_import.started"      // Cost import process started
	TopicCostImportCompleted     = "cost_import.completed"    // Cost import process completed
	TopicCostImportFailed        = "cost_import.failed"       // Cost import process failed
	TopicIntegrationConnected    = "integration.connected"    // External integration connected
	TopicIntegrationDisconnected = "integration.disconnected" // External integration disconnected
	TopicIntegrationError        = "integration.error"        // Integration error occurred

	// Attribution and analytics events
	TopicAttributionCalculated  = "attribution.calculated"   // Attribution model calculation completed
	TopicAttributionUpdated     = "attribution.updated"      // Attribution data updated
	TopicMMMProjectionGenerated = "mmm.projection_generated" // MMM projection generated
	TopicMMMProjectionUpdated   = "mmm.projection_updated"   // MMM projection updated
	TopicFunnelAnalyzed         = "funnel.analyzed"          // Funnel analysis completed
	TopicDashboardUpdated       = "dashboard.updated"        // Dashboard data updated
	TopicDashboardRefreshDue    = "dashboard.refresh_due"    // Live dashboard refresh is due

	// Optimization events
	TopicOptimizationRuleCreated  = "optimization.rule_created"             // New optimization rule created
	TopicOptimizationRuleUpdated  = "optimization.rule_updated"             // Optimization rule updated
	TopicOptimizationRuleDeleted  = "optimization.rule_deleted"             // Optimization rule deleted
	TopicOptimizationRuleExecuted = "optimization.rule_executed"            // Optimization rule executed
	TopicRecommendationGenerated  = "optimization.recommendation_generated" // New recommendation generated
	TopicBudgetReallocated        = "optimization.budget_reallocated"       // Budget reallocation applied

	// License and billing events
	TopicLicenseUpdated     = "license.updated"      // License information updated
	TopicLicenseExpired     = "license.expired"      // License expired
	TopicLicenseRenewed     = "license.renewed"      // License renewed
	TopicUsageLimitReached  = "usage.limit_reached"  // Usage limit reached
	TopicUsageLimitExceeded = "usage.limit_exceeded" // Usage limit exceeded

	// Configuration events
	TopicConfigUpdated          = "config.updated"           // Configuration updated via KV store
	TopicConfigNamespaceUpdated = "config.namespace_updated" // Configuration namespace updated

	// SDK Management events
	TopicSDKConfigCreated = "sdk.config_created" // SDK configuration created
	TopicSDKConfigUpdated = "sdk.config_updated" // SDK configuration updated
	TopicSDKConfigDeleted = "sdk.config_deleted" // SDK configuration deleted
	TopicSDKEventReceived = "sdk.event_received" // Event received from SDK

	// Additional topics for realtime bridge
	TopicFunnelDiscovered   = "funnel.discovered"   // Funnel discovered
	TopicFunnelSaved        = "funnel.saved"        // Funnel saved
	TopicRulesetCreated     = "ruleset.created"     // Ruleset created
	TopicRulesetUpdated     = "ruleset.updated"     // Ruleset updated
	TopicRulesetDeleted     = "ruleset.deleted"     // Ruleset deleted
	TopicExecutionStarted   = "execution.started"   // Execution started
	TopicExecutionCompleted = "execution.completed" // Execution completed
	TopicExecutionFailed    = "execution.failed"    // Execution failed
	TopicKVConfigUpdated    = "kv_config.updated"   // KV config updated
	TopicKVConfigDeleted    = "kv_config.deleted"   // KV config deleted

	// ETL Pipeline events
	TopicETLStarted      = "etl.started"       // ETL pipeline started
	TopicETLStopped      = "etl.stopped"       // ETL pipeline stopped
	TopicETLJobQueued    = "etl.job_queued"    // ETL job queued
	TopicETLJobStarted   = "etl.job_started"   // ETL job started
	TopicETLJobCompleted = "etl.job_completed" // ETL job completed
	TopicETLJobFailed    = "etl.job_failed"    // ETL job failed
	TopicETLScaling      = "etl.scaling"       // ETL worker scaling event

	// Webhook events
	TopicWebhookReceived        = "webhook.received"         // Webhook received from platform
	TopicPlatformDataChanged    = "platform.data_changed"    // Platform data changed (triggers sync)
	TopicPlatformConversionSync = "platform.conversion_sync" // Platform conversion needs sync

	// Sync events
	TopicSyncStarted   = "sync.started"   // Sync job started
	TopicSyncProgress  = "sync.progress"  // Sync job progress update
	TopicSyncCompleted = "sync.completed" // Sync job completed successfully
	TopicSyncFailed    = "sync.failed"    // Sync job failed
	TopicSyncCancelled = "sync.cancelled" // Sync job cancelled

	// Code indexing events
	TopicCodeIndexStats = "code_index.stats" // Code index stats updated

	// =====================================================
	// Outlet Business Rules Engine Topics
	// =====================================================

	// Support Ticket events
	TopicTicketCreated   = "ticket.created"   // Support ticket created
	TopicTicketUpdated   = "ticket.updated"   // Support ticket updated
	TopicTicketEscalated = "ticket.escalated" // Support ticket escalated to higher priority
	TopicTicketResolved  = "ticket.resolved"  // Support ticket marked as resolved
	TopicTicketClosed    = "ticket.closed"    // Support ticket closed

	// Payment events
	TopicPaymentSucceeded = "payment.succeeded" // Payment succeeded
	TopicPaymentFailed    = "payment.failed"    // Payment failed
	TopicRefundCreated    = "refund.created"    // Refund issued

	// Subscription events
	TopicSubscriptionCreated  = "subscription.created"  // New subscription created
	TopicSubscriptionUpdated  = "subscription.updated"  // Subscription modified
	TopicSubscriptionCanceled = "subscription.canceled" // Subscription canceled
	TopicSubscriptionPastDue  = "subscription.past_due" // Subscription payment past due
	TopicSubscriptionRenewed  = "subscription.renewed"  // Subscription renewed

	// Email events
	TopicEmailSent       = "email.sent"       // Email sent
	TopicEmailDelivered  = "email.delivered"  // Email delivered to recipient
	TopicEmailBounced    = "email.bounced"    // Email bounced (hard/soft)
	TopicEmailComplained = "email.complained" // Email marked as spam
	TopicEmailOpened     = "email.opened"     // Email opened (tracking pixel)
	TopicEmailClicked    = "email.clicked"    // Link in email clicked

	// Customer events
	TopicCustomerCreated = "customer.created" // New customer created
	TopicCustomerUpdated = "customer.updated" // Customer profile updated
	TopicCustomerChurned = "customer.churned" // Customer churned (canceled all subscriptions)

	// Contact/Lead events
	TopicContactCreated      = "contact.created"      // New contact/subscriber added
	TopicContactUnsubscribed = "contact.unsubscribed" // Contact unsubscribed from list

	// Order/Checkout events
	TopicCheckoutCompleted = "checkout.completed" // Checkout session completed
	TopicOrderCreated      = "order.created"      // Order created
	TopicOrderCompleted    = "order.completed"    // Order fulfilled/completed
)

// System lifecycle event structures

// SystemInitializingEvent is emitted when system startup begins
type SystemInitializingEvent struct {
	Timestamp   time.Time `json:"timestamp"`
	Version     string    `json:"version"`
	Environment string    `json:"environment"`
}

// SystemConfiguredEvent is emitted when configuration is loaded and validated
type SystemConfiguredEvent struct {
	Timestamp     time.Time `json:"timestamp"`
	ConfigValid   bool      `json:"config_valid"`
	DatabaseDSN   string    `json:"database_dsn"`
	ClickHouseDSN string    `json:"clickhouse_dsn"`
}

// SystemReadyEvent is emitted when all prerequisites are met and system is ready for processing
type SystemReadyEvent struct {
	Timestamp        time.Time `json:"timestamp"`
	DatabaseReady    bool      `json:"database_ready"`
	ClickHouseReady  bool      `json:"clickhouse_ready"`
	ServicesReady    bool      `json:"services_ready"`
	MigrationVersion int64     `json:"migration_version"`
}

// SystemStartedEvent is emitted when all services are running and ready for traffic
type SystemStartedEvent struct {
	Timestamp     time.Time `json:"timestamp"`
	ServicesCount int       `json:"services_count"`
	DashboardURL  string    `json:"dashboard_url"`
	APIEndpoint   string    `json:"api_endpoint"`
}

// SystemShutdownEvent is emitted when graceful shutdown is initiated
type SystemShutdownEvent struct {
	Timestamp   time.Time `json:"timestamp"`
	Reason      string    `json:"reason"`
	GracePeriod int       `json:"grace_period_seconds"`
}

// Database configuration event structures

// DatabaseConfigUpdatedEvent is emitted when database configuration is updated
type DatabaseConfigUpdatedEvent struct {
	Timestamp      time.Time `json:"timestamp"`
	DatabaseType   string    `json:"database_type"` // "postgres" or "clickhouse"
	Host           string    `json:"host"`
	Port           int       `json:"port"`
	Database       string    `json:"database"`
	User           string    `json:"user"`
	SSLMode        string    `json:"ssl_mode,omitempty"` // Only for PostgreSQL
	ConnectionTest bool      `json:"connection_test"`
	ConfigSaved    bool      `json:"config_saved"`
}

// PostgresConfigUpdatedEvent is emitted when PostgreSQL configuration is updated
type PostgresConfigUpdatedEvent struct {
	Timestamp      time.Time `json:"timestamp"`
	Host           string    `json:"host"`
	Port           int       `json:"port"`
	Database       string    `json:"database"`
	User           string    `json:"user"`
	SSLMode        string    `json:"ssl_mode"`
	ConnectionTest bool      `json:"connection_test"`
	ConfigSaved    bool      `json:"config_saved"`
}

// ClickHouseConfigUpdatedEvent is emitted when ClickHouse configuration is updated
type ClickHouseConfigUpdatedEvent struct {
	Timestamp      time.Time `json:"timestamp"`
	Host           string    `json:"host"`
	Port           int       `json:"port"`
	Database       string    `json:"database"`
	User           string    `json:"user"`
	ConnectionTest bool      `json:"connection_test"`
	ConfigSaved    bool      `json:"config_saved"`
}

// Organization and Brand management event structures

// OrgCreatedEvent is emitted when a new organization is created
type OrgCreatedEvent struct {
	OrgID     string    `json:"org_id"`
	Name      string    `json:"name"`
	Plan      string    `json:"plan"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

// OrgUpdatedEvent is emitted when organization settings are updated
type OrgUpdatedEvent struct {
	OrgID     string                 `json:"org_id"`
	Changes   map[string]interface{} `json:"changes"`
	UpdatedBy string                 `json:"updated_by"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// BrandCreatedEvent is emitted when a new brand is created
type BrandCreatedEvent struct {
	BrandID   string    `json:"brand_id"`
	OrgID     string    `json:"org_id"`
	Name      string    `json:"name"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

// BrandUpdatedEvent is emitted when brand settings are updated
type BrandUpdatedEvent struct {
	BrandID   string                 `json:"brand_id"`
	OrgID     string                 `json:"org_id"`
	Changes   map[string]interface{} `json:"changes"`
	UpdatedBy string                 `json:"updated_by"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// UserInvitedEvent is emitted when a user is invited to an organization
type UserInvitedEvent struct {
	UserID    string    `json:"user_id"`
	OrgID     string    `json:"org_id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	InvitedBy string    `json:"invited_by"`
	InvitedAt time.Time `json:"invited_at"`
}

// UserJoinedEvent is emitted when a user joins an organization
type UserJoinedEvent struct {
	UserID   string    `json:"user_id"`
	OrgID    string    `json:"org_id"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

// UserRoleChangedEvent is emitted when a user's role changes
type UserRoleChangedEvent struct {
	UserID    string    `json:"user_id"`
	OrgID     string    `json:"org_id"`
	OldRole   string    `json:"old_role"`
	NewRole   string    `json:"new_role"`
	ChangedBy string    `json:"changed_by"`
	ChangedAt time.Time `json:"changed_at"`
}

// API Key management event structures

// APIKeyCreatedEvent is emitted when a new API key is created
type APIKeyCreatedEvent struct {
	APIKeyID  string    `json:"apikey_id"`
	OrgID     string    `json:"org_id"`
	BrandID   string    `json:"brand_id,omitempty"`
	Name      string    `json:"name"`
	Provider  string    `json:"provider"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

// APIKeyUpdatedEvent is emitted when an API key is updated
type APIKeyUpdatedEvent struct {
	APIKeyID  string                 `json:"apikey_id"`
	OrgID     string                 `json:"org_id"`
	Changes   map[string]interface{} `json:"changes"`
	UpdatedBy string                 `json:"updated_by"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// APIKeyUsedEvent is emitted when an API key is used for a request
type APIKeyUsedEvent struct {
	APIKeyID   string    `json:"apikey_id"`
	OrgID      string    `json:"org_id"`
	BrandID    string    `json:"brand_id,omitempty"`
	Endpoint   string    `json:"endpoint"`
	Method     string    `json:"method"`
	StatusCode int       `json:"status_code"`
	UsedAt     time.Time `json:"used_at"`
}

// Data ingestion event structures

// DataIngestedEvent is emitted when new data is ingested from external source
type DataIngestedEvent struct {
	OrgID       string    `json:"org_id"`
	BrandID     string    `json:"brand_id,omitempty"`
	Source      string    `json:"source"`
	DataType    string    `json:"data_type"`
	RecordCount int64     `json:"record_count"`
	IngestedAt  time.Time `json:"ingested_at"`
}

// CostImportStartedEvent is emitted when cost import process starts
type CostImportStartedEvent struct {
	ImportID  string    `json:"import_id"`
	OrgID     string    `json:"org_id"`
	BrandID   string    `json:"brand_id,omitempty"`
	Source    string    `json:"source"`
	StartedAt time.Time `json:"started_at"`
}

// CostImportCompletedEvent is emitted when cost import process completes
type CostImportCompletedEvent struct {
	ImportID    string    `json:"import_id"`
	OrgID       string    `json:"org_id"`
	BrandID     string    `json:"brand_id,omitempty"`
	Source      string    `json:"source"`
	RecordCount int64     `json:"record_count"`
	Duration    int64     `json:"duration_seconds"`
	CompletedAt time.Time `json:"completed_at"`
}

// IntegrationConnectedEvent is emitted when external integration connects
type IntegrationConnectedEvent struct {
	IntegrationID string    `json:"integration_id"`
	OrgID         string    `json:"org_id"`
	BrandID       string    `json:"brand_id,omitempty"`
	Provider      string    `json:"provider"`
	ConnectedAt   time.Time `json:"connected_at"`
}

// IntegrationErrorEvent is emitted when integration error occurs
type IntegrationErrorEvent struct {
	IntegrationID string    `json:"integration_id"`
	OrgID         string    `json:"org_id"`
	BrandID       string    `json:"brand_id,omitempty"`
	Provider      string    `json:"provider"`
	Error         string    `json:"error"`
	OccurredAt    time.Time `json:"occurred_at"`
}

// Attribution and analytics event structures

// AttributionCalculatedEvent is emitted when attribution model calculation completes
type AttributionCalculatedEvent struct {
	OrgID        string    `json:"org_id"`
	BrandID      string    `json:"brand_id,omitempty"`
	Model        string    `json:"model"`
	Period       string    `json:"period"`
	Touchpoints  int64     `json:"touchpoints"`
	CalculatedAt time.Time `json:"calculated_at"`
}

// AttributionUpdatedEvent is emitted when attribution data is updated
type AttributionUpdatedEvent struct {
	OrgID     string    `json:"org_id"`
	BrandID   string    `json:"brand_id,omitempty"`
	Model     string    `json:"model"`
	Period    string    `json:"period"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MMMProjectionGeneratedEvent is emitted when MMM projection is generated
type MMMProjectionGeneratedEvent struct {
	OrgID          string    `json:"org_id"`
	BrandID        string    `json:"brand_id,omitempty"`
	ModelID        string    `json:"model_id"`
	Period         string    `json:"period"`
	ProjectionType string    `json:"projection_type"`
	GeneratedAt    time.Time `json:"generated_at"`
}

// FunnelAnalyzedEvent is emitted when funnel analysis completes
type FunnelAnalyzedEvent struct {
	OrgID      string    `json:"org_id"`
	BrandID    string    `json:"brand_id,omitempty"`
	FunnelID   string    `json:"funnel_id"`
	StageCount int       `json:"stage_count"`
	AnalyzedAt time.Time `json:"analyzed_at"`
}

// DashboardUpdatedEvent is emitted when dashboard data is updated
type DashboardUpdatedEvent struct {
	OrgID      string    `json:"org_id"`
	BrandID    string    `json:"brand_id,omitempty"`
	WidgetType string    `json:"widget_type"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// DashboardRefreshDueEvent is emitted when a live dashboard is due for refresh
type DashboardRefreshDueEvent struct {
	DashboardID    string    `json:"dashboard_id"`
	OrganizationID string    `json:"organization_id"`
	UserID         string    `json:"user_id"`
	Timestamp      time.Time `json:"timestamp"`
}

// Optimization event structures

// OptimizationRuleCreatedEvent is emitted when new optimization rule is created
type OptimizationRuleCreatedEvent struct {
	RuleID    string    `json:"rule_id"`
	OrgID     string    `json:"org_id"`
	BrandID   string    `json:"brand_id,omitempty"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

// OptimizationRuleExecutedEvent is emitted when optimization rule is executed
type OptimizationRuleExecutedEvent struct {
	RuleID     string    `json:"rule_id"`
	OrgID      string    `json:"org_id"`
	BrandID    string    `json:"brand_id,omitempty"`
	Action     string    `json:"action"`
	Impact     string    `json:"impact"`
	ExecutedAt time.Time `json:"executed_at"`
}

// RecommendationGeneratedEvent is emitted when new recommendation is generated
type RecommendationGeneratedEvent struct {
	RecommendationID string    `json:"recommendation_id"`
	OrgID            string    `json:"org_id"`
	BrandID          string    `json:"brand_id,omitempty"`
	Type             string    `json:"type"`
	Priority         string    `json:"priority"`
	GeneratedAt      time.Time `json:"generated_at"`
}

// BudgetReallocatedEvent is emitted when budget reallocation is applied
type BudgetReallocatedEvent struct {
	OrgID         string    `json:"org_id"`
	BrandID       string    `json:"brand_id,omitempty"`
	FromChannel   string    `json:"from_channel"`
	ToChannel     string    `json:"to_channel"`
	Amount        float64   `json:"amount"`
	ReallocatedAt time.Time `json:"reallocated_at"`
}

// License and billing event structures

// LicenseUpdatedEvent is emitted when license information is updated
type LicenseUpdatedEvent struct {
	OrgID     string    `json:"org_id"`
	Plan      string    `json:"plan"`
	Features  []string  `json:"features"`
	ExpiresAt time.Time `json:"expires_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// LicenseExpiredEvent is emitted when license expires
type LicenseExpiredEvent struct {
	OrgID     string    `json:"org_id"`
	ExpiredAt time.Time `json:"expired_at"`
}

// UsageLimitReachedEvent is emitted when usage limit is reached
type UsageLimitReachedEvent struct {
	OrgID        string    `json:"org_id"`
	BrandID      string    `json:"brand_id,omitempty"`
	LimitType    string    `json:"limit_type"`
	CurrentUsage int64     `json:"current_usage"`
	Limit        int64     `json:"limit"`
	ReachedAt    time.Time `json:"reached_at"`
}

// UsageLimitExceededEvent is emitted when usage limit is exceeded
type UsageLimitExceededEvent struct {
	OrgID        string    `json:"org_id"`
	BrandID      string    `json:"brand_id,omitempty"`
	LimitType    string    `json:"limit_type"`
	CurrentUsage int64     `json:"current_usage"`
	Limit        int64     `json:"limit"`
	ExceededAt   time.Time `json:"exceeded_at"`
}

// Configuration event structures

// ConfigUpdatedEvent is emitted when configuration is updated via KV store
type ConfigUpdatedEvent struct {
	OrgID     string    `json:"org_id"`
	BrandID   string    `json:"brand_id,omitempty"`
	Namespace string    `json:"namespace"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedBy string    `json:"updated_by"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ConfigNamespaceUpdatedEvent is emitted when configuration namespace is updated
type ConfigNamespaceUpdatedEvent struct {
	OrgID     string    `json:"org_id"`
	BrandID   string    `json:"brand_id,omitempty"`
	Namespace string    `json:"namespace"`
	KeyCount  int       `json:"key_count"`
	UpdatedBy string    `json:"updated_by"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SDK Management event structures

// SDKConfigCreatedEvent is emitted when SDK configuration is created
type SDKConfigCreatedEvent struct {
	ConfigID  string    `json:"config_id"`
	OrgID     string    `json:"org_id"`
	BrandID   string    `json:"brand_id,omitempty"`
	Name      string    `json:"name"`
	Platform  string    `json:"platform"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

// SDKConfigUpdatedEvent is emitted when SDK configuration is updated
type SDKConfigUpdatedEvent struct {
	ConfigID  string                 `json:"config_id"`
	OrgID     string                 `json:"org_id"`
	BrandID   string                 `json:"brand_id,omitempty"`
	Changes   map[string]interface{} `json:"changes"`
	UpdatedBy string                 `json:"updated_by"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// SDKConfigDeletedEvent is emitted when SDK configuration is deleted
type SDKConfigDeletedEvent struct {
	ConfigID  string    `json:"config_id"`
	OrgID     string    `json:"org_id"`
	BrandID   string    `json:"brand_id,omitempty"`
	Name      string    `json:"name"`
	DeletedBy string    `json:"deleted_by"`
	DeletedAt time.Time `json:"deleted_at"`
}

// SDKEventReceivedEvent is emitted when event is received from SDK
type SDKEventReceivedEvent struct {
	EventID    string    `json:"event_id"`
	OrgID      string    `json:"org_id"`
	BrandID    string    `json:"brand_id,omitempty"`
	ConfigID   string    `json:"config_id"`
	EventType  string    `json:"event_type"`
	ReceivedAt time.Time `json:"received_at"`
}

// Additional event structures for realtime bridge

// FunnelDiscoveredEvent is emitted when a funnel is discovered
type FunnelDiscoveredEvent struct {
	OrgID        string    `json:"org_id"`
	BrandID      string    `json:"brand_id,omitempty"`
	FunnelID     string    `json:"funnel_id"`
	Name         string    `json:"name"`
	StageCount   int       `json:"stage_count"`
	DiscoveredAt time.Time `json:"discovered_at"`
}

// FunnelSavedEvent is emitted when a funnel is saved
type FunnelSavedEvent struct {
	OrgID    string    `json:"org_id"`
	BrandID  string    `json:"brand_id,omitempty"`
	FunnelID string    `json:"funnel_id"`
	Name     string    `json:"name"`
	SavedBy  string    `json:"saved_by"`
	SavedAt  time.Time `json:"saved_at"`
}

// OrgDeletedEvent is emitted when an organization is deleted
type OrgDeletedEvent struct {
	OrgID     string    `json:"org_id"`
	Name      string    `json:"name"`
	DeletedBy string    `json:"deleted_by"`
	DeletedAt time.Time `json:"deleted_at"`
}

// BrandDeletedEvent is emitted when a brand is deleted
type BrandDeletedEvent struct {
	BrandID   string    `json:"brand_id"`
	OrgID     string    `json:"org_id"`
	Name      string    `json:"name"`
	DeletedBy string    `json:"deleted_by"`
	DeletedAt time.Time `json:"deleted_at"`
}

// APIKeyDeletedEvent is emitted when an API key is deleted
type APIKeyDeletedEvent struct {
	APIKeyID  string    `json:"apikey_id"`
	OrgID     string    `json:"org_id"`
	BrandID   string    `json:"brand_id,omitempty"`
	Name      string    `json:"name"`
	DeletedBy string    `json:"deleted_by"`
	DeletedAt time.Time `json:"deleted_at"`
}

// APIKeyToggledEvent is emitted when an API key is toggled
type APIKeyToggledEvent struct {
	APIKeyID  string    `json:"apikey_id"`
	OrgID     string    `json:"org_id"`
	BrandID   string    `json:"brand_id,omitempty"`
	Name      string    `json:"name"`
	Enabled   bool      `json:"enabled"`
	ToggledBy string    `json:"toggled_by"`
	ToggledAt time.Time `json:"toggled_at"`
}

// IntegrationDisconnectedEvent is emitted when integration disconnects
type IntegrationDisconnectedEvent struct {
	IntegrationID  string    `json:"integration_id"`
	OrgID          string    `json:"org_id"`
	BrandID        string    `json:"brand_id,omitempty"`
	Provider       string    `json:"provider"`
	DisconnectedAt time.Time `json:"disconnected_at"`
}

// RulesetCreatedEvent is emitted when a ruleset is created
type RulesetCreatedEvent struct {
	RulesetID string    `json:"ruleset_id"`
	OrgID     string    `json:"org_id"`
	BrandID   string    `json:"brand_id,omitempty"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

// RulesetUpdatedEvent is emitted when a ruleset is updated
type RulesetUpdatedEvent struct {
	RulesetID string                 `json:"ruleset_id"`
	OrgID     string                 `json:"org_id"`
	BrandID   string                 `json:"brand_id,omitempty"`
	Changes   map[string]interface{} `json:"changes"`
	UpdatedBy string                 `json:"updated_by"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// RulesetDeletedEvent is emitted when a ruleset is deleted
type RulesetDeletedEvent struct {
	RulesetID string    `json:"ruleset_id"`
	OrgID     string    `json:"org_id"`
	BrandID   string    `json:"brand_id,omitempty"`
	Name      string    `json:"name"`
	DeletedBy string    `json:"deleted_by"`
	DeletedAt time.Time `json:"deleted_at"`
}

// ExecutionStartedEvent is emitted when execution starts
type ExecutionStartedEvent struct {
	ExecutionID string    `json:"execution_id"`
	OrgID       string    `json:"org_id"`
	BrandID     string    `json:"brand_id,omitempty"`
	RulesetID   string    `json:"ruleset_id"`
	StartedBy   string    `json:"started_by"`
	StartedAt   time.Time `json:"started_at"`
}

// ExecutionCompletedEvent is emitted when execution completes
type ExecutionCompletedEvent struct {
	ExecutionID string    `json:"execution_id"`
	OrgID       string    `json:"org_id"`
	BrandID     string    `json:"brand_id,omitempty"`
	RulesetID   string    `json:"ruleset_id"`
	Result      string    `json:"result"`
	CompletedAt time.Time `json:"completed_at"`
}

// ExecutionFailedEvent is emitted when execution fails
type ExecutionFailedEvent struct {
	ExecutionID string    `json:"execution_id"`
	OrgID       string    `json:"org_id"`
	BrandID     string    `json:"brand_id,omitempty"`
	RulesetID   string    `json:"ruleset_id"`
	Error       string    `json:"error"`
	FailedAt    time.Time `json:"failed_at"`
}

// UserLeftEvent is emitted when a user leaves an organization
type UserLeftEvent struct {
	UserID string    `json:"user_id"`
	OrgID  string    `json:"org_id"`
	Role   string    `json:"role"`
	LeftAt time.Time `json:"left_at"`
}

// CostImportFailedEvent is emitted when cost import fails
type CostImportFailedEvent struct {
	ImportID string    `json:"import_id"`
	OrgID    string    `json:"org_id"`
	BrandID  string    `json:"brand_id,omitempty"`
	Source   string    `json:"source"`
	Error    string    `json:"error"`
	FailedAt time.Time `json:"failed_at"`
}

// KVConfigUpdatedEvent is emitted when KV config is updated
type KVConfigUpdatedEvent struct {
	OrgID     string    `json:"org_id"`
	BrandID   string    `json:"brand_id,omitempty"`
	Namespace string    `json:"namespace"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedBy string    `json:"updated_by"`
	UpdatedAt time.Time `json:"updated_at"`
}

// KVConfigDeletedEvent is emitted when KV config is deleted
type KVConfigDeletedEvent struct {
	OrgID     string    `json:"org_id"`
	BrandID   string    `json:"brand_id,omitempty"`
	Namespace string    `json:"namespace"`
	Key       string    `json:"key"`
	DeletedBy string    `json:"deleted_by"`
	DeletedAt time.Time `json:"deleted_at"`
}

// ETL Pipeline event structures

// ETLStartedEvent is emitted when ETL pipeline starts
type ETLStartedEvent struct {
	Timestamp   time.Time `json:"timestamp"`
	WorkerCount int       `json:"worker_count"`
	CPUCount    int       `json:"cpu_count"`
	MemoryGB    int       `json:"memory_gb"`
}

// ETLStoppedEvent is emitted when ETL pipeline stops
type ETLStoppedEvent struct {
	Timestamp time.Time `json:"timestamp"`
}

// ETLJobQueuedEvent is emitted when a job is queued
type ETLJobQueuedEvent struct {
	JobID    string    `json:"job_id"`
	JobType  string    `json:"job_type"`
	OrgID    string    `json:"org_id"`
	BrandID  string    `json:"brand_id,omitempty"`
	Priority int       `json:"priority"`
	QueuedAt time.Time `json:"queued_at"`
}

// ETLJobStartedEvent is emitted when a job starts processing
type ETLJobStartedEvent struct {
	JobID     string    `json:"job_id"`
	JobType   string    `json:"job_type"`
	WorkerID  int       `json:"worker_id"`
	StartedAt time.Time `json:"started_at"`
}

// ETLJobCompletedEvent is emitted when a job completes
type ETLJobCompletedEvent struct {
	JobID       string    `json:"job_id"`
	JobType     string    `json:"job_type"`
	WorkerID    int       `json:"worker_id"`
	CompletedAt time.Time `json:"completed_at"`
	Duration    int64     `json:"duration_ms"`
}

// ETLJobFailedEvent is emitted when a job fails
type ETLJobFailedEvent struct {
	JobID    string    `json:"job_id"`
	JobType  string    `json:"job_type"`
	WorkerID int       `json:"worker_id"`
	Error    string    `json:"error"`
	FailedAt time.Time `json:"failed_at"`
}

// ETLScalingEvent is emitted when worker count changes
type ETLScalingEvent struct {
	OldWorkerCount int       `json:"old_worker_count"`
	NewWorkerCount int       `json:"new_worker_count"`
	CPUUsage       float64   `json:"cpu_usage"`
	MemoryUsage    float64   `json:"memory_usage"`
	Reason         string    `json:"reason"`
	Timestamp      time.Time `json:"timestamp"`
}

// Webhook event structures

// WebhookReceivedEvent is emitted when a webhook is received from a platform
type WebhookReceivedEvent struct {
	WebhookID   string                 `json:"webhook_id"`
	BrandID     string                 `json:"brand_id"`
	Platform    string                 `json:"platform"`
	EventType   string                 `json:"event_type"`
	Payload     map[string]interface{} `json:"payload"`
	ReceivedAt  time.Time              `json:"received_at"`
	ProcessedAt time.Time              `json:"processed_at,omitempty"`
}

// PlatformDataChangedEvent is emitted when platform data changes and needs to be synced
type PlatformDataChangedEvent struct {
	BrandID    string    `json:"brand_id"`
	Platform   string    `json:"platform"`
	ChangeType string    `json:"change_type"` // campaign_updated, adset_updated, etc.
	EntityID   string    `json:"entity_id"`   // ID of changed entity
	EntityType string    `json:"entity_type"` // campaign, adset, ad, etc.
	Timestamp  time.Time `json:"timestamp"`
}

// PlatformConversionSyncEvent is emitted when platform conversions need to be synced
type PlatformConversionSyncEvent struct {
	BrandID       string    `json:"brand_id"`
	Platform      string    `json:"platform"`
	ConversionID  string    `json:"conversion_id"`
	EventType     string    `json:"event_type"`
	Revenue       int64     `json:"revenue"`
	OrderID       string    `json:"order_id"`
	CustomerEmail string    `json:"customer_email,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
}

// Sync event structures

// SyncStartedEvent is emitted when a sync job starts
type SyncStartedEvent struct {
	SourceConnectionID string    `json:"sourceConnectionId"`
	CollectionID       string    `json:"collectionId"`
	JobID              string    `json:"jobId"`
	Status             string    `json:"status"`
	Timestamp          time.Time `json:"timestamp"`
}

// SyncProgressEvent is emitted for sync progress updates
type SyncProgressEvent struct {
	SourceConnectionID string    `json:"sourceConnectionId"`
	CollectionID       string    `json:"collectionId"`
	JobID              string    `json:"jobId"`
	Status             string    `json:"status"`
	ItemsProcessed     int       `json:"itemsProcessed,omitempty"`
	EntitiesInserted   int       `json:"entitiesInserted,omitempty"`
	EntitiesUpdated    int       `json:"entitiesUpdated,omitempty"`
	EntitiesDeleted    int       `json:"entitiesDeleted,omitempty"`
	EntitiesFailed     int       `json:"entitiesFailed,omitempty"`
	Timestamp          time.Time `json:"timestamp"`
}

// SyncCompletedEvent is emitted when a sync job completes successfully
type SyncCompletedEvent struct {
	SourceConnectionID string    `json:"sourceConnectionId"`
	CollectionID       string    `json:"collectionId"`
	JobID              string    `json:"jobId"`
	Status             string    `json:"status"`
	EntitiesInserted   int       `json:"entitiesInserted,omitempty"`
	EntitiesUpdated    int       `json:"entitiesUpdated,omitempty"`
	EntitiesDeleted    int       `json:"entitiesDeleted,omitempty"`
	EntitiesFailed     int       `json:"entitiesFailed,omitempty"`
	DurationSeconds    int       `json:"durationSeconds,omitempty"`
	Timestamp          time.Time `json:"timestamp"`
}

// SyncFailedEvent is emitted when a sync job fails
type SyncFailedEvent struct {
	SourceConnectionID string    `json:"sourceConnectionId"`
	CollectionID       string    `json:"collectionId"`
	JobID              string    `json:"jobId"`
	Status             string    `json:"status"`
	Error              string    `json:"error,omitempty"`
	DurationSeconds    int       `json:"durationSeconds,omitempty"`
	Timestamp          time.Time `json:"timestamp"`
}

// SyncCancelledEvent is emitted when a sync job is cancelled
type SyncCancelledEvent struct {
	SourceConnectionID string    `json:"sourceConnectionId"`
	CollectionID       string    `json:"collectionId"`
	JobID              string    `json:"jobId"`
	Status             string    `json:"status"`
	Timestamp          time.Time `json:"timestamp"`
}

// CodeIndexStatsEvent is emitted when code index stats are updated
type CodeIndexStatsEvent struct {
	SourceConnectionID string    `json:"sourceConnectionId"`
	RepoURL            string    `json:"repoUrl"`
	RepoName           string    `json:"repoName"`
	SnippetsCount      int       `json:"snippetsCount"`
	DocsCount          int       `json:"docsCount"`
	FilesCount         int       `json:"filesCount"`
	CommitSHA          string    `json:"commitSha,omitempty"`
	Branch             string    `json:"branch,omitempty"`
	Timestamp          time.Time `json:"timestamp"`
}

// =====================================================
// Outlet Business Rules Engine Event Structures
// =====================================================

// TicketEvent is emitted for support ticket lifecycle events
type TicketEvent struct {
	OrgID      string                 `json:"org_id"`
	TicketID   string                 `json:"ticket_id"`
	CustomerID string                 `json:"customer_id,omitempty"`
	Subject    string                 `json:"subject,omitempty"`
	Status     string                 `json:"status,omitempty"`
	Priority   string                 `json:"priority,omitempty"`
	Category   string                 `json:"category,omitempty"`
	AssignedTo string                 `json:"assigned_to,omitempty"`
	Changes    map[string]interface{} `json:"changes,omitempty"` // For update events
	Timestamp  time.Time              `json:"timestamp"`
}

// PaymentEvent is emitted for payment lifecycle events
type PaymentEvent struct {
	OrgID          string    `json:"org_id"`
	PaymentID      string    `json:"payment_id"`
	CustomerID     string    `json:"customer_id"`
	SubscriptionID string    `json:"subscription_id,omitempty"`
	InvoiceID      string    `json:"invoice_id,omitempty"`
	Amount         int64     `json:"amount"` // In cents
	Currency       string    `json:"currency"`
	Status         string    `json:"status"`
	FailureReason  string    `json:"failure_reason,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
}

// SubscriptionEvent is emitted for subscription lifecycle events
type SubscriptionEvent struct {
	OrgID          string    `json:"org_id"`
	SubscriptionID string    `json:"subscription_id"`
	CustomerID     string    `json:"customer_id"`
	ProductID      string    `json:"product_id,omitempty"`
	PriceID        string    `json:"price_id,omitempty"`
	Status         string    `json:"status"`
	PreviousStatus string    `json:"previous_status,omitempty"`
	CancelReason   string    `json:"cancel_reason,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
}

// EmailEvent is emitted for email delivery events
type EmailEvent struct {
	OrgID      string    `json:"org_id"`
	EmailID    string    `json:"email_id"`
	ContactID  string    `json:"contact_id"`
	Email      string    `json:"email,omitempty"` // recipient address
	ListID     string    `json:"list_id,omitempty"`
	SequenceID string    `json:"sequence_id,omitempty"`
	CampaignID string    `json:"campaign_id,omitempty"`
	Subject    string    `json:"subject,omitempty"`
	Status     string    `json:"status"`                // sent, bounced, complained, opened, clicked
	BounceType string    `json:"bounce_type,omitempty"` // hard, soft
	ClickedURL string    `json:"clicked_url,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

// CustomerEvent is emitted for customer lifecycle events
type CustomerEvent struct {
	OrgID             string                 `json:"org_id"`
	CustomerID        string                 `json:"customer_id"`
	Email             string                 `json:"email,omitempty"`
	Name              string                 `json:"name,omitempty"`
	LTV               int64                  `json:"ltv,omitempty"` // Lifetime value in cents
	SubscriptionCount int                    `json:"subscription_count,omitempty"`
	Changes           map[string]interface{} `json:"changes,omitempty"`
	Timestamp         time.Time              `json:"timestamp"`
}

// ContactEvent is emitted for contact/subscriber events
type ContactEvent struct {
	OrgID     string    `json:"org_id"`
	ContactID string    `json:"contact_id"`
	Email     string    `json:"email"`
	ListID    string    `json:"list_id,omitempty"`
	Source    string    `json:"source,omitempty"` // How they were added
	Timestamp time.Time `json:"timestamp"`
}

// RefundEvent is emitted when a refund is issued
type RefundEvent struct {
	OrgID      string    `json:"org_id"`
	RefundID   string    `json:"refund_id"`
	PaymentID  string    `json:"payment_id"`
	CustomerID string    `json:"customer_id"`
	Amount     int64     `json:"amount"` // In cents
	Reason     string    `json:"reason,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

// CheckoutEvent is emitted when a checkout session completes
type CheckoutEvent struct {
	OrgID         string                 `json:"org_id"`
	OrderID       string                 `json:"order_id,omitempty"`
	OrderNumber   string                 `json:"order_number,omitempty"`
	CustomerEmail string                 `json:"customer_email"`
	CustomerName  string                 `json:"customer_name,omitempty"`
	ProductID     string                 `json:"product_id,omitempty"`
	ProductName   string                 `json:"product_name,omitempty"`
	ProductSlug   string                 `json:"product_slug,omitempty"`
	Amount        int64                  `json:"amount"` // In cents
	Currency      string                 `json:"currency"`
	CheckoutType  string                 `json:"checkout_type,omitempty"` // product_checkout, workshop_registration, subscription
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
}

// OrderEvent is emitted for order lifecycle events
type OrderEvent struct {
	OrgID         string          `json:"org_id"`
	OrderID       string          `json:"order_id"`
	OrderNumber   string          `json:"order_number"`
	CustomerEmail string          `json:"customer_email"`
	CustomerName  string          `json:"customer_name,omitempty"`
	Status        string          `json:"status"`
	Amount        int64           `json:"amount"` // In cents
	Currency      string          `json:"currency"`
	Items         []OrderItemInfo `json:"items,omitempty"`
	Timestamp     time.Time       `json:"timestamp"`
}

// OrderItemInfo contains order item details for events
type OrderItemInfo struct {
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	Quantity    int    `json:"quantity"`
	Amount      int64  `json:"amount"` // In cents
}
