package auth

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

// Gate manages authorization abilities and policies
type Gate interface {
	Define(ability string, callback GateCallback)
	Policy(model interface{}, policy Policy)
	Check(ctx context.Context, ability string, args ...interface{}) bool
	Allows(ctx context.Context, ability string, args ...interface{}) bool
	Denies(ctx context.Context, ability string, args ...interface{}) bool
	ForUser(user User) Gate
	Authorize(ctx context.Context, ability string, args ...interface{}) error
	Any(ctx context.Context, abilities []string, args ...interface{}) bool
	None(ctx context.Context, abilities []string, args ...interface{}) bool
	Before(callback BeforeCallback)
	After(callback AfterCallback)
	GetPolicies() map[string]Policy
	GetAbilities() map[string]GateCallback
	HasAbility(ability string) bool
	HasPolicy(model interface{}) bool
}

// GateCallback defines authorization logic for an ability
type GateCallback func(ctx context.Context, user User, args ...interface{}) bool

// BeforeCallback runs before authorization checks
type BeforeCallback func(ctx context.Context, user User, ability string, args ...interface{}) *bool

// AfterCallback runs after authorization checks
type AfterCallback func(ctx context.Context, user User, ability string, result bool, args ...interface{}) *bool

// Policy defines authorization logic for a model
type Policy interface {
	Before(ctx context.Context, user User, ability string) *bool
}

// AuthorizationManager manages gates and policies
type AuthorizationManager interface {
	Gate() Gate
	SetGate(gate Gate)
	RegisterPolicy(model interface{}, policy Policy)
	RegisterAbility(ability string, callback GateCallback)
	Can(ctx context.Context, user User, ability string, args ...interface{}) bool
	Cannot(ctx context.Context, user User, ability string, args ...interface{}) bool
	Authorize(ctx context.Context, user User, ability string, args ...interface{}) error
	ForUser(user User) Gate
}

// Authorizable provides authorization methods for users
type Authorizable interface {
	Can(ctx context.Context, ability string, args ...interface{}) bool
	Cannot(ctx context.Context, ability string, args ...interface{}) bool
	CanAny(ctx context.Context, abilities []string, args ...interface{}) bool
	CanNone(ctx context.Context, abilities []string, args ...interface{}) bool
}

// PolicyManager manages model policies
type PolicyManager interface {
	Register(model interface{}, policy Policy)
	Get(model interface{}) Policy
	Has(model interface{}) bool
	Call(ctx context.Context, policy Policy, method string, user User, args ...interface{}) (bool, error)
	Forget(model interface{})
	GetPolicies() map[string]Policy
}

// AbilityManager manages gate abilities
type AbilityManager interface {
	Register(ability string, callback GateCallback)
	Get(ability string) GateCallback
	Has(ability string) bool
	Call(ctx context.Context, ability string, user User, args ...interface{}) bool
	Forget(ability string)
	GetAbilities() map[string]GateCallback
}

// DefaultGate implements the Gate interface
type DefaultGate struct {
	container       Container
	userResolver    UserResolver
	abilities       map[string]GateCallback
	policies        map[string]Policy
	beforeCallbacks []BeforeCallback
	afterCallbacks  []AfterCallback
}

// Container provides dependency injection
type Container interface {
	Make(abstract string) (interface{}, error)
	Bind(abstract string, concrete interface{})
	Singleton(abstract string, concrete interface{})
	Has(abstract string) bool
}

// UserResolver resolves the current user
type UserResolver func(ctx context.Context) User

// NewDefaultGate creates a new gate instance
func NewDefaultGate(container Container, userResolver UserResolver) *DefaultGate {
	return &DefaultGate{
		container:       container,
		userResolver:    userResolver,
		abilities:       make(map[string]GateCallback),
		policies:        make(map[string]Policy),
		beforeCallbacks: make([]BeforeCallback, 0),
		afterCallbacks:  make([]AfterCallback, 0),
	}
}

// Define registers an ability with a callback
func (g *DefaultGate) Define(ability string, callback GateCallback) {
	g.abilities[ability] = callback
}

