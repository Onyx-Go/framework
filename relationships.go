package onyx

import (
	"fmt"
	"reflect"
	"strings"
)

// Relationship interfaces and types

// Relationship represents a relationship between models
type Relationship interface {
	GetResults() (interface{}, error)
	AddConstraint(column, operator string, value interface{}) Relationship
	OrderBy(column, direction string) Relationship
	Limit(limit int) Relationship
	GetQuery() *QueryBuilder
	GetRelated() interface{}
	GetParent() interface{}
	GetForeignKey() string
	GetLocalKey() string
}

// RelationshipType defines the type of relationship
type RelationshipType string

const (
	RelationshipBelongsTo    RelationshipType = "belongs_to"
	RelationshipHasOne       RelationshipType = "has_one"
	RelationshipHasMany      RelationshipType = "has_many"
	RelationshipBelongsToMany RelationshipType = "belongs_to_many"
	RelationshipMorphTo      RelationshipType = "morph_to"
	RelationshipMorphOne     RelationshipType = "morph_one"
	RelationshipMorphMany    RelationshipType = "morph_many"
	RelationshipHasOneThrough RelationshipType = "has_one_through"
	RelationshipHasManyThrough RelationshipType = "has_many_through"
)

// BaseRelationship contains common relationship functionality
type BaseRelationship struct {
	parent     interface{}
	related    interface{}
	foreignKey string
	localKey   string
	query      *QueryBuilder
	constraints []QueryConstraint
	orderBy    []OrderByClause
	limitValue *int
	relType    RelationshipType
}

// QueryConstraint represents a query constraint
type QueryConstraint struct {
	Column   string
	Operator string
	Value    interface{}
}

// OrderByClause represents an order by clause
type OrderByClause struct {
	Column    string
	Direction string
}

// NewBaseRelationship creates a new base relationship
func NewBaseRelationship(parent, related interface{}, foreignKey, localKey string, relType RelationshipType) *BaseRelationship {
	return &BaseRelationship{
		parent:      parent,
		related:     related,
		foreignKey:  foreignKey,
		localKey:    localKey,
		relType:     relType,
		constraints: []QueryConstraint{},
		orderBy:     []OrderByClause{},
	}
}

// GetParent returns the parent model
func (br *BaseRelationship) GetParent() interface{} {
	return br.parent
}

// GetRelated returns the related model
func (br *BaseRelationship) GetRelated() interface{} {
	return br.related
}

// GetForeignKey returns the foreign key
func (br *BaseRelationship) GetForeignKey() string {
	return br.foreignKey
}

// GetLocalKey returns the local key
func (br *BaseRelationship) GetLocalKey() string {
	return br.localKey
}

// AddConstraint adds a constraint to the relationship
func (br *BaseRelationship) AddConstraint(column, operator string, value interface{}) Relationship {
	br.constraints = append(br.constraints, QueryConstraint{
		Column:   column,
		Operator: operator,
		Value:    value,
	})
	return br
}

// GetResults is a placeholder implementation - should be overridden by concrete types
func (br *BaseRelationship) GetResults() (interface{}, error) {
	return nil, fmt.Errorf("GetResults must be implemented by concrete relationship types")
}

// OrderBy adds an order by clause
func (br *BaseRelationship) OrderBy(column, direction string) Relationship {
	br.orderBy = append(br.orderBy, OrderByClause{
		Column:    column,
		Direction: direction,
	})
	return br
}

// Limit sets the limit for the relationship query
func (br *BaseRelationship) Limit(limit int) Relationship {
	br.limitValue = &limit
	return br
}

