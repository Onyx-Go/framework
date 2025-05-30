package onyx

import (
	"testing"
)

// Test models for relationship testing
type RelUser struct {
	BaseModel
	Name  string `db:"name" json:"name"`
	Email string `db:"email" json:"email"`
	
	// Relationships
	Profile *RelProfile        `json:"profile,omitempty"`
	Posts   []*RelPost         `json:"posts,omitempty"`
	Tags    []*RelTag          `json:"tags,omitempty"`
	Comments []*RelComment     `json:"comments,omitempty"`
}

func (u *RelUser) TableName() string {
	return "users"
}

type RelProfile struct {
	BaseModel
	UserID uint   `db:"user_id" json:"user_id"`
	Bio    string `db:"bio" json:"bio"`
	
	// Relationships
	User *RelUser `json:"user,omitempty"`
}

func (p *RelProfile) TableName() string {
	return "profiles"
}

type RelPost struct {
	BaseModel
	UserID uint   `db:"user_id" json:"user_id"`
	Title  string `db:"title" json:"title"`
	Body   string `db:"body" json:"body"`
	
	// Relationships
	User     *RelUser      `json:"user,omitempty"`
	Comments []*RelComment `json:"comments,omitempty"`
	Tags     []*RelTag     `json:"tags,omitempty"`
}

func (p *RelPost) TableName() string {
	return "posts"
}

type RelComment struct {
	BaseModel
	PostID          uint   `db:"post_id" json:"post_id"`
	UserID          uint   `db:"user_id" json:"user_id"`
	Content         string `db:"content" json:"content"`
	CommentableType string `db:"commentable_type" json:"commentable_type"`
	CommentableID   uint   `db:"commentable_id" json:"commentable_id"`
	
	// Relationships
	Post *RelPost `json:"post,omitempty"`
	User *RelUser `json:"user,omitempty"`
}

func (c *RelComment) TableName() string {
	return "comments"
}

type RelTag struct {
	BaseModel
	Name string `db:"name" json:"name"`
	
	// Relationships
	Posts []*RelPost `json:"posts,omitempty"`
	Users []*RelUser `json:"users,omitempty"`
}

func (t *RelTag) TableName() string {
	return "tags"
}

// Country and City for through relationship testing
type RelCountry struct {
	BaseModel
	Name string `db:"name" json:"name"`
	
	// Relationships
	Users []*RelUser `json:"users,omitempty"`
}

func (c *RelCountry) TableName() string {
	return "countries"
}

type RelCity struct {
	BaseModel
	CountryID uint   `db:"country_id" json:"country_id"`
	Name      string `db:"name" json:"name"`
	
	// Relationships
	Country *RelCountry `json:"country,omitempty"`
	Users   []*RelUser  `json:"users,omitempty"`
}

func (c *RelCity) TableName() string {
	return "cities"
}

// TestBelongsToRelationship tests the belongs to relationship
func TestBelongsToRelationship(t *testing.T) {
	profile := &RelProfile{UserID: 1}
	user := &RelUser{}
	
	relationship := NewBelongsTo(profile, user, "user_id", "id")
	
	if relationship.GetForeignKey() != "user_id" {
		t.Errorf("Expected foreign key 'user_id', got %s", relationship.GetForeignKey())
	}
	
	if relationship.GetLocalKey() != "id" {
		t.Errorf("Expected local key 'id', got %s", relationship.GetLocalKey())
	}
	
	if relationship.GetParent() != profile {
		t.Error("Expected parent to be profile")
	}
	
	if relationship.GetRelated() != user {
		t.Error("Expected related to be user")
	}
}

// TestHasOneRelationship tests the has one relationship
func TestHasOneRelationship(t *testing.T) {
	user := &RelUser{}
	profile := &RelProfile{}
	
	relationship := NewHasOne(user, profile, "user_id", "id")
	
	if relationship.GetForeignKey() != "user_id" {
		t.Errorf("Expected foreign key 'user_id', got %s", relationship.GetForeignKey())
	}
	
	if relationship.GetLocalKey() != "id" {
		t.Errorf("Expected local key 'id', got %s", relationship.GetLocalKey())
	}
	
	if relationship.GetParent() != user {
		t.Error("Expected parent to be user")
	}
	
	if relationship.GetRelated() != profile {
		t.Error("Expected related to be profile")
	}
}