// Policy registers a policy for a model
func (g *DefaultGate) Policy(model interface{}, policy Policy) {
	modelType := reflect.TypeOf(model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}
	g.policies[modelType.Name()] = policy
}

// Check determines if the user can perform the given ability
func (g *DefaultGate) Check(ctx context.Context, ability string, args ...interface{}) bool {
	return g.inspect(ctx, ability, args...)
}

// Allows is an alias for Check
func (g *DefaultGate) Allows(ctx context.Context, ability string, args ...interface{}) bool {
	return g.Check(ctx, ability, args...)
}

// Denies returns the opposite of Check
func (g *DefaultGate) Denies(ctx context.Context, ability string, args ...interface{}) bool {
	return !g.Check(ctx, ability, args...)
}

// ForUser creates a gate for a specific user
func (g *DefaultGate) ForUser(user User) Gate {
	return NewDefaultGate(g.container, func(ctx context.Context) User { return user })
}

// Authorize checks authorization and returns an error if denied
func (g *DefaultGate) Authorize(ctx context.Context, ability string, args ...interface{}) error {
	if !g.Check(ctx, ability, args...) {
		return fmt.Errorf("this action is unauthorized")
	}
	return nil
}

// Any checks if the user can perform any of the given abilities
func (g *DefaultGate) Any(ctx context.Context, abilities []string, args ...interface{}) bool {
	for _, ability := range abilities {
		if g.Check(ctx, ability, args...) {
			return true
		}
	}
	return false
}

// None checks if the user cannot perform any of the given abilities
func (g *DefaultGate) None(ctx context.Context, abilities []string, args ...interface{}) bool {
	return !g.Any(ctx, abilities, args...)
}

// Before adds a before callback
func (g *DefaultGate) Before(callback BeforeCallback) {
	g.beforeCallbacks = append(g.beforeCallbacks, callback)
}

// After adds an after callback
func (g *DefaultGate) After(callback AfterCallback) {
	g.afterCallbacks = append(g.afterCallbacks, callback)
}

// GetPolicies returns all registered policies
func (g *DefaultGate) GetPolicies() map[string]Policy {
	policies := make(map[string]Policy)
	for name, policy := range g.policies {
		policies[name] = policy
	}
	return policies
}

// GetAbilities returns all registered abilities
func (g *DefaultGate) GetAbilities() map[string]GateCallback {
	abilities := make(map[string]GateCallback)
	for name, callback := range g.abilities {
		abilities[name] = callback
	}
	return abilities
}

// HasAbility checks if an ability is registered
func (g *DefaultGate) HasAbility(ability string) bool {
	_, exists := g.abilities[ability]
	return exists
}

// HasPolicy checks if a policy is registered for the model
func (g *DefaultGate) HasPolicy(model interface{}) bool {
	modelType := reflect.TypeOf(model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}
	_, exists := g.policies[modelType.Name()]
	return exists
}

// Private methods

func (g *DefaultGate) inspect(ctx context.Context, ability string, args ...interface{}) bool {
	user := g.resolveUser(ctx)
	if user == nil {
		return false
	}

	result := g.callBeforeCallbacks(ctx, user, ability, args...)
	if result != nil {
		return *result
	}

	result = g.callAuthCallback(ctx, user, ability, args...)
	if result == nil {
		defaultResult := false
		result = &defaultResult
	}

	return g.callAfterCallbacks(ctx, user, ability, *result, args...)
}

func (g *DefaultGate) callBeforeCallbacks(ctx context.Context, user User, ability string, args ...interface{}) *bool {
	for _, callback := range g.beforeCallbacks {
		if result := callback(ctx, user, ability, args...); result != nil {
			return result
		}
	}
	return nil
}

func (g *DefaultGate) callAfterCallbacks(ctx context.Context, user User, ability string, result bool, args ...interface{}) bool {
	for _, callback := range g.afterCallbacks {
		if afterResult := callback(ctx, user, ability, result, args...); afterResult != nil {
			return *afterResult
		}
	}
	return result
}

func (g *DefaultGate) callAuthCallback(ctx context.Context, user User, ability string, args ...interface{}) *bool {
	callback := g.resolveAuthCallback(ctx, user, ability, args...)
	if callback != nil {
		result := callback(ctx, user, args...)
		return &result
	}
	return nil
}

