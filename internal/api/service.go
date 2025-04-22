package api

import (
	"context"
	"fmt"
	"sync"
	"time"

	apiv1 "stahl/api/gen/go"
	"stahl/internal/app/stahl"
	"stahl/internal/config"
	"stahl/internal/schema"
	"stahl/internal/schema/pg"
	"stahl/internal/worker"

	"github.com/jmoiron/sqlx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ConfigurationService implements API for Stahl configuration management
type ConfigurationService struct {
	apiv1.UnimplementedConfigurationServiceServer

	mu           sync.RWMutex
	config       config.Summary
	manager      worker.IManager
	schema       schema.ISchemaGenerator
	db           *sqlx.DB
	startTime    time.Time
	isRunning    bool
	changeTables []string
}

// NewConfigurationService creates a new instance of ConfigurationService
func NewConfigurationService(cfg config.Summary, manager worker.IManager, schema schema.ISchemaGenerator) *ConfigurationService {
	// Get the database connection from the schema generator
	var db *sqlx.DB
	if pgSchema, ok := schema.(*pg.SchemaGenerator); ok {
		db = pgSchema.GetDB()
	}

	return &ConfigurationService{
		config:    cfg,
		manager:   manager,
		schema:    schema,
		db:        db,
		startTime: time.Now(),
		isRunning: true,
	}
}

// GetConfig returns the current system configuration
func (s *ConfigurationService) GetConfig(ctx context.Context, req *apiv1.GetConfigRequest) (*apiv1.GetConfigResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Convert current configuration to API format
	apiConfig := &apiv1.StahlConfig{
		Service: &apiv1.ServiceConfig{
			MetricsPort: uint32(s.config.Service.MetricsPort),
		},
		Db: &apiv1.DatabaseConfig{
			DriverName:  s.config.Drivers.Db.DriverName,
			Host:        s.config.Drivers.Db.Host,
			Port:        uint32(s.config.Drivers.Db.Port),
			User:        s.config.Drivers.Db.User,
			Password:    s.config.Drivers.Db.Password,
			Database:    s.config.Drivers.Db.Database,
			PingTimeout: s.config.Drivers.Db.PingTimeout.String(),
			PingPeriod:  s.config.Drivers.Db.PingPeriod.String(),
		},
		Kafka: &apiv1.KafkaConfig{
			DriverName:   s.config.Drivers.Output.DriverName,
			TableChannel: s.config.Drivers.Output.TableChannel,
			Brokers:      s.config.Drivers.Output.Brokers,
		},
		Schema: &apiv1.SchemaConfig{
			Tables: s.config.UserSchema.TableNames,
		},
		Producer: &apiv1.ProducerConfig{
			Period:      s.config.Producer.Period.String(),
			StopOnError: s.config.Producer.StopOnError,
			EnableDlq:   s.config.Producer.EnableDLQ,
		},
		Consumer: &apiv1.ConsumerConfig{
			MaxBatchWait:  s.config.Consumer.MaxBatchWait.String(),
			MaxTasksBatch: int32(s.config.Consumer.MaxTasksBatch),
			StopOnError:   s.config.Consumer.StopOnError,
			EnableDlq:     s.config.Consumer.EnableDLQ,
		},
	}

	return &apiv1.GetConfigResponse{
		Config: apiConfig,
	}, nil
}

// UpdateConfig updates the system configuration
func (s *ConfigurationService) UpdateConfig(ctx context.Context, req *apiv1.UpdateConfigRequest) (*apiv1.UpdateConfigResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if req.Config == nil {
		return &apiv1.UpdateConfigResponse{
			Success: false,
			Message: "Invalid configuration: config is nil",
		}, status.Error(codes.InvalidArgument, "Config is nil")
	}

	// Parse time duration strings
	producerPeriod, err := time.ParseDuration(req.Config.Producer.Period)
	if err != nil {
		return &apiv1.UpdateConfigResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid producer period format: %v", err),
		}, status.Error(codes.InvalidArgument, "Invalid producer period format")
	}

	consumerMaxBatchWait, err := time.ParseDuration(req.Config.Consumer.MaxBatchWait)
	if err != nil {
		return &apiv1.UpdateConfigResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid consumer max batch wait format: %v", err),
		}, status.Error(codes.InvalidArgument, "Invalid consumer max batch wait format")
	}

	dbPingTimeout, err := time.ParseDuration(req.Config.Db.PingTimeout)
	if err != nil {
		return &apiv1.UpdateConfigResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid DB ping timeout format: %v", err),
		}, status.Error(codes.InvalidArgument, "Invalid DB ping timeout format")
	}

	dbPingPeriod, err := time.ParseDuration(req.Config.Db.PingPeriod)
	if err != nil {
		return &apiv1.UpdateConfigResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid DB ping period format: %v", err),
		}, status.Error(codes.InvalidArgument, "Invalid DB ping period format")
	}

	// Create new configuration
	newConfig := config.Summary{
		UserSchema: config.UserSchemaConfig{
			TableNames: req.Config.Schema.Tables,
		},
		Producer: config.ProducerConfig{
			Period:      producerPeriod,
			StopOnError: req.Config.Producer.StopOnError,
			EnableDLQ:   req.Config.Producer.EnableDlq,
		},
		Drivers: struct {
			Db     config.DatabaseConfig `yaml:"db"`
			Output config.OutputConfig   `yaml:"output"`
		}{
			Db: config.DatabaseConfig{
				DriverName:  req.Config.Db.DriverName,
				Host:        req.Config.Db.Host,
				Port:        uint16(req.Config.Db.Port),
				User:        req.Config.Db.User,
				Password:    req.Config.Db.Password,
				Database:    req.Config.Db.Database,
				PingTimeout: dbPingTimeout,
				PingPeriod:  dbPingPeriod,
			},
			Output: config.OutputConfig{
				DriverName:   req.Config.Kafka.DriverName,
				TableChannel: req.Config.Kafka.TableChannel,
				Brokers:      req.Config.Kafka.Brokers,
			},
		},
		Consumer: config.ConsumerConfig{
			MaxTasksBatch: int(req.Config.Consumer.MaxTasksBatch),
			MaxBatchWait:  consumerMaxBatchWait,
			StopOnError:   req.Config.Consumer.StopOnError,
			EnableDLQ:     req.Config.Consumer.EnableDlq,
		},
		Service: config.ServiceConfig{
			MetricsPort: uint16(req.Config.Service.MetricsPort),
		},
		Schema: s.config.Schema, // Keep the current schema configuration
	}

	// Restart the manager with the new configuration
	if err := s.restartManager(ctx, newConfig); err != nil {
		return &apiv1.UpdateConfigResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to restart manager: %v", err),
		}, status.Error(codes.Internal, "Failed to restart manager")
	}

	// Update the service configuration
	s.config = newConfig

	return &apiv1.UpdateConfigResponse{
		Success: true,
		Message: "Configuration updated successfully",
	}, nil
}