// TestHasManyRelationship tests the has many relationship
func TestHasManyRelationship(t *testing.T) {
	user := &RelUser{}
	post := &RelPost{}
	
	relationship := NewHasMany(user, post, "user_id", "id")
	
	if relationship.GetForeignKey() != "user_id" {
		t.Errorf("Expected foreign key 'user_id', got %s", relationship.GetForeignKey())
	}
	
	if relationship.GetLocalKey() != "id" {
		t.Errorf("Expected local key 'id', got %s", relationship.GetLocalKey())
	}
	
	if relationship.GetParent() != user {
		t.Error("Expected parent to be user")
	}
	
	if relationship.GetRelated() != post {
		t.Error("Expected related to be post")
	}
}

// TestBelongsToManyRelationship tests the belongs to many relationship
func TestBelongsToManyRelationship(t *testing.T) {
	user := &RelUser{}
	tag := &RelTag{}
	
	relationship := NewBelongsToMany(user, tag, "user_tag", "user_id", "tag_id", "id", "id")
	
	if relationship.GetForeignKey() != "user_id" {
		t.Errorf("Expected foreign key 'user_id', got %s", relationship.GetForeignKey())
	}
	
	if relationship.GetLocalKey() != "id" {
		t.Errorf("Expected local key 'id', got %s", relationship.GetLocalKey())
	}
	
	if relationship.pivotTable != "user_tag" {
		t.Errorf("Expected pivot table 'user_tag', got %s", relationship.pivotTable)
	}
	
	if relationship.foreignPivotKey != "user_id" {
		t.Errorf("Expected foreign pivot key 'user_id', got %s", relationship.foreignPivotKey)
	}
	
	if relationship.relatedPivotKey != "tag_id" {
		t.Errorf("Expected related pivot key 'tag_id', got %s", relationship.relatedPivotKey)
	}
}

// TestMorphToRelationship tests the morph to relationship
func TestMorphToRelationship(t *testing.T) {
	comment := &RelComment{}
	
	relationship := NewMorphTo(comment, "commentable_type", "commentable_id")
	
	if relationship.morphType != "commentable_type" {
		t.Errorf("Expected morph type 'commentable_type', got %s", relationship.morphType)
	}
	
	if relationship.morphId != "commentable_id" {
		t.Errorf("Expected morph id 'commentable_id', got %s", relationship.morphId)
	}
	
	if relationship.GetParent() != comment {
		t.Error("Expected parent to be comment")
	}
}

// TestMorphOneRelationship tests the morph one relationship
func TestMorphOneRelationship(t *testing.T) {
	post := &RelPost{}
	comment := &RelComment{}
	
	relationship := NewMorphOne(post, comment, "commentable_type", "commentable_id", "id")
	
	if relationship.morphType != "commentable_type" {
		t.Errorf("Expected morph type 'commentable_type', got %s", relationship.morphType)
	}
	
	if relationship.morphId != "commentable_id" {
		t.Errorf("Expected morph id 'commentable_id', got %s", relationship.morphId)
	}
	
	if relationship.morphClass != "relpost" {
		t.Errorf("Expected morph class 'relpost', got %s", relationship.morphClass)
	}
	
	if relationship.GetParent() != post {
		t.Error("Expected parent to be post")
	}
	
	if relationship.GetRelated() != comment {
		t.Error("Expected related to be comment")
	}
}

// TestMorphManyRelationship tests the morph many relationship
func TestMorphManyRelationship(t *testing.T) {
	post := &RelPost{}
	comment := &RelComment{}
	
	relationship := NewMorphMany(post, comment, "commentable_type", "commentable_id", "id")
	
	if relationship.morphType != "commentable_type" {
		t.Errorf("Expected morph type 'commentable_type', got %s", relationship.morphType)
	}
	
	if relationship.morphId != "commentable_id" {
		t.Errorf("Expected morph id 'commentable_id', got %s", relationship.morphId)
	}
	
	if relationship.morphClass != "relpost" {
		t.Errorf("Expected morph class 'relpost', got %s", relationship.morphClass)
	}
	
	if relationship.GetParent() != post {
		t.Error("Expected parent to be post")
	}
	
	if relationship.GetRelated() != comment {
		t.Error("Expected related to be comment")
	}
}

// TestHasOneThroughRelationship tests the has one through relationship
func TestHasOneThroughRelationship(t *testing.T) {
	country := &RelCountry{}
	user := &RelUser{}
	city := &RelCity{}
	
	relationship := NewHasOneThrough(country, user, city, "country_id", "city_id", "id", "id")
	
	if relationship.firstKey != "country_id" {
		t.Errorf("Expected first key 'country_id', got %s", relationship.firstKey)
	}
	
	if relationship.secondKey != "city_id" {
		t.Errorf("Expected second key 'city_id', got %s", relationship.secondKey)
	}
	
	if relationship.localKey != "id" {
		t.Errorf("Expected local key 'id', got %s", relationship.localKey)
	}
	
	if relationship.secondLocalKey != "id" {
		t.Errorf("Expected second local key 'id', got %s", relationship.secondLocalKey)
	}
	
	if relationship.GetParent() != country {
		t.Error("Expected parent to be country")
	}
	
	if relationship.GetRelated() != user {
		t.Error("Expected related to be user")
	}
	
	if relationship.throughModel != city {
		t.Error("Expected through model to be city")
	}
}