func (g *DefaultGate) resolveAuthCallback(ctx context.Context, user User, ability string, args ...interface{}) GateCallback {
	if len(args) > 0 && g.firstArgumentIsModel(args[0]) {
		return g.resolvePolicyCallback(ctx, user, ability, args[0])
	}

	if callback, exists := g.abilities[ability]; exists {
		return callback
	}

	return nil
}

func (g *DefaultGate) firstArgumentIsModel(argument interface{}) bool {
	if argument == nil {
		return false
	}

	argType := reflect.TypeOf(argument)
	if argType.Kind() == reflect.Ptr {
		argType = argType.Elem()
	}

	return argType.Kind() == reflect.Struct
}

func (g *DefaultGate) resolvePolicyCallback(ctx context.Context, user User, ability string, model interface{}) GateCallback {
	policy := g.resolvePolicy(model)
	if policy == nil {
		return nil
	}

	beforeResult := policy.Before(ctx, user, ability)
	if beforeResult != nil {
		return func(ctx context.Context, user User, args ...interface{}) bool {
			return *beforeResult
		}
	}

	policyValue := reflect.ValueOf(policy)
	methodName := g.formatAbilityToMethod(ability)
	method := policyValue.MethodByName(methodName)

	if !method.IsValid() {
		return nil
	}

	return func(ctx context.Context, user User, args ...interface{}) bool {
		methodArgs := []reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(user),
		}
		for _, arg := range args {
			methodArgs = append(methodArgs, reflect.ValueOf(arg))
		}

		results := method.Call(methodArgs)
		if len(results) > 0 && results[0].Kind() == reflect.Bool {
			return results[0].Bool()
		}
		return false
	}
}

func (g *DefaultGate) resolvePolicy(model interface{}) Policy {
	modelType := reflect.TypeOf(model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	if policy, exists := g.policies[modelType.Name()]; exists {
		return policy
	}

	return nil
}

func (g *DefaultGate) formatAbilityToMethod(ability string) string {
	if len(ability) == 0 {
		return ""
	}
	
	// Convert kebab-case or snake_case to PascalCase
	parts := strings.FieldsFunc(ability, func(c rune) bool {
		return c == '-' || c == '_'
	})
	
	var result strings.Builder
	for _, part := range parts {
		if len(part) > 0 {
			result.WriteString(strings.ToUpper(string(part[0])))
			if len(part) > 1 {
				result.WriteString(strings.ToLower(part[1:]))
			}
		}
	}
	
	return result.String()
}

func (g *DefaultGate) resolveUser(ctx context.Context) User {
	if g.userResolver != nil {
		return g.userResolver(ctx)
	}
	return nil
}

// BasePolicy provides a default policy implementation
type BasePolicy struct{}

// Before runs before all policy methods
func (bp *BasePolicy) Before(ctx context.Context, user User, ability string) *bool {
	return nil
}

// DefaultPolicyManager implements PolicyManager
type DefaultPolicyManager struct {
	policies map[string]Policy
}

// NewDefaultPolicyManager creates a new policy manager
func NewDefaultPolicyManager() *DefaultPolicyManager {
	return &DefaultPolicyManager{
		policies: make(map[string]Policy),
	}
}

// Register registers a policy for a model
func (pm *DefaultPolicyManager) Register(model interface{}, policy Policy) {
	modelType := reflect.TypeOf(model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}
	pm.policies[modelType.Name()] = policy
}

// Get retrieves a policy for a model
func (pm *DefaultPolicyManager) Get(model interface{}) Policy {
	modelType := reflect.TypeOf(model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}
	return pm.policies[modelType.Name()]
}

// Has checks if a policy exists for a model
func (pm *DefaultPolicyManager) Has(model interface{}) bool {
	return pm.Get(model) != nil
}

// Call calls a policy method
func (pm *DefaultPolicyManager) Call(ctx context.Context, policy Policy, method string, user User, args ...interface{}) (bool, error) {
	policyValue := reflect.ValueOf(policy)
	methodValue := policyValue.MethodByName(method)

	if !methodValue.IsValid() {
		return false, fmt.Errorf("method %s not found on policy", method)
	}

	methodArgs := []reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(user),
	}
	for _, arg := range args {
		methodArgs = append(methodArgs, reflect.ValueOf(arg))
	}

	results := methodValue.Call(methodArgs)
	if len(results) > 0 && results[0].Kind() == reflect.Bool {
		return results[0].Bool(), nil
	}

	return false, fmt.Errorf("policy method %s did not return a boolean", method)
}