// GetQuery returns the query builder for this relationship
func (br *BaseRelationship) GetQuery() *QueryBuilder {
	if br.query == nil {
		// This would need access to a database instance
		// For now, we'll return an empty query builder that needs to be configured
		br.query = &QueryBuilder{
			selects:         []string{"*"},
			wheres:          []whereClause{},
			orders:          []string{},
			joins:           []string{},
			groupBy:         []string{},
			having:          []whereClause{},
			bindings:        []interface{}{},
			eagerLoad:       make(map[string]interface{}),
			eagerLoadEngine: nil,
		}
		br.setupQuery()
	}
	return br.query
}

// setupQuery sets up the basic query for the relationship
func (br *BaseRelationship) setupQuery() {
	tableName := getTableName(br.related)
	br.query.Table(tableName)
	
	// Add constraints
	for _, constraint := range br.constraints {
		br.query.Where(constraint.Column, constraint.Operator, constraint.Value)
	}
	
	// Add order by clauses
	for _, order := range br.orderBy {
		br.query.OrderBy(order.Column, order.Direction)
	}
	
	// Add limit
	if br.limitValue != nil {
		br.query.Limit(*br.limitValue)
	}
}

// BelongsTo relationship implementation
type BelongsTo struct {
	*BaseRelationship
}

// NewBelongsTo creates a new belongs to relationship
func NewBelongsTo(parent, related interface{}, foreignKey, ownerKey string) *BelongsTo {
	if foreignKey == "" {
		foreignKey = getDefaultForeignKey(related)
	}
	if ownerKey == "" {
		ownerKey = "id"
	}
	
	return &BelongsTo{
		BaseRelationship: NewBaseRelationship(parent, related, foreignKey, ownerKey, RelationshipBelongsTo),
	}
}

// GetResults gets the results for belongs to relationship
func (bt *BelongsTo) GetResults() (interface{}, error) {
	parentValue := getKeyValue(bt.parent, bt.foreignKey)
	if parentValue == nil {
		return nil, nil
	}
	
	var result interface{}
	query := bt.GetQuery()
	err := query.Where(bt.localKey, "=", parentValue).First(&result)
	return result, err
}

// HasOne relationship implementation
type HasOne struct {
	*BaseRelationship
}

// NewHasOne creates a new has one relationship
func NewHasOne(parent, related interface{}, foreignKey, localKey string) *HasOne {
	if foreignKey == "" {
		foreignKey = getDefaultForeignKey(parent)
	}
	if localKey == "" {
		localKey = "id"
	}
	
	return &HasOne{
		BaseRelationship: NewBaseRelationship(parent, related, foreignKey, localKey, RelationshipHasOne),
	}
}

// GetResults gets the results for has one relationship
func (ho *HasOne) GetResults() (interface{}, error) {
	parentValue := getKeyValue(ho.parent, ho.localKey)
	if parentValue == nil {
		return nil, nil
	}
	
	var result interface{}
	query := ho.GetQuery()
	err := query.Where(ho.foreignKey, "=", parentValue).First(&result)
	return result, err
}

// HasMany relationship implementation
type HasMany struct {
	*BaseRelationship
}

// NewHasMany creates a new has many relationship
func NewHasMany(parent, related interface{}, foreignKey, localKey string) *HasMany {
	if foreignKey == "" {
		foreignKey = getDefaultForeignKey(parent)
	}
	if localKey == "" {
		localKey = "id"
	}
	
	return &HasMany{
		BaseRelationship: NewBaseRelationship(parent, related, foreignKey, localKey, RelationshipHasMany),
	}
}

// GetResults gets the results for has many relationship
func (hm *HasMany) GetResults() (interface{}, error) {
	parentValue := getKeyValue(hm.parent, hm.localKey)
	if parentValue == nil {
		return make([]interface{}, 0), nil
	}
	
	var results []interface{}
	query := hm.GetQuery()
	err := query.Where(hm.foreignKey, "=", parentValue).Get(&results)
	return results, err
}

// BelongsToMany relationship implementation
type BelongsToMany struct {
	*BaseRelationship
	pivotTable   string
	foreignPivotKey string
	relatedPivotKey string
	pivotColumns []string
	withTimestamps bool
}