// TestHasManyThroughRelationship tests the has many through relationship
func TestHasManyThroughRelationship(t *testing.T) {
	country := &RelCountry{}
	user := &RelUser{}
	city := &RelCity{}
	
	relationship := NewHasManyThrough(country, user, city, "country_id", "city_id", "id", "id")
	
	if relationship.firstKey != "country_id" {
		t.Errorf("Expected first key 'country_id', got %s", relationship.firstKey)
	}
	
	if relationship.secondKey != "city_id" {
		t.Errorf("Expected second key 'city_id', got %s", relationship.secondKey)
	}
	
	if relationship.localKey != "id" {
		t.Errorf("Expected local key 'id', got %s", relationship.localKey)
	}
	
	if relationship.secondLocalKey != "id" {
		t.Errorf("Expected second local key 'id', got %s", relationship.secondLocalKey)
	}
	
	if relationship.GetParent() != country {
		t.Error("Expected parent to be country")
	}
	
	if relationship.GetRelated() != user {
		t.Error("Expected related to be user")
	}
	
	if relationship.throughModel != city {
		t.Error("Expected through model to be city")
	}
}

// TestRelationshipConstraints tests relationship constraints
func TestRelationshipConstraints(t *testing.T) {
	user := &RelUser{}
	post := &RelPost{}
	
	relationship := NewHasMany(user, post, "user_id", "id")
	
	// Test adding constraints
	relationship.AddConstraint("status", "=", "published")
	relationship.OrderBy("created_at", "desc")
	relationship.Limit(10)
	
	if len(relationship.constraints) != 1 {
		t.Errorf("Expected 1 constraint, got %d", len(relationship.constraints))
	}
	
	if relationship.constraints[0].Column != "status" {
		t.Errorf("Expected constraint column 'status', got %s", relationship.constraints[0].Column)
	}
	
	if relationship.constraints[0].Operator != "=" {
		t.Errorf("Expected constraint operator '=', got %s", relationship.constraints[0].Operator)
	}
	
	if relationship.constraints[0].Value != "published" {
		t.Errorf("Expected constraint value 'published', got %v", relationship.constraints[0].Value)
	}
	
	if len(relationship.orderBy) != 1 {
		t.Errorf("Expected 1 order by clause, got %d", len(relationship.orderBy))
	}
	
	if relationship.orderBy[0].Column != "created_at" {
		t.Errorf("Expected order by column 'created_at', got %s", relationship.orderBy[0].Column)
	}
	
	if relationship.orderBy[0].Direction != "desc" {
		t.Errorf("Expected order by direction 'desc', got %s", relationship.orderBy[0].Direction)
	}
	
	if relationship.limitValue == nil || *relationship.limitValue != 10 {
		t.Errorf("Expected limit value 10, got %v", relationship.limitValue)
	}
}

// TestEagerLoadingEngine tests the eager loading engine
func TestEagerLoadingEngine(t *testing.T) {
	engine := NewEagerLoadingEngine()
	
	if engine == nil {
		t.Fatal("Expected eager loading engine to be created")
	}
	
	if engine.relations == nil {
		t.Fatal("Expected relations map to be initialized")
	}
	
	// Test adding relations
	engine.AddRelation("posts", nil)
	engine.AddRelation("posts.comments", nil)
	
	if len(engine.relations) != 1 {
		t.Errorf("Expected 1 top-level relation, got %d", len(engine.relations))
	}
	
	postsRelation, exists := engine.relations["posts"]
	if !exists {
		t.Fatal("Expected 'posts' relation to exist")
	}
	
	if postsRelation.Name != "posts" {
		t.Errorf("Expected relation name 'posts', got %s", postsRelation.Name)
	}
	
	if len(postsRelation.Nested) != 1 {
		t.Errorf("Expected 1 nested relation, got %d", len(postsRelation.Nested))
	}
	
	commentsRelation, exists := postsRelation.Nested["comments"]
	if !exists {
		t.Fatal("Expected nested 'comments' relation to exist")
	}
	
	if commentsRelation.Name != "comments" {
		t.Errorf("Expected nested relation name 'comments', got %s", commentsRelation.Name)
	}
}

