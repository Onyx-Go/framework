package migrations

// indexBuilder implements the Index interface
type indexBuilder struct {
	name      string
	columns   []string
	indexType string // "", "unique", "primary", "spatial", "fulltext"
	algorithm string
	where     string
}

// Basic properties

// Name sets the index name
func (ib *indexBuilder) Name(name string) Index {
	ib.name = name
	return ib
}

// Algorithm sets the index algorithm
func (ib *indexBuilder) Algorithm(algorithm string) Index {
	ib.algorithm = algorithm
	return ib
}

// Index types

// Unique marks the index as unique
func (ib *indexBuilder) Unique() Index {
	ib.indexType = "unique"
	return ib
}

// Spatial marks the index as spatial
func (ib *indexBuilder) Spatial() Index {
	ib.indexType = "spatial"
	return ib
}

// FullText marks the index as full-text
func (ib *indexBuilder) FullText() Index {
	ib.indexType = "fulltext"
	return ib
}

// Conditions (PostgreSQL partial indexes)

// Where adds a condition for partial indexes
func (ib *indexBuilder) Where(condition string) Index {
	ib.where = condition
	return ib
}

// GetDefinition returns the index definition for SQL generation
func (ib *indexBuilder) GetDefinition() IndexDefinition {
	indexType := ib.indexType
	if indexType == "primary" {
		indexType = ""
	}
	
	return IndexDefinition{
		Name:      ib.name,
		Columns:   ib.columns,
		Type:      indexType,
		Algorithm: ib.algorithm,
		Where:     ib.where,
	}
}

// Ensure indexBuilder implements Index
var _ Index = (*indexBuilder)(nil)