// AddTable adds a new table for replication
func (s *ConfigurationService) AddTable(ctx context.Context, req *apiv1.AddTableRequest) (*apiv1.AddTableResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if req.TableName == "" {
		return &apiv1.AddTableResponse{
			Success: false,
			Message: "Table name cannot be empty",
		}, status.Error(codes.InvalidArgument, "Table name cannot be empty")
	}

	if req.KafkaTopic == "" {
		return &apiv1.AddTableResponse{
			Success: false,
			Message: "Kafka topic cannot be empty",
		}, status.Error(codes.InvalidArgument, "Kafka topic cannot be empty")
	}

	// Check if table already exists in the configuration
	for _, table := range s.config.UserSchema.TableNames {
		if table == req.TableName {
			return &apiv1.AddTableResponse{
				Success: false,
				Message: fmt.Sprintf("Table %s is already being replicated", req.TableName),
			}, status.Error(codes.AlreadyExists, "Table already being replicated")
		}
	}

	// Add the table to the configuration
	s.config.UserSchema.TableNames = append(s.config.UserSchema.TableNames, req.TableName)

	// Add the Kafka topic mapping
	if s.config.Drivers.Output.TableChannel == nil {
		s.config.Drivers.Output.TableChannel = make(map[string]string)
	}
	s.config.Drivers.Output.TableChannel[req.TableName] = req.KafkaTopic

	// Create changelog table, trigger and procedure for the new table
	schemaConfig := s.schema.GetConfig()
	schemaConfig.TableNames = s.config.UserSchema.TableNames

	if err := s.createChangelogForTable(ctx, req.TableName); err != nil {
		// Rollback configuration changes if we fail to create the changelog
		s.config.UserSchema.TableNames = s.config.UserSchema.TableNames[:len(s.config.UserSchema.TableNames)-1]
		delete(s.config.Drivers.Output.TableChannel, req.TableName)

		return &apiv1.AddTableResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create changelog for table: %v", err),
		}, status.Error(codes.Internal, "Failed to create changelog")
	}

	// Restart the manager with the updated configuration
	if err := s.restartManager(ctx, s.config); err != nil {
		return &apiv1.AddTableResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to restart manager: %v", err),
		}, status.Error(codes.Internal, "Failed to restart manager")
	}

	return &apiv1.AddTableResponse{
		Success: true,
		Message: fmt.Sprintf("Table %s added for replication", req.TableName),
	}, nil
}

