package migrations

import (
	"database/sql"
	"fmt"
)

// BaseMigration provides base functionality for migrations
type BaseMigration struct {
	name    string
	batch   int
	schema  SchemaBuilder
	config  *MigrationConfig
}

// NewBaseMigration creates a new base migration
func NewBaseMigration(name string, schema SchemaBuilder) *BaseMigration {
	return &BaseMigration{
		name:   name,
		batch:  0,
		schema: schema,
	}
}

// GetName returns the migration name
func (bm *BaseMigration) GetName() string {
	return bm.name
}

// GetBatch returns the batch number
func (bm *BaseMigration) GetBatch() int {
	return bm.batch
}

// SetBatch sets the batch number
func (bm *BaseMigration) SetBatch(batch int) {
	bm.batch = batch
}

// GetSchema returns the schema builder
func (bm *BaseMigration) GetSchema() SchemaBuilder {
	return bm.schema
}

// SetSchema sets the schema builder
func (bm *BaseMigration) SetSchema(schema SchemaBuilder) {
	bm.schema = schema
}

// GetConfig returns the migration configuration
func (bm *BaseMigration) GetConfig() *MigrationConfig {
	return bm.config
}

// SetConfig sets the migration configuration
func (bm *BaseMigration) SetConfig(config *MigrationConfig) {
	bm.config = config
}

// Up runs the migration (to be implemented by concrete migrations)
func (bm *BaseMigration) Up() error {
	return fmt.Errorf("Up() method must be implemented by concrete migration")
}

// Down rolls back the migration (to be implemented by concrete migrations)
func (bm *BaseMigration) Down() error {
	return fmt.Errorf("Down() method must be implemented by concrete migration")
}

// Helper methods for common migration operations

// CreateTable creates a new table with the given callback
func (bm *BaseMigration) CreateTable(tableName string, callback func(Table)) error {
	if bm.schema == nil {
		return fmt.Errorf("schema builder not available")
	}
	return bm.schema.Create(tableName, callback)
}

// AlterTable alters an existing table with the given callback
func (bm *BaseMigration) AlterTable(tableName string, callback func(Table)) error {
	if bm.schema == nil {
		return fmt.Errorf("schema builder not available")
	}
	return bm.schema.Alter(tableName, callback)
}

// DropTable drops a table
func (bm *BaseMigration) DropTable(tableName string) error {
	if bm.schema == nil {
		return fmt.Errorf("schema builder not available")
	}
	return bm.schema.Drop(tableName)
}

// DropTableIfExists drops a table if it exists
func (bm *BaseMigration) DropTableIfExists(tableName string) error {
	if bm.schema == nil {
		return fmt.Errorf("schema builder not available")
	}
	return bm.schema.DropIfExists(tableName)
}

// RenameTable renames a table
func (bm *BaseMigration) RenameTable(from, to string) error {
	if bm.schema == nil {
		return fmt.Errorf("schema builder not available")
	}
	return bm.schema.Rename(from, to)
}

// HasTable checks if a table exists
func (bm *BaseMigration) HasTable(tableName string) (bool, error) {
	if bm.schema == nil {
		return false, fmt.Errorf("schema builder not available")
	}
	return bm.schema.HasTable(tableName)
}

// HasColumn checks if a column exists in a table
func (bm *BaseMigration) HasColumn(tableName, columnName string) (bool, error) {
	if bm.schema == nil {
		return false, fmt.Errorf("schema builder not available")
	}
	return bm.schema.HasColumn(tableName, columnName)
}

// HasIndex checks if an index exists
func (bm *BaseMigration) HasIndex(tableName, indexName string) (bool, error) {
	if bm.schema == nil {
		return false, fmt.Errorf("schema builder not available")
	}
	return bm.schema.HasIndex(tableName, indexName)
}

// HasForeignKey checks if a foreign key exists
func (bm *BaseMigration) HasForeignKey(tableName, keyName string) (bool, error) {
	if bm.schema == nil {
		return false, fmt.Errorf("schema builder not available")
	}
	return bm.schema.HasForeignKey(tableName, keyName)
}

// ExecuteRaw executes raw SQL
func (bm *BaseMigration) ExecuteRaw(sql string, bindings ...interface{}) error {
	if bm.schema == nil {
		return fmt.Errorf("schema builder not available")
	}
	return bm.schema.Raw(sql, bindings...)
}

// GetConnection returns the database connection
func (bm *BaseMigration) GetConnection() *sql.DB {
	if bm.schema == nil {
		return nil
	}
	return bm.schema.GetConnection()
}

// GetDriverName returns the database driver name
func (bm *BaseMigration) GetDriverName() string {
	if bm.schema == nil {
		return ""
	}
	return bm.schema.GetDriverName()
}

// String returns a string representation of the migration
func (bm *BaseMigration) String() string {
	return fmt.Sprintf("Migration{name: %s, batch: %d}", bm.name, bm.batch)
}

// ExampleMigration is an example of how to create a migration
type ExampleMigration struct {
	*BaseMigration
}

// NewExampleMigration creates a new example migration
func NewExampleMigration(schema SchemaBuilder) *ExampleMigration {
	return &ExampleMigration{
		BaseMigration: NewBaseMigration("create_users_table", schema),
	}
}

