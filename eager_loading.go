package onyx

import (
	"fmt"
	"reflect"
	"strings"
)

// EagerLoadingEngine handles eager loading of relationships
type EagerLoadingEngine struct {
	relations map[string]*EagerLoadDefinition
}

// EagerLoadDefinition defines how a relationship should be eagerly loaded
type EagerLoadDefinition struct {
	Name        string
	Constraints func(*QueryBuilder)
	Nested      map[string]*EagerLoadDefinition
}

// NewEagerLoadingEngine creates a new eager loading engine
func NewEagerLoadingEngine() *EagerLoadingEngine {
	return &EagerLoadingEngine{
		relations: make(map[string]*EagerLoadDefinition),
	}
}

// AddRelation adds a relationship to be eagerly loaded
func (ele *EagerLoadingEngine) AddRelation(name string, constraints func(*QueryBuilder)) {
	parts := strings.Split(name, ".")
	current := ele.relations
	
	for i, part := range parts {
		if current[part] == nil {
			current[part] = &EagerLoadDefinition{
				Name:   part,
				Nested: make(map[string]*EagerLoadDefinition),
			}
		}
		
		if i == len(parts)-1 {
			current[part].Constraints = constraints
		} else {
			current = current[part].Nested
		}
	}
}

// LoadForModels loads all registered relationships for the given models
func (ele *EagerLoadingEngine) LoadForModels(models []interface{}) error {
	if len(models) == 0 {
		return nil
	}
	
	for _, definition := range ele.relations {
		err := ele.loadRelationshipForModels(models, definition)
		if err != nil {
			return err
		}
	}
	
	return nil
}

// loadRelationshipForModels loads a specific relationship for models
func (ele *EagerLoadingEngine) loadRelationshipForModels(models []interface{}, definition *EagerLoadDefinition) error {
	// Group models by type
	modelsByType := ele.groupModelsByType(models)
	
	for modelType, typeModels := range modelsByType {
		err := ele.loadRelationshipForModelType(typeModels, modelType, definition)
		if err != nil {
			return err
		}
	}
	
	return nil
}

// loadRelationshipForModelType loads a relationship for a specific model type
func (ele *EagerLoadingEngine) loadRelationshipForModelType(models []interface{}, modelType reflect.Type, definition *EagerLoadDefinition) error {
	// Get relationship factory from registry
	modelName := modelType.Name()
	relationFactory, exists := GetRelationship(modelName, definition.Name)
	if !exists {
		return fmt.Errorf("relationship %s not found for model %s", definition.Name, modelName)
	}
	
	// Create relationship instance
	relationship := relationFactory()
	
	// Apply constraints if any
	if definition.Constraints != nil {
		query := relationship.GetQuery()
		definition.Constraints(query)
	}
	
	// Load based on relationship type
	err := ele.loadByRelationshipType(models, relationship, definition)
	if err != nil {
		return err
	}
	
	// Load nested relationships if any
	if len(definition.Nested) > 0 {
		err = ele.loadNestedRelationships(models, definition)
		if err != nil {
			return err
		}
	}
	
	return nil
}

// loadByRelationshipType loads based on the specific relationship type
func (ele *EagerLoadingEngine) loadByRelationshipType(models []interface{}, relationship Relationship, definition *EagerLoadDefinition) error {
	switch rel := relationship.(type) {
	case *BelongsTo:
		return ele.loadBelongsTo(models, rel, definition.Name)
	case *HasOne:
		return ele.loadHasOne(models, rel, definition.Name)
	case *HasMany:
		return ele.loadHasMany(models, rel, definition.Name)
	case *BelongsToMany:
		return ele.loadBelongsToMany(models, rel, definition.Name)
	case *MorphTo:
		return ele.loadMorphTo(models, rel, definition.Name)
	case *MorphOne:
		return ele.loadMorphOne(models, rel, definition.Name)
	case *MorphMany:
		return ele.loadMorphMany(models, rel, definition.Name)
	case *HasOneThrough:
		return ele.loadHasOneThrough(models, rel, definition.Name)
	case *HasManyThrough:
		return ele.loadHasManyThrough(models, rel, definition.Name)
	default:
		return fmt.Errorf("unsupported relationship type for eager loading")
	}
}