// RemoveTable removes a table from replication
func (s *ConfigurationService) RemoveTable(ctx context.Context, req *apiv1.RemoveTableRequest) (*apiv1.RemoveTableResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if req.TableName == "" {
		return &apiv1.RemoveTableResponse{
			Success: false,
			Message: "Table name cannot be empty",
		}, status.Error(codes.InvalidArgument, "Table name cannot be empty")
	}

	// Check if table exists in the configuration
	tableExists := false
	var tableIndex int
	for i, table := range s.config.UserSchema.TableNames {
		if table == req.TableName {
			tableExists = true
			tableIndex = i
			break
		}
	}

	if !tableExists {
		return &apiv1.RemoveTableResponse{
			Success: false,
			Message: fmt.Sprintf("Table %s is not being replicated", req.TableName),
		}, status.Error(codes.NotFound, "Table not found")
	}

	// Remove the table from the configuration
	s.config.UserSchema.TableNames = append(
		s.config.UserSchema.TableNames[:tableIndex],
		s.config.UserSchema.TableNames[tableIndex+1:]...,
	)

	// Remove the changelog table and trigger for the table
	if err := s.removeChangelogForTable(ctx, req.TableName); err != nil {
		// Rollback configuration changes if we fail to remove the changelog
		s.config.UserSchema.TableNames = append(
			s.config.UserSchema.TableNames[:tableIndex],
			append([]string{req.TableName}, s.config.UserSchema.TableNames[tableIndex:]...)...,
		)

		return &apiv1.RemoveTableResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to remove changelog for table: %v", err),
		}, status.Error(codes.Internal, "Failed to remove changelog")
	}

	// Remove the Kafka topic mapping
	delete(s.config.Drivers.Output.TableChannel, req.TableName)

	// Restart the manager with the updated configuration
	if err := s.restartManager(ctx, s.config); err != nil {
		return &apiv1.RemoveTableResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to restart manager: %v", err),
		}, status.Error(codes.Internal, "Failed to restart manager")
	}

	return &apiv1.RemoveTableResponse{
		Success: true,
		Message: fmt.Sprintf("Table %s removed from replication", req.TableName),
	}, nil
}

// ListTables returns a list of replicated tables
func (s *ConfigurationService) ListTables(ctx context.Context, req *apiv1.ListTablesRequest) (*apiv1.ListTablesResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &apiv1.ListTablesResponse{
		Tables:       s.config.UserSchema.TableNames,
		TableMapping: s.config.Drivers.Output.TableChannel,
	}, nil
}

