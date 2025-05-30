package onyx

import (
	"fmt"
	"reflect"
	"strings"
)

// QueryBuilder extensions for relationships

// With loads relationships eagerly
func (qb *QueryBuilder) With(relations ...string) *QueryBuilder {
	if qb.eagerLoad == nil {
		qb.eagerLoad = make(map[string]interface{})
	}
	
	for _, relation := range relations {
		qb.eagerLoad[relation] = nil
	}
	
	return qb
}

// WithConstraints loads relationships with constraints
func (qb *QueryBuilder) WithConstraints(relations map[string]func(*QueryBuilder)) *QueryBuilder {
	if qb.eagerLoad == nil {
		qb.eagerLoad = make(map[string]interface{})
	}
	
	for relation, constraint := range relations {
		qb.eagerLoad[relation] = constraint
	}
	
	return qb
}

// WhereHas adds a where clause based on relationship existence
func (qb *QueryBuilder) WhereHas(relation string, callback func(*QueryBuilder), operator string, count int) *QueryBuilder {
	subQuery := qb.getRelationshipSubQuery(relation, callback)
	
	if subQuery != nil {
		// Add existence constraint
		qb.whereExists(subQuery, operator, count)
	}
	
	return qb
}

// WhereDoesntHave adds a where clause for relationship non-existence
func (qb *QueryBuilder) WhereDoesntHave(relation string, callback func(*QueryBuilder)) *QueryBuilder {
	subQuery := qb.getRelationshipSubQuery(relation, callback)
	
	if subQuery != nil {
		qb.whereNotExists(subQuery)
	}
	
	return qb
}

// DoesntHave adds a constraint for models that don't have a relationship
func (qb *QueryBuilder) DoesntHave(relation string) *QueryBuilder {
	return qb.WhereDoesntHave(relation, nil)
}

// Has adds a constraint for models that have a relationship
func (qb *QueryBuilder) Has(relation string, operator string, count int) *QueryBuilder {
	return qb.WhereHas(relation, nil, operator, count)
}

// WithCount adds a count of related models
func (qb *QueryBuilder) WithCount(relations ...string) *QueryBuilder {
	for _, relation := range relations {
		countColumn := relation + "_count"
		subQuery := qb.getRelationshipCountSubQuery(relation)
		
		if subQuery != nil {
			qb.selectRaw(fmt.Sprintf("(%s) as %s", subQuery.toSQL(), countColumn))
		}
	}
	
	return qb
}

// WithCountConstraints adds counts with constraints
func (qb *QueryBuilder) WithCountConstraints(relations map[string]func(*QueryBuilder)) *QueryBuilder {
	for relation, constraint := range relations {
		countColumn := relation + "_count"
		subQuery := qb.getRelationshipCountSubQuery(relation)
		
		if subQuery != nil && constraint != nil {
			constraint(subQuery)
			qb.selectRaw(fmt.Sprintf("(%s) as %s", subQuery.toSQL(), countColumn))
		}
	}
	
	return qb
}

// LoadRelationships loads relationships for the given models
func (qb *QueryBuilder) LoadRelationships(models []interface{}) error {
	if qb.eagerLoad == nil || len(qb.eagerLoad) == 0 {
		return nil
	}
	
	for relation, constraint := range qb.eagerLoad {
		err := qb.loadRelationship(models, relation, constraint)
		if err != nil {
			return fmt.Errorf("failed to load relationship %s: %v", relation, err)
		}
	}
	
	return nil
}

// Helper methods for relationship queries

// getRelationshipSubQuery creates a subquery for relationship constraints
func (qb *QueryBuilder) getRelationshipSubQuery(relation string, callback func(*QueryBuilder)) *QueryBuilder {
	// Get the model type to find relationship definitions
	if qb.table == "" {
		return nil
	}
	
	// Create a subquery for the relationship
	subQuery := NewQueryBuilder(qb.db)
	
	// This is a simplified implementation - in a real system you would:
	// 1. Parse the relation name to find the relationship definition
	// 2. Determine the relationship type (belongsTo, hasMany, etc.)
	// 3. Set up the appropriate table and where conditions
	
	// For demonstration, assume it's a basic hasMany relationship
	relationTable := relation + "s" // Simple pluralization
	subQuery.Table(relationTable)
	
	// Apply the callback constraints if provided
	if callback != nil {
		callback(subQuery)
	}
	
	// Add the relationship constraint (this would be dynamic based on the relationship type)
	foreignKey := qb.table[:len(qb.table)-1] + "_id" // Remove 's' and add '_id'
	subQuery.whereRaw(fmt.Sprintf("%s.%s = %s.id", relationTable, foreignKey, qb.table))
	
	return subQuery
}