// Up creates the users table
func (em *ExampleMigration) Up() error {
	return em.CreateTable("users", func(table Table) {
		table.ID()
		table.String("name", 255)
		table.String("email", 255).Unique()
		table.String("password", 255)
		table.RememberToken()
		table.Timestamps()
	})
}

// Down drops the users table
func (em *ExampleMigration) Down() error {
	return em.DropTable("users")
}

// CreateUsersTableMigration is a concrete example migration
type CreateUsersTableMigration struct {
	*BaseMigration
}

// NewCreateUsersTableMigration creates a new users table migration
func NewCreateUsersTableMigration(schema SchemaBuilder) *CreateUsersTableMigration {
	return &CreateUsersTableMigration{
		BaseMigration: NewBaseMigration("2023_01_01_000000_create_users_table", schema),
	}
}

// Up creates the users table with comprehensive structure
func (cutm *CreateUsersTableMigration) Up() error {
	return cutm.CreateTable("users", func(table Table) {
		table.ID()
		table.String("name", 255).NotNull()
		table.String("email", 255).Unique().NotNull()
		table.String("email_verified_at", 255).Nullable()
		table.String("password", 255).NotNull()
		table.RememberToken()
		table.Timestamps()
		table.SoftDeletes()
		
		// Add indexes
		table.Index("email")
		table.Index("created_at")
	})
}

// Down drops the users table
func (cutm *CreateUsersTableMigration) Down() error {
	return cutm.DropTable("users")
}

// CreatePasswordResetsTableMigration creates password resets table
type CreatePasswordResetsTableMigration struct {
	*BaseMigration
}

// NewCreatePasswordResetsTableMigration creates a new password resets migration
func NewCreatePasswordResetsTableMigration(schema SchemaBuilder) *CreatePasswordResetsTableMigration {
	return &CreatePasswordResetsTableMigration{
		BaseMigration: NewBaseMigration("2023_01_01_000001_create_password_resets_table", schema),
	}
}

// Up creates the password resets table
func (cprtm *CreatePasswordResetsTableMigration) Up() error {
	return cprtm.CreateTable("password_resets", func(table Table) {
		table.String("email", 255)
		table.String("token", 255)
		table.Timestamp("created_at").Nullable()
		
		// Add index on email
		table.Index("email")
	})
}

// Down drops the password resets table
func (cprtm *CreatePasswordResetsTableMigration) Down() error {
	return cprtm.DropTable("password_resets")
}

// CreateFailedJobsTableMigration creates failed jobs table
type CreateFailedJobsTableMigration struct {
	*BaseMigration
}

// NewCreateFailedJobsTableMigration creates a new failed jobs migration
func NewCreateFailedJobsTableMigration(schema SchemaBuilder) *CreateFailedJobsTableMigration {
	return &CreateFailedJobsTableMigration{
		BaseMigration: NewBaseMigration("2023_01_01_000002_create_failed_jobs_table", schema),
	}
}

// Up creates the failed jobs table
func (cfjtm *CreateFailedJobsTableMigration) Up() error {
	return cfjtm.CreateTable("failed_jobs", func(table Table) {
		table.ID()
		table.String("uuid", 255).Unique()
		table.Text("connection")
		table.Text("queue")
		table.LongText("payload")
		table.LongText("exception")
		table.Timestamp("failed_at").Default("CURRENT_TIMESTAMP")
		
		// Add index on uuid
		table.Index("uuid")
	})
}

// Down drops the failed jobs table
func (cfjtm *CreateFailedJobsTableMigration) Down() error {
	return cfjtm.DropTable("failed_jobs")
}

// MigrationRegistry holds registered migrations
type MigrationRegistry struct {
	migrations map[string]func(SchemaBuilder) Migration
}

// NewMigrationRegistry creates a new migration registry
func NewMigrationRegistry() *MigrationRegistry {
	return &MigrationRegistry{
		migrations: make(map[string]func(SchemaBuilder) Migration),
	}
}

// Register registers a migration factory function
func (mr *MigrationRegistry) Register(name string, factory func(SchemaBuilder) Migration) {
	mr.migrations[name] = factory
}

// Get retrieves a migration by name
func (mr *MigrationRegistry) Get(name string, schema SchemaBuilder) (Migration, bool) {
	if factory, exists := mr.migrations[name]; exists {
		return factory(schema), true
	}
	return nil, false
}

// GetAll returns all registered migration names
func (mr *MigrationRegistry) GetAll() []string {
	names := make([]string, 0, len(mr.migrations))
	for name := range mr.migrations {
		names = append(names, name)
	}
	return names
}

// Clear clears all registered migrations
func (mr *MigrationRegistry) Clear() {
	mr.migrations = make(map[string]func(SchemaBuilder) Migration)
}

// RegisterCommonMigrations registers common framework migrations
func (mr *MigrationRegistry) RegisterCommonMigrations() {
	mr.Register("create_users_table", func(schema SchemaBuilder) Migration {
		return NewCreateUsersTableMigration(schema)
	})
	
	mr.Register("create_password_resets_table", func(schema SchemaBuilder) Migration {
		return NewCreatePasswordResetsTableMigration(schema)
	})
	
	mr.Register("create_failed_jobs_table", func(schema SchemaBuilder) Migration {
		return NewCreateFailedJobsTableMigration(schema)
	})
}