// Forget removes a policy for a model
func (pm *DefaultPolicyManager) Forget(model interface{}) {
	modelType := reflect.TypeOf(model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}
	delete(pm.policies, modelType.Name())
}

// GetPolicies returns all registered policies
func (pm *DefaultPolicyManager) GetPolicies() map[string]Policy {
	policies := make(map[string]Policy)
	for name, policy := range pm.policies {
		policies[name] = policy
	}
	return policies
}

// DefaultAbilityManager implements AbilityManager
type DefaultAbilityManager struct {
	abilities map[string]GateCallback
}

// NewDefaultAbilityManager creates a new ability manager
func NewDefaultAbilityManager() *DefaultAbilityManager {
	return &DefaultAbilityManager{
		abilities: make(map[string]GateCallback),
	}
}

// Register registers an ability with a callback
func (am *DefaultAbilityManager) Register(ability string, callback GateCallback) {
	am.abilities[ability] = callback
}

// Get retrieves an ability callback
func (am *DefaultAbilityManager) Get(ability string) GateCallback {
	return am.abilities[ability]
}

// Has checks if an ability exists
func (am *DefaultAbilityManager) Has(ability string) bool {
	_, exists := am.abilities[ability]
	return exists
}

// Call calls an ability callback
func (am *DefaultAbilityManager) Call(ctx context.Context, ability string, user User, args ...interface{}) bool {
	callback := am.Get(ability)
	if callback == nil {
		return false
	}
	return callback(ctx, user, args...)
}

// Forget removes an ability
func (am *DefaultAbilityManager) Forget(ability string) {
	delete(am.abilities, ability)
}

// GetAbilities returns all registered abilities
func (am *DefaultAbilityManager) GetAbilities() map[string]GateCallback {
	abilities := make(map[string]GateCallback)
	for name, callback := range am.abilities {
		abilities[name] = callback
	}
	return abilities
}

// AuthorizationError represents an authorization error
type AuthorizationError struct {
	Ability string
	User    User
	Message string
}

func (e *AuthorizationError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("authorization failed for ability: %s", e.Ability)
}

// NewAuthorizationError creates a new authorization error
func NewAuthorizationError(ability string, user User, message string) *AuthorizationError {
	return &AuthorizationError{
		Ability: ability,
		User:    user,
		Message: message,
	}
}

// PolicyResolutionError represents a policy resolution error
type PolicyResolutionError struct {
	Model   interface{}
	Policy  string
	Message string
}

func (e *PolicyResolutionError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("policy resolution failed for model: %T", e.Model)
}

// Helper functions

// ResolveAbilityMethod converts an ability name to a method name
func ResolveAbilityMethod(ability string) string {
	if len(ability) == 0 {
		return ""
	}

	// Convert kebab-case or snake_case to PascalCase
	parts := strings.FieldsFunc(ability, func(c rune) bool {
		return c == '-' || c == '_' || c == ' '
	})

	var result strings.Builder
	for _, part := range parts {
		if len(part) > 0 {
			result.WriteString(strings.ToUpper(string(part[0])))
			if len(part) > 1 {
				result.WriteString(strings.ToLower(part[1:]))
			}
		}
	}

	return result.String()
}

// GetModelName returns the name of a model type
func GetModelName(model interface{}) string {
	modelType := reflect.TypeOf(model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}
	return modelType.Name()
}

// IsModel checks if a value is a model (struct)
func IsModel(value interface{}) bool {
	if value == nil {
		return false
	}

	valueType := reflect.TypeOf(value)
	if valueType.Kind() == reflect.Ptr {
		valueType = valueType.Elem()
	}

	return valueType.Kind() == reflect.Struct
}