// UpdateKafkaConfig updates the Kafka configuration
func (s *ConfigurationService) UpdateKafkaConfig(ctx context.Context, req *apiv1.UpdateKafkaConfigRequest) (*apiv1.UpdateKafkaConfigResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if req.KafkaConfig == nil {
		return &apiv1.UpdateKafkaConfigResponse{
			Success: false,
			Message: "Kafka configuration cannot be nil",
		}, status.Error(codes.InvalidArgument, "Kafka configuration is nil")
	}

	// Update the Kafka configuration
	s.config.Drivers.Output.DriverName = req.KafkaConfig.DriverName
	s.config.Drivers.Output.TableChannel = req.KafkaConfig.TableChannel
	s.config.Drivers.Output.Brokers = req.KafkaConfig.Brokers

	// Restart the manager with the updated configuration
	if err := s.restartManager(ctx, s.config); err != nil {
		return &apiv1.UpdateKafkaConfigResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to restart manager: %v", err),
		}, status.Error(codes.Internal, "Failed to restart manager")
	}

	return &apiv1.UpdateKafkaConfigResponse{
		Success: true,
		Message: "Kafka configuration updated successfully",
	}, nil
}

// UpdateTableMapping updates the mapping of tables to Kafka topics
func (s *ConfigurationService) UpdateTableMapping(ctx context.Context, req *apiv1.UpdateTableMappingRequest) (*apiv1.UpdateTableMappingResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if req.TableMapping == nil {
		return &apiv1.UpdateTableMappingResponse{
			Success: false,
			Message: "Table mapping cannot be nil",
		}, status.Error(codes.InvalidArgument, "Table mapping is nil")
	}

	// Check that all tables in the mapping exist in the configuration
	for table := range req.TableMapping {
		tableExists := false
		for _, configTable := range s.config.UserSchema.TableNames {
			if table == configTable {
				tableExists = true
				break
			}
		}
		if !tableExists {
			return &apiv1.UpdateTableMappingResponse{
				Success: false,
				Message: fmt.Sprintf("Table %s is not being replicated", table),
			}, status.Error(codes.NotFound, "Table not found")
		}
	}

	// Update the table mapping
	s.config.Drivers.Output.TableChannel = req.TableMapping

	// Restart the manager with the updated configuration
	if err := s.restartManager(ctx, s.config); err != nil {
		return &apiv1.UpdateTableMappingResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to restart manager: %v", err),
		}, status.Error(codes.Internal, "Failed to restart manager")
	}

	return &apiv1.UpdateTableMappingResponse{
		Success: true,
		Message: "Table mapping updated successfully",
	}, nil
}

// GetStatus returns the current replication status
func (s *ConfigurationService) GetStatus(ctx context.Context, req *apiv1.GetStatusRequest) (*apiv1.GetStatusResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Get real status information from the manager
	tableStatuses := make([]*apiv1.TableStatus, len(s.config.UserSchema.TableNames))
	for i, tableName := range s.config.UserSchema.TableNames {
		changelogTable := ""
		if s.config.Schema.ChangelogTableNames != nil {
			changelogTable = s.config.Schema.ChangelogTableNames[tableName]
		}

		// Get metrics from the manager if available
		processedEvents, pendingEvents := s.manager.GetTableStats(ctx, tableName)
		lastProcessedTime := s.manager.GetLastProcessedTime(ctx, tableName)

		tableStatuses[i] = &apiv1.TableStatus{
			TableName:       tableName,
			ChangelogTable:  changelogTable,
			ProcessedEvents: processedEvents,
			PendingEvents:   pendingEvents,
			LastProcessedAt: lastProcessedTime.Format(time.RFC3339),
		}
	}

	return &apiv1.GetStatusResponse{
		TableStatuses:           tableStatuses,
		ReplicationRunningSince: s.startTime.Format(time.RFC3339),
	}, nil
}

// Helper methods

