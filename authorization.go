package onyx

import (
	"fmt"
	"reflect"
)

type Gate interface {
	Define(ability string, callback GateCallback)
	Policy(model interface{}, policy Policy)
	Check(ability string, args ...interface{}) bool
	Allows(ability string, args ...interface{}) bool
	Denies(ability string, args ...interface{}) bool
	ForUser(user User) Gate
	Authorize(ability string, args ...interface{}) error
	Any(abilities []string, args ...interface{}) bool
	None(abilities []string, args ...interface{}) bool
	Before(callback BeforeCallback)
	After(callback AfterCallback)
}

type GateCallback func(user User, args ...interface{}) bool
type BeforeCallback func(user User, ability string, args ...interface{}) *bool
type AfterCallback func(user User, ability string, result bool, args ...interface{}) *bool

type Policy interface {
	Before(user User, ability string) *bool
}

type AuthorizationGate struct {
	container     *Container
	userResolver  func() User
	abilities     map[string]GateCallback
	policies      map[string]Policy
	beforeCallbacks []BeforeCallback
	afterCallbacks  []AfterCallback
}

func NewAuthorizationGate(container *Container, userResolver func() User) *AuthorizationGate {
	return &AuthorizationGate{
		container:       container,
		userResolver:    userResolver,
		abilities:       make(map[string]GateCallback),
		policies:        make(map[string]Policy),
		beforeCallbacks: make([]BeforeCallback, 0),
		afterCallbacks:  make([]AfterCallback, 0),
	}
}

func (ag *AuthorizationGate) Define(ability string, callback GateCallback) {
	ag.abilities[ability] = callback
}

func (ag *AuthorizationGate) Policy(model interface{}, policy Policy) {
	modelType := reflect.TypeOf(model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}
	ag.policies[modelType.Name()] = policy
}

func (ag *AuthorizationGate) Check(ability string, args ...interface{}) bool {
	return ag.inspect(ability, args...)
}

func (ag *AuthorizationGate) Allows(ability string, args ...interface{}) bool {
	return ag.Check(ability, args...)
}

func (ag *AuthorizationGate) Denies(ability string, args ...interface{}) bool {
	return !ag.Check(ability, args...)
}

func (ag *AuthorizationGate) ForUser(user User) Gate {
	return NewAuthorizationGate(ag.container, func() User { return user })
}

func (ag *AuthorizationGate) Authorize(ability string, args ...interface{}) error {
	if !ag.Check(ability, args...) {
		return fmt.Errorf("this action is unauthorized")
	}
	return nil
}

func (ag *AuthorizationGate) Any(abilities []string, args ...interface{}) bool {
	for _, ability := range abilities {
		if ag.Check(ability, args...) {
			return true
		}
	}
	return false
}

func (ag *AuthorizationGate) None(abilities []string, args ...interface{}) bool {
	return !ag.Any(abilities, args...)
}

func (ag *AuthorizationGate) Before(callback BeforeCallback) {
	ag.beforeCallbacks = append(ag.beforeCallbacks, callback)
}

func (ag *AuthorizationGate) After(callback AfterCallback) {
	ag.afterCallbacks = append(ag.afterCallbacks, callback)
}

func (ag *AuthorizationGate) inspect(ability string, args ...interface{}) bool {
	user := ag.resolveUser()
	if user == nil {
		return false
	}

	result := ag.callBeforeCallbacks(user, ability, args...)
	if result != nil {
		return *result
	}

	result = ag.callAuthCallback(user, ability, args...)
	if result == nil {
		defaultResult := false
		result = &defaultResult
	}

	return ag.callAfterCallbacks(user, ability, *result, args...)
}

func (ag *AuthorizationGate) callBeforeCallbacks(user User, ability string, args ...interface{}) *bool {
	for _, callback := range ag.beforeCallbacks {
		if result := callback(user, ability, args...); result != nil {
			return result
		}
	}
	return nil
}

func (ag *AuthorizationGate) callAfterCallbacks(user User, ability string, result bool, args ...interface{}) bool {
	for _, callback := range ag.afterCallbacks {
		if afterResult := callback(user, ability, result, args...); afterResult != nil {
			return *afterResult
		}
	}
	return result
}

func (ag *AuthorizationGate) callAuthCallback(user User, ability string, args ...interface{}) *bool {
	callback := ag.resolveAuthCallback(user, ability, args...)
	if callback != nil {
		result := callback(user, args...)
		return &result
	}
	return nil
}