// NewBelongsToMany creates a new belongs to many relationship
func NewBelongsToMany(parent, related interface{}, pivotTable, foreignPivotKey, relatedPivotKey, parentKey, relatedKey string) *BelongsToMany {
	if pivotTable == "" {
		pivotTable = getDefaultPivotTableName(parent, related)
	}
	if foreignPivotKey == "" {
		foreignPivotKey = getDefaultForeignKey(parent)
	}
	if relatedPivotKey == "" {
		relatedPivotKey = getDefaultForeignKey(related)
	}
	if parentKey == "" {
		parentKey = "id"
	}
	if relatedKey == "" {
		relatedKey = "id"
	}
	
	return &BelongsToMany{
		BaseRelationship: NewBaseRelationship(parent, related, foreignPivotKey, parentKey, RelationshipBelongsToMany),
		pivotTable:       pivotTable,
		foreignPivotKey:  foreignPivotKey,
		relatedPivotKey:  relatedPivotKey,
		pivotColumns:     []string{},
		withTimestamps:   false,
	}
}

// WithPivot adds pivot columns to be selected
func (btm *BelongsToMany) WithPivot(columns ...string) *BelongsToMany {
	btm.pivotColumns = append(btm.pivotColumns, columns...)
	return btm
}

// WithTimestamps includes timestamps in pivot table
func (btm *BelongsToMany) WithTimestamps() *BelongsToMany {
	btm.withTimestamps = true
	return btm
}

// GetResults gets the results for belongs to many relationship
func (btm *BelongsToMany) GetResults() (interface{}, error) {
	parentValue := getKeyValue(btm.parent, btm.localKey)
	if parentValue == nil {
		return make([]interface{}, 0), nil
	}
	
	relatedTable := getTableName(btm.related)
	query := btm.GetQuery()
	
	// Build select columns
	selectColumns := []string{fmt.Sprintf("%s.*", relatedTable)}
	
	// Add pivot columns
	for _, col := range btm.pivotColumns {
		selectColumns = append(selectColumns, fmt.Sprintf("%s.%s as pivot_%s", btm.pivotTable, col, col))
	}
	
	// Add timestamps if requested
	if btm.withTimestamps {
		selectColumns = append(selectColumns, 
			fmt.Sprintf("%s.created_at as pivot_created_at", btm.pivotTable),
			fmt.Sprintf("%s.updated_at as pivot_updated_at", btm.pivotTable))
	}
	
	// Join with pivot table
	query.Select(selectColumns...).
		Join(btm.pivotTable, fmt.Sprintf("%s.%s", relatedTable, "id"), "=", fmt.Sprintf("%s.%s", btm.pivotTable, btm.relatedPivotKey)).
		Where(fmt.Sprintf("%s.%s", btm.pivotTable, btm.foreignPivotKey), "=", parentValue)
	
	var results []interface{}
	err := query.Get(&results)
	return results, err
}

// Attach attaches related models to the pivot table
func (btm *BelongsToMany) Attach(relatedIds []interface{}, pivotData map[string]interface{}) error {
	parentValue := getKeyValue(btm.parent, btm.localKey)
	if parentValue == nil {
		return fmt.Errorf("parent key value is nil")
	}
	
	// This would need a database connection
	// For now, return nil to avoid compilation errors
	// Implementation pending proper database instance management
	_ = parentValue
	_ = relatedIds
	_ = pivotData
	
	return nil
}

// Detach removes related models from the pivot table
func (btm *BelongsToMany) Detach(relatedIds []interface{}) error {
	parentValue := getKeyValue(btm.parent, btm.localKey)
	if parentValue == nil {
		return fmt.Errorf("parent key value is nil")
	}
	
	// This would need a database connection
	// For now, return nil to avoid compilation errors
	// Implementation pending proper database instance management
	_ = parentValue
	_ = relatedIds
	
	return nil
}