// restartManager stops the current manager and starts a new one with the updated configuration
func (s *ConfigurationService) restartManager(ctx context.Context, cfg config.Summary) error {
	// First, stop the current manager if it exists
	if s.manager != nil {
		// Call Stop() to gracefully stop existing workers
		s.manager.Stop()

		// Give some time for the manager to stop all operations
		time.Sleep(100 * time.Millisecond)
	}

	// Create a new manager with the updated configuration
	newManager, err := createManager(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create new manager: %w", err)
	}

	// Start the new manager with a controlled context
	runCtx, cancel := context.WithCancel(ctx)

	// Start producers first to initialize changelog monitoring
	if err := newManager.StartProducers(runCtx); err != nil {
		cancel() // Cancel the context if producers fail to start
		return fmt.Errorf("failed to start producers: %w", err)
	}

	// Then start consumers to process events
	if err := newManager.StartConsumers(runCtx); err != nil {
		cancel() // Cancel the context if consumers fail to start
		return fmt.Errorf("failed to start consumers: %w", err)
	}

	// Get error channel for monitoring
	errCh := newManager.Start(runCtx)

	// Monitor for errors in a separate goroutine
	go func() {
		select {
		case err := <-errCh:
			if err != nil {
				// Log the error
				fmt.Printf("Manager error: %v\n", err)

				// Try to restart the manager if needed
				// This is a simple retry mechanism
				time.Sleep(5 * time.Second)
				if s.isRunning {
					fmt.Println("Attempting to restart manager...")
					if restartErr := s.restartManager(context.Background(), s.config); restartErr != nil {
						fmt.Printf("Failed to restart manager: %v\n", restartErr)
					}
				}
			}
		case <-runCtx.Done():
			// Context done, manager being stopped deliberately
			fmt.Println("Manager context cancelled, stopping workers...")
		}
	}()

	// Update the manager reference
	s.manager = newManager

	return nil
}

// createManager creates a new manager with the given configuration
func createManager(ctx context.Context, cfg config.Summary) (worker.IManager, error) {
	// Call the existing stahl.GetManager function directly
	m, err := stahl.GetManager(cfg)
	if err != nil {
		return nil, fmt.Errorf("stahl.GetManager: %w", err)
	}
	return m, nil
}

// createChangelogForTable creates a changelog table, trigger and procedure for a table
func (s *ConfigurationService) createChangelogForTable(ctx context.Context, tableName string) error {
	if s.db == nil {
		return fmt.Errorf("database connection not available")
	}

	// Use the SchemaGenerator to create the changelog table and trigger
	tableGenerator, err := pg.NewTableSchemaGenerator(s.db, tableName, s.config.Schema.ReplicationStatusTableName)
	if err != nil {
		return fmt.Errorf("failed to create table schema generator: %w", err)
	}

	// Create the changelog table
	changelogTableName, err := tableGenerator.CreateChangelogTable(ctx)
	if err != nil {
		return fmt.Errorf("failed to create changelog table: %w", err)
	}

	// Create the trigger
	if err := tableGenerator.CreateTrigger(ctx, changelogTableName); err != nil {
		return fmt.Errorf("failed to create trigger: %w", err)
	}

	// Update the changelog table mapping
	if s.config.Schema.ChangelogTableNames == nil {
		s.config.Schema.ChangelogTableNames = make(map[string]string)
	}
	s.config.Schema.ChangelogTableNames[tableName] = changelogTableName

	return nil
}

// removeChangelogForTable removes the changelog table, trigger and procedure for a table
func (s *ConfigurationService) removeChangelogForTable(ctx context.Context, tableName string) error {
	if s.db == nil {
		return fmt.Errorf("database connection not available")
	}

	// Get the changelog table name
	changelogTableName, exists := s.config.Schema.ChangelogTableNames[tableName]
	if !exists {
		return fmt.Errorf("changelog table not found for table %s", tableName)
	}

	// Use the SchemaGenerator to remove the trigger and changelog table
	tableGenerator, err := pg.NewTableSchemaGenerator(s.db, tableName, s.config.Schema.ReplicationStatusTableName)
	if err != nil {
		return fmt.Errorf("failed to create table schema generator: %w", err)
	}

	// Drop the trigger
	if err := tableGenerator.DropTrigger(ctx); err != nil {
		return fmt.Errorf("failed to drop trigger: %w", err)
	}

	// Drop the changelog table
	if err := tableGenerator.DropChangelogTable(ctx, changelogTableName); err != nil {
		return fmt.Errorf("failed to drop changelog table: %w", err)
	}

	// Remove the table from the changelog table mapping
	delete(s.config.Schema.ChangelogTableNames, tableName)

	return nil
}