// getRelationshipCountSubQuery creates a count subquery for relationships
func (qb *QueryBuilder) getRelationshipCountSubQuery(relation string) *QueryBuilder {
	if qb.table == "" {
		return nil
	}
	
	// Create a count subquery for the relationship
	subQuery := NewQueryBuilder(qb.db)
	
	// For demonstration, assume it's a basic hasMany relationship
	relationTable := relation + "s" // Simple pluralization
	subQuery.Table(relationTable)
	subQuery.selectRaw("COUNT(*)")
	
	// Add the relationship constraint
	foreignKey := qb.table[:len(qb.table)-1] + "_id" // Remove 's' and add '_id'
	subQuery.whereRaw(fmt.Sprintf("%s.%s = %s.id", relationTable, foreignKey, qb.table))
	
	return subQuery
}

// whereExists adds an exists constraint
func (qb *QueryBuilder) whereExists(subQuery *QueryBuilder, operator string, count int) *QueryBuilder {
	if subQuery == nil {
		return qb
	}
	
	sql := subQuery.toSQL()
	
	if operator == "" {
		operator = ">="
	}
	
	if count == 0 {
		count = 1
	}
	
	// For count-based constraints, use a count subquery
	if count != 1 || operator != ">=" {
		countSQL := fmt.Sprintf("(%s) %s %d", sql, operator, count)
		qb.whereRaw(countSQL)
	} else {
		// Simple exists check
		existsSQL := fmt.Sprintf("EXISTS (%s)", sql)
		qb.whereRaw(existsSQL)
	}
	
	return qb
}

// whereNotExists adds a not exists constraint
func (qb *QueryBuilder) whereNotExists(subQuery *QueryBuilder) *QueryBuilder {
	if subQuery == nil {
		return qb
	}
	
	sql := subQuery.toSQL()
	existsSQL := fmt.Sprintf("NOT EXISTS (%s)", sql)
	qb.whereRaw(existsSQL)
	
	return qb
}

// loadRelationship loads a specific relationship for models
func (qb *QueryBuilder) loadRelationship(models []interface{}, relation string, constraint interface{}) error {
	if len(models) == 0 {
		return nil
	}
	
	// Parse nested relationships (e.g., "user.posts")
	parts := strings.Split(relation, ".")
	currentRelation := parts[0]
	
	// Get relationship definition for the current model type
	modelType := reflect.TypeOf(models[0])
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}
	
	// Load the relationship based on its type
	err := qb.loadRelationshipByType(models, currentRelation, constraint)
	if err != nil {
		return err
	}
	
	// If there are nested relationships, load them recursively
	if len(parts) > 1 {
		nestedRelation := strings.Join(parts[1:], ".")
		return qb.loadNestedRelationships(models, currentRelation, nestedRelation, constraint)
	}
	
	return nil
}

// loadRelationshipByType loads a relationship based on its type
func (qb *QueryBuilder) loadRelationshipByType(models []interface{}, relation string, constraint interface{}) error {
	// This would determine the relationship type and load accordingly
	// For now, implementing a basic version
	
	// Collect parent IDs
	parentIds := make([]interface{}, 0, len(models))
	for _, model := range models {
		id := getKeyValue(model, "id")
		if id != nil {
			parentIds = append(parentIds, id)
		}
	}
	
	if len(parentIds) == 0 {
		return nil
	}
	
	// Create a basic relationship query
	// This would need to be enhanced based on actual relationship definitions
	relatedQuery := NewQueryBuilder(qb.db)
	
	// Apply constraints if provided
	if constraintFunc, ok := constraint.(func(*QueryBuilder)); ok && constraintFunc != nil {
		constraintFunc(relatedQuery)
	}
	
	// Execute the relationship query
	// This is a simplified implementation
	return nil
}

// loadNestedRelationships loads nested relationships
func (qb *QueryBuilder) loadNestedRelationships(models []interface{}, currentRelation, nestedRelation string, constraint interface{}) error {
	// Get related models from the current relationship
	relatedModels := make([]interface{}, 0)
	
	for range models {
		// Get the related models for this parent
		// This would need to access the loaded relationship data
		// For now, this is a placeholder
	}
	
	if len(relatedModels) > 0 {
		// Recursively load the nested relationships
		return qb.loadRelationship(relatedModels, nestedRelation, constraint)
	}
	
	return nil
}

// Relationship query scopes