// loadBelongsTo loads belongs to relationships
func (ele *EagerLoadingEngine) loadBelongsTo(models []interface{}, relationship *BelongsTo, relationName string) error {
	// Collect foreign key values
	foreignKeyValues := make([]interface{}, 0, len(models))
	modelMap := make(map[interface{}][]interface{})
	
	for _, model := range models {
		foreignValue := getKeyValue(model, relationship.GetForeignKey())
		if foreignValue != nil {
			foreignKeyValues = append(foreignKeyValues, foreignValue)
			modelMap[foreignValue] = append(modelMap[foreignValue], model)
		}
	}
	
	if len(foreignKeyValues) == 0 {
		return nil
	}
	
	// Query related models
	query := relationship.GetQuery()
	query.WhereIn(relationship.GetLocalKey(), foreignKeyValues)
	
	var relatedModels []interface{}
	err := query.Get(&relatedModels)
	if err != nil {
		return err
	}
	
	// Map related models back to parent models
	return ele.mapBelongsToResults(relatedModels, modelMap, relationship.GetLocalKey(), relationName)
}

// loadHasOne loads has one relationships
func (ele *EagerLoadingEngine) loadHasOne(models []interface{}, relationship *HasOne, relationName string) error {
	// Collect parent key values
	parentKeyValues := make([]interface{}, 0, len(models))
	modelMap := make(map[interface{}]interface{})
	
	for _, model := range models {
		parentValue := getKeyValue(model, relationship.GetLocalKey())
		if parentValue != nil {
			parentKeyValues = append(parentKeyValues, parentValue)
			modelMap[parentValue] = model
		}
	}
	
	if len(parentKeyValues) == 0 {
		return nil
	}
	
	// Query related models
	query := relationship.GetQuery()
	query.WhereIn(relationship.GetForeignKey(), parentKeyValues)
	
	var relatedModels []interface{}
	err := query.Get(&relatedModels)
	if err != nil {
		return err
	}
	
	// Map related models back to parent models
	return ele.mapHasOneResults(relatedModels, modelMap, relationship.GetForeignKey(), relationName)
}

// loadHasMany loads has many relationships
func (ele *EagerLoadingEngine) loadHasMany(models []interface{}, relationship *HasMany, relationName string) error {
	// Collect parent key values
	parentKeyValues := make([]interface{}, 0, len(models))
	modelMap := make(map[interface{}]interface{})
	
	for _, model := range models {
		parentValue := getKeyValue(model, relationship.GetLocalKey())
		if parentValue != nil {
			parentKeyValues = append(parentKeyValues, parentValue)
			modelMap[parentValue] = model
		}
	}
	
	if len(parentKeyValues) == 0 {
		return nil
	}
	
	// Query related models
	query := relationship.GetQuery()
	query.WhereIn(relationship.GetForeignKey(), parentKeyValues)
	
	var relatedModels []interface{}
	err := query.Get(&relatedModels)
	if err != nil {
		return err
	}
	
	// Map related models back to parent models
	return ele.mapHasManyResults(relatedModels, modelMap, relationship.GetForeignKey(), relationName)
}

// loadBelongsToMany loads belongs to many relationships
func (ele *EagerLoadingEngine) loadBelongsToMany(models []interface{}, relationship *BelongsToMany, relationName string) error {
	// This would need access to pivot table information
	// For now, implementing a simplified version
	
	parentKeyValues := make([]interface{}, 0, len(models))
	modelMap := make(map[interface{}]interface{})
	
	for _, model := range models {
		parentValue := getKeyValue(model, relationship.GetLocalKey())
		if parentValue != nil {
			parentKeyValues = append(parentKeyValues, parentValue)
			modelMap[parentValue] = model
		}
	}
	
	if len(parentKeyValues) == 0 {
		return nil
	}
	
	// This would need to query through the pivot table
	// For now, returning without error
	return nil
}