// Sync synchronizes the pivot table with the given related IDs
func (btm *BelongsToMany) Sync(relatedIds []interface{}, pivotData map[string]interface{}) error {
	// First, remove all existing relationships
	err := btm.Detach([]interface{}{})
	if err != nil {
		return err
	}
	
	// Then attach the new ones
	return btm.Attach(relatedIds, pivotData)
}

// EagerLoader handles eager loading of relationships
type EagerLoader struct {
	relations map[string]func([]interface{}) error
}

// NewEagerLoader creates a new eager loader
func NewEagerLoader() *EagerLoader {
	return &EagerLoader{
		relations: make(map[string]func([]interface{}) error),
	}
}

// AddRelation adds a relation to be eagerly loaded
func (el *EagerLoader) AddRelation(name string, loader func([]interface{}) error) {
	el.relations[name] = loader
}

// Load loads all registered relations for the given models
func (el *EagerLoader) Load(models []interface{}) error {
	for name, loader := range el.relations {
		err := loader(models)
		if err != nil {
			return fmt.Errorf("failed to load relation %s: %v", name, err)
		}
	}
	return nil
}

// Helper functions

// getTableName gets the table name for a model
func getTableName(model interface{}) string {
	if tableable, ok := model.(interface{ TableName() string }); ok {
		return tableable.TableName()
	}
	
	// Default table name based on struct name
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	
	name := t.Name()
	return strings.ToLower(name) + "s" // Simple pluralization
}

// getDefaultForeignKey gets the default foreign key for a model
func getDefaultForeignKey(model interface{}) string {
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	
	name := strings.ToLower(t.Name())
	return name + "_id"
}

// getDefaultPivotTableName gets the default pivot table name for two models
func getDefaultPivotTableName(model1, model2 interface{}) string {
	name1 := getModelName(model1)
	name2 := getModelName(model2)
	
	// Alphabetical order
	if name1 > name2 {
		name1, name2 = name2, name1
	}
	
	return name1 + "_" + name2
}

// getModelName gets the model name
func getModelName(model interface{}) string {
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return strings.ToLower(t.Name())
}

// getKeyValue gets the value of a key from a model
func getKeyValue(model interface{}, key string) interface{} {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	
	if v.Kind() != reflect.Struct {
		return nil
	}
	
	// Try to find field by name (case insensitive)
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)
		
		// Check json tag first
		if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
			tagParts := strings.Split(tag, ",")
			if tagParts[0] == key {
				return fieldValue.Interface()
			}
		}
		
		// Check field name
		if strings.EqualFold(field.Name, key) {
			return fieldValue.Interface()
		}
		
		// Check db tag
		if tag := field.Tag.Get("db"); tag == key {
			return fieldValue.Interface()
		}
		
		// Check embedded structs
		if field.Anonymous && fieldValue.Kind() == reflect.Struct {
			if result := getKeyValueFromStruct(fieldValue, key); result != nil {
				return result
			}
		}
	}
	
	return nil
}

// getKeyValueFromStruct gets the value of a key from a struct value
func getKeyValueFromStruct(v reflect.Value, key string) interface{} {
	if v.Kind() != reflect.Struct {
		return nil
	}
	
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)
		
		// Check json tag first
		if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
			tagParts := strings.Split(tag, ",")
			if tagParts[0] == key {
				return fieldValue.Interface()
			}
		}
		
		// Check field name
		if strings.EqualFold(field.Name, key) {
			return fieldValue.Interface()
		}
		
		// Check db tag
		if tag := field.Tag.Get("db"); tag == key {
			return fieldValue.Interface()
		}
		
		// Check embedded structs recursively
		if field.Anonymous && fieldValue.Kind() == reflect.Struct {
			if result := getKeyValueFromStruct(fieldValue, key); result != nil {
				return result
			}
		}
	}
	
	return nil
}