// WhereRelation adds a where clause for a specific relationship
func (qb *QueryBuilder) WhereRelation(relation, column, operator string, value interface{}) *QueryBuilder {
	// Join the related table and add the where clause
	// This would need to be implemented based on relationship definitions
	return qb
}

// OrWhereRelation adds an OR where clause for a specific relationship
func (qb *QueryBuilder) OrWhereRelation(relation, column, operator string, value interface{}) *QueryBuilder {
	// Similar to WhereRelation but with OR
	return qb
}

// WhereRelationIn adds a where in clause for a relationship
func (qb *QueryBuilder) WhereRelationIn(relation, column string, values []interface{}) *QueryBuilder {
	// Join the related table and add the where in clause
	return qb
}

// QueryBuilder extensions for eager loading

// eagerLoad field needs to be added to QueryBuilder
type QueryBuilderWithEagerLoad struct {
	*QueryBuilder
	eagerLoad map[string]interface{}
}

// NewQueryBuilderWithEagerLoad creates a new query builder with eager loading support
func NewQueryBuilderWithEagerLoad(db *DB) *QueryBuilderWithEagerLoad {
	return &QueryBuilderWithEagerLoad{
		QueryBuilder: NewQueryBuilder(db),
		eagerLoad:    make(map[string]interface{}),
	}
}

// Relationship Collection helpers

// RelationshipCollection manages a collection of models with relationships
type RelationshipCollection struct {
	models []interface{}
	loaded map[string]bool
}

// NewRelationshipCollection creates a new relationship collection
func NewRelationshipCollection(models []interface{}) *RelationshipCollection {
	return &RelationshipCollection{
		models: models,
		loaded: make(map[string]bool),
	}
}

// Load loads relationships for all models in the collection
func (rc *RelationshipCollection) Load(relations ...string) error {
	for _, relation := range relations {
		if rc.loaded[relation] {
			continue // Already loaded
		}
		
		err := rc.loadRelation(relation)
		if err != nil {
			return err
		}
		
		rc.loaded[relation] = true
	}
	
	return nil
}

// LoadMissing loads relationships that haven't been loaded yet
func (rc *RelationshipCollection) LoadMissing(relations ...string) error {
	var missing []string
	
	for _, relation := range relations {
		if !rc.loaded[relation] {
			missing = append(missing, relation)
		}
	}
	
	return rc.Load(missing...)
}

// loadRelation loads a specific relationship
func (rc *RelationshipCollection) loadRelation(relation string) error {
	// Implementation would depend on relationship definitions
	// This is a placeholder
	return nil
}

// GetModels returns the models in the collection
func (rc *RelationshipCollection) GetModels() []interface{} {
	return rc.models
}

// IsLoaded checks if a relationship has been loaded
func (rc *RelationshipCollection) IsLoaded(relation string) bool {
	return rc.loaded[relation]
}

// Relationship helper functions

// GetRelationshipValue gets the value of a relationship from a model
func GetRelationshipValue(model interface{}, relationName string) interface{} {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	
	if v.Kind() != reflect.Struct {
		return nil
	}
	
	// Look for a field with the relationship name
	field := v.FieldByName(relationName)
	if field.IsValid() {
		return field.Interface()
	}
	
	return nil
}

// SetRelationshipValue sets the value of a relationship on a model
func SetRelationshipValue(model interface{}, relationName string, value interface{}) error {
	v := reflect.ValueOf(model)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("model must be a pointer to set relationship value")
	}
	
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("model must be a struct")
	}
	
	field := v.FieldByName(relationName)
	if !field.IsValid() {
		return fmt.Errorf("relationship field %s not found", relationName)
	}
	
	if !field.CanSet() {
		return fmt.Errorf("relationship field %s cannot be set", relationName)
	}
	
	valueToSet := reflect.ValueOf(value)
	if !valueToSet.Type().AssignableTo(field.Type()) {
		return fmt.Errorf("value type %s is not assignable to field type %s", 
			valueToSet.Type(), field.Type())
	}
	
	field.Set(valueToSet)
	return nil
}

// CreateRelationshipIndex creates database indexes for relationship foreign keys
// func CreateRelationshipIndex(tableName, columnName string) error {
// 	// This would need a global database instance
// 	// Implementation pending proper database instance management
// 	return nil
// }

// DropRelationshipIndex drops database indexes for relationship foreign keys  
// func DropRelationshipIndex(tableName, columnName string) error {
// 	// This would need a global database instance
// 	// Implementation pending proper database instance management
// 	return nil
// }