// Result mapping functions

// mapBelongsToResults maps belongs to results to parent models
func (ele *EagerLoadingEngine) mapBelongsToResults(relatedModels []interface{}, modelMap map[interface{}][]interface{}, localKey, relationName string) error {
	results := relatedModels
	
	for _, related := range results {
		localValue := getKeyValue(related, localKey)
		if localValue != nil {
			if parentModels, exists := modelMap[localValue]; exists {
				for _, parent := range parentModels {
					err := SetRelationshipValue(parent, relationName, related)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	
	return nil
}

// mapHasOneResults maps has one results to parent models
func (ele *EagerLoadingEngine) mapHasOneResults(relatedModels []interface{}, modelMap map[interface{}]interface{}, foreignKey, relationName string) error {
	results := relatedModels
	
	for _, related := range results {
		foreignValue := getKeyValue(related, foreignKey)
		if foreignValue != nil {
			if parent, exists := modelMap[foreignValue]; exists {
				err := SetRelationshipValue(parent, relationName, related)
				if err != nil {
					return err
				}
			}
		}
	}
	
	return nil
}

// mapHasManyResults maps has many results to parent models
func (ele *EagerLoadingEngine) mapHasManyResults(relatedModels []interface{}, modelMap map[interface{}]interface{}, foreignKey, relationName string) error {
	// Group related models by foreign key
	relatedByForeignKey := make(map[interface{}][]interface{})
	
	results := relatedModels
	
	for _, related := range results {
		foreignValue := getKeyValue(related, foreignKey)
		if foreignValue != nil {
			relatedByForeignKey[foreignValue] = append(relatedByForeignKey[foreignValue], related)
		}
	}
	
	// Map grouped results to parent models
	for foreignValue, relatedGroup := range relatedByForeignKey {
		if parent, exists := modelMap[foreignValue]; exists {
			err := SetRelationshipValue(parent, relationName, relatedGroup)
			if err != nil {
				return err
			}
		}
	}
	
	// Set empty slices for parents with no related models
	for _, parent := range modelMap {
		currentValue := GetRelationshipValue(parent, relationName)
		if currentValue == nil {
			err := SetRelationshipValue(parent, relationName, []interface{}{})
			if err != nil {
				return err
			}
		}
	}
	
	return nil
}

// loadNestedRelationships loads nested relationships
func (ele *EagerLoadingEngine) loadNestedRelationships(models []interface{}, definition *EagerLoadDefinition) error {
	for _, nestedDefinition := range definition.Nested {
		// Get related models from the current relationship
		relatedModels := make([]interface{}, 0)
		
		for _, model := range models {
			relationValue := GetRelationshipValue(model, definition.Name)
			if relationValue != nil {
				// Handle different types of relationship results
				switch v := relationValue.(type) {
				case []interface{}:
					relatedModels = append(relatedModels, v...)
				case interface{}:
					relatedModels = append(relatedModels, v)
				}
			}
		}
		
		if len(relatedModels) > 0 {
			err := ele.loadRelationshipForModels(relatedModels, nestedDefinition)
			if err != nil {
				return err
			}
		}
	}
	
	return nil
}

// Helper functions

// groupModelsByType groups models by their reflect.Type
func (ele *EagerLoadingEngine) groupModelsByType(models []interface{}) map[reflect.Type][]interface{} {
	groups := make(map[reflect.Type][]interface{})
	
	for _, model := range models {
		modelType := reflect.TypeOf(model)
		if modelType.Kind() == reflect.Ptr {
			modelType = modelType.Elem()
		}
		
		groups[modelType] = append(groups[modelType], model)
	}
	
	return groups
}

// QueryBuilder enhancement for eager loading

// WithEager enables eager loading for queries
func (qb *QueryBuilder) WithEager(relations ...string) *QueryBuilder {
	if qb.eagerLoadEngine == nil {
		qb.eagerLoadEngine = NewEagerLoadingEngine()
	}
	
	for _, relation := range relations {
		qb.eagerLoadEngine.AddRelation(relation, nil)
	}
	
	return qb
}

// WithEagerConstraints enables eager loading with constraints
func (qb *QueryBuilder) WithEagerConstraints(relations map[string]func(*QueryBuilder)) *QueryBuilder {
	if qb.eagerLoadEngine == nil {
		qb.eagerLoadEngine = NewEagerLoadingEngine()
	}
	
	for relation, constraints := range relations {
		qb.eagerLoadEngine.AddRelation(relation, constraints)
	}
	
	return qb
}

// GetWithRelationships executes the query and loads relationships
func (qb *QueryBuilder) GetWithRelationships() ([]interface{}, error) {
	// Execute the base query
	var results []interface{}
	err := qb.Get(&results)
	if err != nil {
		return nil, err
	}
	
	models := results
	
	// Load relationships if eager loading is enabled
	if qb.eagerLoadEngine != nil {
		err = qb.eagerLoadEngine.LoadForModels(models)
		if err != nil {
			return nil, err
		}
	}
	
	return models, nil
}

// FirstWithRelationships gets the first result with relationships
func (qb *QueryBuilder) FirstWithRelationships(dest interface{}) error {
	err := qb.First(dest)
	if err != nil {
		return err
	}
	
	// Load relationships if eager loading is enabled
	if qb.eagerLoadEngine != nil {
		err = qb.eagerLoadEngine.LoadForModels([]interface{}{dest})
		if err != nil {
			return err
		}
	}
	
	return nil
}

// Lazy Loading Support

// LazyLoader provides lazy loading functionality for individual models
type LazyLoader struct {
	model interface{}
}

// NewLazyLoader creates a new lazy loader for a model
func NewLazyLoader(model interface{}) *LazyLoader {
	return &LazyLoader{model: model}
}

// Load loads specific relationships for the model
func (ll *LazyLoader) Load(relations ...string) error {
	engine := NewEagerLoadingEngine()
	
	for _, relation := range relations {
		engine.AddRelation(relation, nil)
	}
	
	return engine.LoadForModels([]interface{}{ll.model})
}

// LoadMissing loads relationships that haven't been loaded yet
func (ll *LazyLoader) LoadMissing(relations ...string) error {
	var missing []string
	
	for _, relation := range relations {
		value := GetRelationshipValue(ll.model, relation)
		if value == nil {
			missing = append(missing, relation)
		}
	}
	
	return ll.Load(missing...)
}

// loadMorphTo loads morph to relationships
func (ele *EagerLoadingEngine) loadMorphTo(models []interface{}, relationship *MorphTo, relationName string) error {
	// MorphTo relationships are more complex and require model registry
	// For now, implementing a simplified version
	return fmt.Errorf("morphTo eager loading not fully implemented")
}

// loadMorphOne loads morph one relationships
func (ele *EagerLoadingEngine) loadMorphOne(models []interface{}, relationship *MorphOne, relationName string) error {
	// Collect parent key values
	parentKeyValues := make([]interface{}, 0, len(models))
	modelMap := make(map[interface{}]interface{})
	
	for _, model := range models {
		parentValue := getKeyValue(model, relationship.GetLocalKey())
		if parentValue != nil {
			parentKeyValues = append(parentKeyValues, parentValue)
			modelMap[parentValue] = model
		}
	}
	
	if len(parentKeyValues) == 0 {
		return nil
	}
	
	// Query related models with morph constraints
	query := relationship.GetQuery()
	query.Where(relationship.morphType, "=", relationship.morphClass).
		WhereIn(relationship.morphId, parentKeyValues)
	
	var relatedModels []interface{}
	err := query.Get(&relatedModels)
	if err != nil {
		return err
	}
	
	// Map related models back to parent models
	return ele.mapMorphOneResults(relatedModels, modelMap, relationship.morphId, relationName)
}

// loadMorphMany loads morph many relationships
func (ele *EagerLoadingEngine) loadMorphMany(models []interface{}, relationship *MorphMany, relationName string) error {
	// Collect parent key values
	parentKeyValues := make([]interface{}, 0, len(models))
	modelMap := make(map[interface{}]interface{})
	
	for _, model := range models {
		parentValue := getKeyValue(model, relationship.GetLocalKey())
		if parentValue != nil {
			parentKeyValues = append(parentKeyValues, parentValue)
			modelMap[parentValue] = model
		}
	}
	
	if len(parentKeyValues) == 0 {
		return nil
	}
	
	// Query related models with morph constraints
	query := relationship.GetQuery()
	query.Where(relationship.morphType, "=", relationship.morphClass).
		WhereIn(relationship.morphId, parentKeyValues)
	
	var relatedModels []interface{}
	err := query.Get(&relatedModels)
	if err != nil {
		return err
	}
	
	// Map related models back to parent models
	return ele.mapMorphManyResults(relatedModels, modelMap, relationship.morphId, relationName)
}

// mapMorphOneResults maps morph one results to parent models
func (ele *EagerLoadingEngine) mapMorphOneResults(relatedModels []interface{}, modelMap map[interface{}]interface{}, morphId, relationName string) error {
	results := relatedModels
	
	for _, related := range results {
		morphIdValue := getKeyValue(related, morphId)
		if morphIdValue != nil {
			if parent, exists := modelMap[morphIdValue]; exists {
				err := SetRelationshipValue(parent, relationName, related)
				if err != nil {
					return err
				}
			}
		}
	}
	
	return nil
}

// mapMorphManyResults maps morph many results to parent models
func (ele *EagerLoadingEngine) mapMorphManyResults(relatedModels []interface{}, modelMap map[interface{}]interface{}, morphId, relationName string) error {
	// Group related models by morph id
	relatedByMorphId := make(map[interface{}][]interface{})
	
	results := relatedModels
	
	for _, related := range results {
		morphIdValue := getKeyValue(related, morphId)
		if morphIdValue != nil {
			relatedByMorphId[morphIdValue] = append(relatedByMorphId[morphIdValue], related)
		}
	}
	
	// Map grouped results to parent models
	for morphIdValue, relatedGroup := range relatedByMorphId {
		if parent, exists := modelMap[morphIdValue]; exists {
			err := SetRelationshipValue(parent, relationName, relatedGroup)
			if err != nil {
				return err
			}
		}
	}
	
	// Set empty slices for parents with no related models
	for _, parent := range modelMap {
		currentValue := GetRelationshipValue(parent, relationName)
		if currentValue == nil {
			err := SetRelationshipValue(parent, relationName, []interface{}{})
			if err != nil {
				return err
			}
		}
	}
	
	return nil
}

// loadHasOneThrough loads has one through relationships
func (ele *EagerLoadingEngine) loadHasOneThrough(models []interface{}, relationship *HasOneThrough, relationName string) error {
	// Collect parent key values
	parentKeyValues := make([]interface{}, 0, len(models))
	modelMap := make(map[interface{}]interface{})
	
	for _, model := range models {
		parentValue := getKeyValue(model, relationship.GetLocalKey())
		if parentValue != nil {
			parentKeyValues = append(parentKeyValues, parentValue)
			modelMap[parentValue] = model
		}
	}
	
	if len(parentKeyValues) == 0 {
		return nil
	}
	
	// Build the through query
	throughTable := getTableName(relationship.throughModel)
	relatedTable := getTableName(relationship.related)
	
	query := relationship.GetQuery()
	query.Table(relatedTable).
		Join(throughTable, fmt.Sprintf("%s.%s", throughTable, relationship.secondLocalKey), "=", fmt.Sprintf("%s.%s", relatedTable, relationship.secondKey)).
		WhereIn(fmt.Sprintf("%s.%s", throughTable, relationship.firstKey), parentKeyValues)
	
	var relatedModels []interface{}
	err := query.Get(&relatedModels)
	if err != nil {
		return err
	}
	
	// Map related models back to parent models
	return ele.mapHasThroughResults(relatedModels, modelMap, throughTable, relationship.firstKey, relationName, false)
}

// loadHasManyThrough loads has many through relationships
func (ele *EagerLoadingEngine) loadHasManyThrough(models []interface{}, relationship *HasManyThrough, relationName string) error {
	// Collect parent key values
	parentKeyValues := make([]interface{}, 0, len(models))
	modelMap := make(map[interface{}]interface{})
	
	for _, model := range models {
		parentValue := getKeyValue(model, relationship.GetLocalKey())
		if parentValue != nil {
			parentKeyValues = append(parentKeyValues, parentValue)
			modelMap[parentValue] = model
		}
	}
	
	if len(parentKeyValues) == 0 {
		return nil
	}
	
	// Build the through query
	throughTable := getTableName(relationship.throughModel)
	relatedTable := getTableName(relationship.related)
	
	query := relationship.GetQuery()
	query.Table(relatedTable).
		Join(throughTable, fmt.Sprintf("%s.%s", throughTable, relationship.secondLocalKey), "=", fmt.Sprintf("%s.%s", relatedTable, relationship.secondKey)).
		WhereIn(fmt.Sprintf("%s.%s", throughTable, relationship.firstKey), parentKeyValues)
	
	var relatedModels []interface{}
	err := query.Get(&relatedModels)
	if err != nil {
		return err
	}
	
	// Map related models back to parent models
	return ele.mapHasThroughResults(relatedModels, modelMap, throughTable, relationship.firstKey, relationName, true)
}

// mapHasThroughResults maps through relationship results to parent models
func (ele *EagerLoadingEngine) mapHasThroughResults(relatedModels []interface{}, modelMap map[interface{}]interface{}, throughTable, throughKey, relationName string, isMany bool) error {
	results := relatedModels
	
	if isMany {
		// Group related models by through key for has many
		relatedByThroughKey := make(map[interface{}][]interface{})
		
		for _, related := range results {
			// This would need to extract the through key from the joined result
			// For simplification, using a placeholder implementation
			throughValue := getKeyValue(related, throughKey)
			if throughValue != nil {
				relatedByThroughKey[throughValue] = append(relatedByThroughKey[throughValue], related)
			}
		}
		
		// Map grouped results to parent models
		for throughValue, relatedGroup := range relatedByThroughKey {
			if parent, exists := modelMap[throughValue]; exists {
				err := SetRelationshipValue(parent, relationName, relatedGroup)
				if err != nil {
					return err
				}
			}
		}
		
		// Set empty slices for parents with no related models
		for _, parent := range modelMap {
			currentValue := GetRelationshipValue(parent, relationName)
			if currentValue == nil {
				err := SetRelationshipValue(parent, relationName, []interface{}{})
				if err != nil {
					return err
				}
			}
		}
	} else {
		// Map single results for has one
		for _, related := range results {
			throughValue := getKeyValue(related, throughKey)
			if throughValue != nil {
				if parent, exists := modelMap[throughValue]; exists {
					err := SetRelationshipValue(parent, relationName, related)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	
	return nil
}

// Model extension for lazy loading
func LoadRelationships(model interface{}, relations ...string) error {
	loader := NewLazyLoader(model)
	return loader.Load(relations...)
}