// getCurrentTimestamp returns the current timestamp
func getCurrentTimestamp() interface{} {
	// This would be implemented based on your timestamp format
	// For now, returning a simple string
	return "2024-01-01 00:00:00"
}

// RelationshipModelInterface that models should implement for relationships
type RelationshipModelInterface interface {
	BelongsTo(related interface{}, foreignKey, ownerKey string) *BelongsTo
	HasOne(related interface{}, foreignKey, localKey string) *HasOne
	HasMany(related interface{}, foreignKey, localKey string) *HasMany
	BelongsToMany(related interface{}, pivotTable, foreignPivotKey, relatedPivotKey, parentKey, relatedKey string) *BelongsToMany
}

// RelationshipModel provides relationship methods for models
type RelationshipModel struct{}

// BelongsTo creates a belongs to relationship
func (rm *RelationshipModel) BelongsTo(parent interface{}, related interface{}, foreignKey, ownerKey string) *BelongsTo {
	return NewBelongsTo(parent, related, foreignKey, ownerKey)
}

// HasOne creates a has one relationship
func (rm *RelationshipModel) HasOne(parent interface{}, related interface{}, foreignKey, localKey string) *HasOne {
	return NewHasOne(parent, related, foreignKey, localKey)
}

// HasMany creates a has many relationship
func (rm *RelationshipModel) HasMany(parent interface{}, related interface{}, foreignKey, localKey string) *HasMany {
	return NewHasMany(parent, related, foreignKey, localKey)
}

// BelongsToMany creates a belongs to many relationship
func (rm *RelationshipModel) BelongsToMany(parent interface{}, related interface{}, pivotTable, foreignPivotKey, relatedPivotKey, parentKey, relatedKey string) *BelongsToMany {
	return NewBelongsToMany(parent, related, pivotTable, foreignPivotKey, relatedPivotKey, parentKey, relatedKey)
}

// MorphTo creates a morph to relationship
func (rm *RelationshipModel) MorphTo(parent interface{}, morphType, morphId string) *MorphTo {
	return NewMorphTo(parent, morphType, morphId)
}

// MorphOne creates a morph one relationship
func (rm *RelationshipModel) MorphOne(parent interface{}, related interface{}, morphType, morphId, parentKey string) *MorphOne {
	return NewMorphOne(parent, related, morphType, morphId, parentKey)
}

// MorphMany creates a morph many relationship
func (rm *RelationshipModel) MorphMany(parent interface{}, related interface{}, morphType, morphId, parentKey string) *MorphMany {
	return NewMorphMany(parent, related, morphType, morphId, parentKey)
}

// HasOneThrough creates a has one through relationship
func (rm *RelationshipModel) HasOneThrough(parent interface{}, related interface{}, through interface{}, firstKey, secondKey, localKey, secondLocalKey string) *HasOneThrough {
	return NewHasOneThrough(parent, related, through, firstKey, secondKey, localKey, secondLocalKey)
}

// HasManyThrough creates a has many through relationship
func (rm *RelationshipModel) HasManyThrough(parent interface{}, related interface{}, through interface{}, firstKey, secondKey, localKey, secondLocalKey string) *HasManyThrough {
	return NewHasManyThrough(parent, related, through, firstKey, secondKey, localKey, secondLocalKey)
}

// RelationshipRegistry manages relationship definitions
type RelationshipRegistry struct {
	relationships map[string]map[string]func() Relationship
}

// NewRelationshipRegistry creates a new relationship registry
func NewRelationshipRegistry() *RelationshipRegistry {
	return &RelationshipRegistry{
		relationships: make(map[string]map[string]func() Relationship),
	}
}

// RegisterRelationship registers a relationship for a model
func (rr *RelationshipRegistry) RegisterRelationship(modelName, relationName string, factory func() Relationship) {
	if rr.relationships[modelName] == nil {
		rr.relationships[modelName] = make(map[string]func() Relationship)
	}
	rr.relationships[modelName][relationName] = factory
}