// TestRelationshipRegistry tests the relationship registry
func TestRelationshipRegistry(t *testing.T) {
	registry := NewRelationshipRegistry()
	
	if registry == nil {
		t.Fatal("Expected relationship registry to be created")
	}
	
	if registry.relationships == nil {
		t.Fatal("Expected relationships map to be initialized")
	}
	
	// Test registering relationships
	registry.RegisterRelationship("User", "posts", func() Relationship {
		return NewHasMany(&RelUser{}, &RelPost{}, "user_id", "id")
	})
	
	registry.RegisterRelationship("User", "profile", func() Relationship {
		return NewHasOne(&RelUser{}, &RelProfile{}, "user_id", "id")
	})
	
	// Test getting relationships
	postsFactory, exists := registry.GetRelationship("User", "posts")
	if !exists {
		t.Fatal("Expected 'posts' relationship to exist for User")
	}
	
	postsRelation := postsFactory()
	if postsRelation == nil {
		t.Fatal("Expected posts relationship to be created")
	}
	
	profileFactory, exists := registry.GetRelationship("User", "profile")
	if !exists {
		t.Fatal("Expected 'profile' relationship to exist for User")
	}
	
	profileRelation := profileFactory()
	if profileRelation == nil {
		t.Fatal("Expected profile relationship to be created")
	}
	
	// Test non-existent relationship
	_, exists = registry.GetRelationship("User", "nonexistent")
	if exists {
		t.Error("Expected 'nonexistent' relationship to not exist")
	}
	
	// Test non-existent model
	_, exists = registry.GetRelationship("NonExistent", "posts")
	if exists {
		t.Error("Expected relationships for 'NonExistent' model to not exist")
	}
}

// TestLazyLoader tests the lazy loader
func TestLazyLoader(t *testing.T) {
	user := &RelUser{BaseModel: BaseModel{ID: 1}}
	loader := NewLazyLoader(user)
	
	if loader == nil {
		t.Fatal("Expected lazy loader to be created")
	}
	
	if loader.model != user {
		t.Error("Expected lazy loader model to be user")
	}
	
	// Test LoadMissing - this would normally check if relationships are already loaded
	// For testing purposes, we'll just ensure it doesn't panic
	err := loader.LoadMissing("posts", "profile")
	if err != nil {
		// This is expected to fail in our test environment without a database
		// The important thing is that it doesn't panic
		t.Logf("LoadMissing failed as expected (no database): %v", err)
	}
}

// TestDefaultForeignKeys tests default foreign key generation
func TestDefaultForeignKeys(t *testing.T) {
	user := &RelUser{}
	post := &RelPost{}
	
	userFK := getDefaultForeignKey(user)
	if userFK != "reluser_id" {
		t.Errorf("Expected default foreign key 'reluser_id', got %s", userFK)
	}
	
	postFK := getDefaultForeignKey(post)
	if postFK != "relpost_id" {
		t.Errorf("Expected default foreign key 'relpost_id', got %s", postFK)
	}
}

// TestTableNames tests table name generation
func TestTableNames(t *testing.T) {
	user := &RelUser{}
	post := &RelPost{}
	profile := &RelProfile{}
	
	userTable := getTableName(user)
	if userTable != "users" {
		t.Errorf("Expected table name 'users', got %s", userTable)
	}
	
	postTable := getTableName(post)
	if postTable != "posts" {
		t.Errorf("Expected table name 'posts', got %s", postTable)
	}
	
	profileTable := getTableName(profile)
	if profileTable != "profiles" {
		t.Errorf("Expected table name 'profiles', got %s", profileTable)
	}
}

// TestPivotTableNames tests pivot table name generation
func TestPivotTableNames(t *testing.T) {
	user := &RelUser{}
	tag := &RelTag{}
	
	pivotTable := getDefaultPivotTableName(user, tag)
	// Should be alphabetical order
	if pivotTable != "reltag_reluser" {
		t.Errorf("Expected pivot table name 'reltag_reluser', got %s", pivotTable)
	}
	
	// Test reverse order
	pivotTable2 := getDefaultPivotTableName(tag, user)
	if pivotTable2 != "reltag_reluser" {
		t.Errorf("Expected pivot table name 'reltag_reluser', got %s", pivotTable2)
	}
}

// TestModelName tests model name extraction
func TestModelName(t *testing.T) {
	user := &RelUser{}
	post := &RelPost{}
	
	userName := getModelName(user)
	if userName != "reluser" {
		t.Errorf("Expected model name 'reluser', got %s", userName)
	}
	
	postName := getModelName(post)
	if postName != "relpost" {
		t.Errorf("Expected model name 'relpost', got %s", postName)
	}
}