func (ag *AuthorizationGate) resolveAuthCallback(user User, ability string, args ...interface{}) GateCallback {
	if len(args) > 0 && ag.firstArgumentIsModel(args[0]) {
		return ag.resolvePolicyCallback(user, ability, args[0])
	}

	if callback, exists := ag.abilities[ability]; exists {
		return callback
	}

	return nil
}

func (ag *AuthorizationGate) firstArgumentIsModel(argument interface{}) bool {
	if argument == nil {
		return false
	}
	
	argType := reflect.TypeOf(argument)
	if argType.Kind() == reflect.Ptr {
		argType = argType.Elem()
	}
	
	return argType.Kind() == reflect.Struct
}

func (ag *AuthorizationGate) resolvePolicyCallback(user User, ability string, model interface{}) GateCallback {
	policy := ag.resolvePolicy(model)
	if policy == nil {
		return nil
	}

	beforeResult := policy.Before(user, ability)
	if beforeResult != nil {
		return func(user User, args ...interface{}) bool {
			return *beforeResult
		}
	}

	policyValue := reflect.ValueOf(policy)
	methodName := ag.formatAbilityToMethod(ability)
	method := policyValue.MethodByName(methodName)
	
	if !method.IsValid() {
		return nil
	}

	return func(user User, args ...interface{}) bool {
		methodArgs := []reflect.Value{reflect.ValueOf(user)}
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

func (ag *AuthorizationGate) resolvePolicy(model interface{}) Policy {
	modelType := reflect.TypeOf(model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}
	
	if policy, exists := ag.policies[modelType.Name()]; exists {
		return policy
	}
	
	return nil
}

func (ag *AuthorizationGate) formatAbilityToMethod(ability string) string {
	if len(ability) == 0 {
		return ""
	}
	return fmt.Sprintf("%s%s", string(ability[0]-32), ability[1:])
}

func (ag *AuthorizationGate) resolveUser() User {
	if ag.userResolver != nil {
		return ag.userResolver()
	}
	return nil
}

type Authorizable interface {
	Can(ability string, args ...interface{}) bool
	Cannot(ability string, args ...interface{}) bool
	CanAny(abilities []string, args ...interface{}) bool
	CanNone(abilities []string, args ...interface{}) bool
}

func CanMiddleware(ability string, args ...interface{}) MiddlewareFunc {
	return func(c *Context) error {
		gate := c.Gate()
		if gate == nil || !gate.Check(ability, args...) {
			return c.JSON(403, map[string]string{
				"error": "Forbidden - Insufficient permissions",
			})
		}
		return c.Next()
	}
}

func (c *Context) Gate() Gate {
	gate, _ := c.app.Container().Make("gate")
	if g, ok := gate.(Gate); ok {
		return g
	}
	return nil
}

func (c *Context) Can(ability string, args ...interface{}) bool {
	gate := c.Gate()
	if gate == nil {
		return false
	}
	return gate.Check(ability, args...)
}

func (c *Context) Cannot(ability string, args ...interface{}) bool {
	return !c.Can(ability, args...)
}

func (c *Context) Authorize(ability string, args ...interface{}) error {
	gate := c.Gate()
	if gate == nil {
		return fmt.Errorf("gate not configured")
	}
	return gate.Authorize(ability, args...)
}

type BasePolicy struct{}

func (bp *BasePolicy) Before(user User, ability string) *bool {
	return nil
}

type UserPolicy struct {
	BasePolicy
}

func (up *UserPolicy) View(user User, model interface{}) bool {
	return true
}

func (up *UserPolicy) Create(user User) bool {
	return user != nil
}

func (up *UserPolicy) Update(user User, model interface{}) bool {
	if userModel, ok := model.(User); ok {
		return user.GetID() == userModel.GetID()
	}
	return false
}

func (up *UserPolicy) Delete(user User, model interface{}) bool {
	if userModel, ok := model.(User); ok {
		return user.GetID() == userModel.GetID()
	}
	return false
}

func RegisterGateCallbacks(gate Gate) {
	gate.Define("view-dashboard", func(user User, args ...interface{}) bool {
		return user != nil
	})

	gate.Define("admin-access", func(user User, args ...interface{}) bool {
		if genericUser, ok := user.(*GenericUser); ok {
			return genericUser.Email == "admin@example.com"
		}
		return false
	})

	gate.Before(func(user User, ability string, args ...interface{}) *bool {
		if genericUser, ok := user.(*GenericUser); ok {
			if genericUser.Email == "superadmin@example.com" {
				result := true
				return &result
			}
		}
		return nil
	})
}