// GetRelationship gets a relationship factory
func (rr *RelationshipRegistry) GetRelationship(modelName, relationName string) (func() Relationship, bool) {
	if modelRelations, exists := rr.relationships[modelName]; exists {
		factory, exists := modelRelations[relationName]
		return factory, exists
	}
	return nil, false
}

// MorphTo relationship implementation  
type MorphTo struct {
	*BaseRelationship
	morphType   string
	morphId     string
	morphClass  string
}

// NewMorphTo creates a new morph to relationship
func NewMorphTo(parent interface{}, morphType, morphId string) *MorphTo {
	return &MorphTo{
		BaseRelationship: NewBaseRelationship(parent, nil, morphId, "id", RelationshipMorphTo),
		morphType:        morphType,
		morphId:          morphId,
	}
}

// GetResults gets the results for morph to relationship
func (mt *MorphTo) GetResults() (interface{}, error) {
	morphTypeValue := getKeyValue(mt.parent, mt.morphType)
	morphIdValue := getKeyValue(mt.parent, mt.morphId)
	
	if morphTypeValue == nil || morphIdValue == nil {
		return nil, nil
	}
	
	// This would need a model registry to map type to actual model
	// For now, return nil as a placeholder
	return nil, fmt.Errorf("morphTo implementation requires model registry")
}

// MorphOne relationship implementation
type MorphOne struct {
	*BaseRelationship
	morphType   string
	morphId     string
	morphClass  string
}

// NewMorphOne creates a new morph one relationship
func NewMorphOne(parent, related interface{}, morphType, morphId, parentKey string) *MorphOne {
	if parentKey == "" {
		parentKey = "id"
	}
	
	return &MorphOne{
		BaseRelationship: NewBaseRelationship(parent, related, morphId, parentKey, RelationshipMorphOne),
		morphType:        morphType,
		morphId:          morphId,
		morphClass:       getModelName(parent),
	}
}

// GetResults gets the results for morph one relationship
func (mo *MorphOne) GetResults() (interface{}, error) {
	parentValue := getKeyValue(mo.parent, mo.localKey)
	if parentValue == nil {
		return nil, nil
	}
	
	var result interface{}
	query := mo.GetQuery()
	err := query.Where(mo.morphType, "=", mo.morphClass).
		Where(mo.morphId, "=", parentValue).
		First(&result)
	return result, err
}

// MorphMany relationship implementation
type MorphMany struct {
	*BaseRelationship
	morphType   string
	morphId     string
	morphClass  string
}

// NewMorphMany creates a new morph many relationship
func NewMorphMany(parent, related interface{}, morphType, morphId, parentKey string) *MorphMany {
	if parentKey == "" {
		parentKey = "id"
	}
	
	return &MorphMany{
		BaseRelationship: NewBaseRelationship(parent, related, morphId, parentKey, RelationshipMorphMany),
		morphType:        morphType,
		morphId:          morphId,
		morphClass:       getModelName(parent),
	}
}

// GetResults gets the results for morph many relationship
func (mm *MorphMany) GetResults() (interface{}, error) {
	parentValue := getKeyValue(mm.parent, mm.localKey)
	if parentValue == nil {
		return make([]interface{}, 0), nil
	}
	
	var results []interface{}
	query := mm.GetQuery()
	err := query.Where(mm.morphType, "=", mm.morphClass).
		Where(mm.morphId, "=", parentValue).
		Get(&results)
	return results, err
}

// HasOneThrough relationship implementation
type HasOneThrough struct {
	*BaseRelationship
	throughModel interface{}
	firstKey     string
	secondKey    string
	localKey     string
	secondLocalKey string
}