// TestKeyValue tests getting key values from models
func TestKeyValue(t *testing.T) {
	user := &RelUser{
		BaseModel: BaseModel{ID: 42},
		Name:      "John Doe",
		Email:     "john@example.com",
	}
	
	// Test with "ID" (struct field name)
	id := getKeyValue(user, "ID") 
	if id != uint(42) {
		t.Errorf("Expected ID 42, got %v", id)
	}
	
	// Test with "id" (lowercase json tag)
	idLower := getKeyValue(user, "id")
	if idLower != uint(42) {
		t.Errorf("Expected ID 42 via 'id' tag, got %v", idLower)
	}
	
	name := getKeyValue(user, "name")
	if name != "John Doe" {
		t.Errorf("Expected name 'John Doe', got %v", name)
	}
	
	email := getKeyValue(user, "email")
	if email != "john@example.com" {
		t.Errorf("Expected email 'john@example.com', got %v", email)
	}
	
	// Test non-existent field
	nonExistent := getKeyValue(user, "nonexistent")
	if nonExistent != nil {
		t.Errorf("Expected nil for non-existent field, got %v", nonExistent)
	}
}

// TestRelationshipModel tests the relationship model methods
func TestRelationshipModel(t *testing.T) {
	rm := &RelationshipModel{}
	user := &RelUser{}
	post := &RelPost{}
	profile := &RelProfile{}
	tag := &RelTag{}
	comment := &RelComment{}
	city := &RelCity{}
	country := &RelCountry{}
	
	// Test BelongsTo
	belongsTo := rm.BelongsTo(profile, user, "user_id", "id")
	if belongsTo == nil {
		t.Fatal("Expected BelongsTo relationship to be created")
	}
	
	// Test HasOne
	hasOne := rm.HasOne(user, profile, "user_id", "id")
	if hasOne == nil {
		t.Fatal("Expected HasOne relationship to be created")
	}
	
	// Test HasMany
	hasMany := rm.HasMany(user, post, "user_id", "id")
	if hasMany == nil {
		t.Fatal("Expected HasMany relationship to be created")
	}
	
	// Test BelongsToMany
	belongsToMany := rm.BelongsToMany(user, tag, "user_tag", "user_id", "tag_id", "id", "id")
	if belongsToMany == nil {
		t.Fatal("Expected BelongsToMany relationship to be created")
	}
	
	// Test MorphTo
	morphTo := rm.MorphTo(comment, "commentable_type", "commentable_id")
	if morphTo == nil {
		t.Fatal("Expected MorphTo relationship to be created")
	}
	
	// Test MorphOne
	morphOne := rm.MorphOne(post, comment, "commentable_type", "commentable_id", "id")
	if morphOne == nil {
		t.Fatal("Expected MorphOne relationship to be created")
	}
	
	// Test MorphMany
	morphMany := rm.MorphMany(post, comment, "commentable_type", "commentable_id", "id")
	if morphMany == nil {
		t.Fatal("Expected MorphMany relationship to be created")
	}
	
	// Test HasOneThrough
	hasOneThrough := rm.HasOneThrough(country, user, city, "country_id", "city_id", "id", "id")
	if hasOneThrough == nil {
		t.Fatal("Expected HasOneThrough relationship to be created")
	}
	
	// Test HasManyThrough
	hasManyThrough := rm.HasManyThrough(country, user, city, "country_id", "city_id", "id", "id")
	if hasManyThrough == nil {
		t.Fatal("Expected HasManyThrough relationship to be created")
	}
}

// Benchmark tests for performance

func BenchmarkNewBelongsTo(b *testing.B) {
	profile := &RelProfile{}
	user := &RelUser{}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewBelongsTo(profile, user, "user_id", "id")
	}
}

func BenchmarkNewHasMany(b *testing.B) {
	user := &RelUser{}
	post := &RelPost{}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewHasMany(user, post, "user_id", "id")
	}
}

func BenchmarkEagerLoadingEngine(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine := NewEagerLoadingEngine()
		engine.AddRelation("posts", nil)
		engine.AddRelation("posts.comments", nil)
		engine.AddRelation("profile", nil)
	}
}

func BenchmarkRelationshipRegistry(b *testing.B) {
	registry := NewRelationshipRegistry()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry.RegisterRelationship("User", "posts", func() Relationship {
			return NewHasMany(&RelUser{}, &RelPost{}, "user_id", "id")
		})
		registry.GetRelationship("User", "posts")
	}
}