// NewHasOneThrough creates a new has one through relationship
func NewHasOneThrough(parent, related, through interface{}, firstKey, secondKey, localKey, secondLocalKey string) *HasOneThrough {
	if firstKey == "" {
		firstKey = getDefaultForeignKey(parent)
	}
	if secondKey == "" {
		secondKey = getDefaultForeignKey(through)
	}
	if localKey == "" {
		localKey = "id"
	}
	if secondLocalKey == "" {
		secondLocalKey = "id"
	}
	
	return &HasOneThrough{
		BaseRelationship: NewBaseRelationship(parent, related, firstKey, localKey, RelationshipHasOneThrough),
		throughModel:     through,
		firstKey:         firstKey,
		secondKey:        secondKey,
		localKey:         localKey,
		secondLocalKey:   secondLocalKey,
	}
}

// GetResults gets the results for has one through relationship
func (hot *HasOneThrough) GetResults() (interface{}, error) {
	parentValue := getKeyValue(hot.parent, hot.localKey)
	if parentValue == nil {
		return nil, nil
	}
	
	// Build the through query
	throughTable := getTableName(hot.throughModel)
	relatedTable := getTableName(hot.related)
	
	query := hot.GetQuery()
	query.Table(relatedTable).
		Join(throughTable, fmt.Sprintf("%s.%s", throughTable, hot.secondLocalKey), "=", fmt.Sprintf("%s.%s", relatedTable, hot.secondKey)).
		Where(fmt.Sprintf("%s.%s", throughTable, hot.firstKey), "=", parentValue)
	
	var result interface{}
	err := query.First(&result)
	return result, err
}

// HasManyThrough relationship implementation
type HasManyThrough struct {
	*BaseRelationship
	throughModel interface{}
	firstKey     string
	secondKey    string
	localKey     string
	secondLocalKey string
}

// NewHasManyThrough creates a new has many through relationship
func NewHasManyThrough(parent, related, through interface{}, firstKey, secondKey, localKey, secondLocalKey string) *HasManyThrough {
	if firstKey == "" {
		firstKey = getDefaultForeignKey(parent)
	}
	if secondKey == "" {
		secondKey = getDefaultForeignKey(through)
	}
	if localKey == "" {
		localKey = "id"
	}
	if secondLocalKey == "" {
		secondLocalKey = "id"
	}
	
	return &HasManyThrough{
		BaseRelationship: NewBaseRelationship(parent, related, firstKey, localKey, RelationshipHasManyThrough),
		throughModel:     through,
		firstKey:         firstKey,
		secondKey:        secondKey,
		localKey:         localKey,
		secondLocalKey:   secondLocalKey,
	}
}

// GetResults gets the results for has many through relationship
func (hmt *HasManyThrough) GetResults() (interface{}, error) {
	parentValue := getKeyValue(hmt.parent, hmt.localKey)
	if parentValue == nil {
		return make([]interface{}, 0), nil
	}
	
	// Build the through query
	throughTable := getTableName(hmt.throughModel)
	relatedTable := getTableName(hmt.related)
	
	query := hmt.GetQuery()
	query.Table(relatedTable).
		Join(throughTable, fmt.Sprintf("%s.%s", throughTable, hmt.secondLocalKey), "=", fmt.Sprintf("%s.%s", relatedTable, hmt.secondKey)).
		Where(fmt.Sprintf("%s.%s", throughTable, hmt.firstKey), "=", parentValue)
	
	var results []interface{}
	err := query.Get(&results)
	return results, err
}

// Global relationship registry
var globalRelationshipRegistry = NewRelationshipRegistry()

// RegisterRelationship registers a relationship globally
func RegisterRelationship(modelName, relationName string, factory func() Relationship) {
	globalRelationshipRegistry.RegisterRelationship(modelName, relationName, factory)
}

// GetRelationship gets a relationship from the global registry
func GetRelationship(modelName, relationName string) (func() Relationship, bool) {
	return globalRelationshipRegistry.GetRelationship(modelName, relationName)
}