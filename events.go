package onyx

import (
	"fmt"
	"reflect"
	"sync"
)

type Event interface {
	GetName() string
}

type Listener interface {
	Handle(event Event) error
}

type ListenerFunc func(event Event) error

func (lf ListenerFunc) Handle(event Event) error {
	return lf(event)
}

type EventDispatcher interface {
	Listen(eventName string, listener Listener)
	ListenFunc(eventName string, listenerFunc ListenerFunc)
	Dispatch(event Event) error
	DispatchUntil(event Event, halt func(interface{}) bool) (interface{}, error)
	Push(eventName string, payload []interface{})
	Flush(eventName string)
	Subscribe(subscriber EventSubscriber)
	Forget(eventName string)
	ForgetPushed()
	HasListeners(eventName string) bool
}

type EventSubscriber interface {
	Subscribe(dispatcher EventDispatcher)
}

type BaseEvent struct {
	name string
}

func NewBaseEvent(name string) *BaseEvent {
	return &BaseEvent{name: name}
}

func (be *BaseEvent) GetName() string {
	return be.name
}

type DefaultEventDispatcher struct {
	listeners map[string][]Listener
	wildcards map[string][]Listener
	pushed    map[string][]interface{}
	mutex     sync.RWMutex
}

func NewEventDispatcher() *DefaultEventDispatcher {
	return &DefaultEventDispatcher{
		listeners: make(map[string][]Listener),
		wildcards: make(map[string][]Listener),
		pushed:    make(map[string][]interface{}),
	}
}

func (ed *DefaultEventDispatcher) Listen(eventName string, listener Listener) {
	ed.mutex.Lock()
	defer ed.mutex.Unlock()

	if ed.isWildcard(eventName) {
		ed.wildcards[eventName] = append(ed.wildcards[eventName], listener)
	} else {
		ed.listeners[eventName] = append(ed.listeners[eventName], listener)
	}
}

func (ed *DefaultEventDispatcher) ListenFunc(eventName string, listenerFunc ListenerFunc) {
	ed.Listen(eventName, listenerFunc)
}

func (ed *DefaultEventDispatcher) Dispatch(event Event) error {
	eventName := event.GetName()
	
	listeners := ed.getListeners(eventName)
	
	for _, listener := range listeners {
		if err := listener.Handle(event); err != nil {
			return err
		}
	}
	
	return nil
}

func (ed *DefaultEventDispatcher) DispatchUntil(event Event, halt func(interface{}) bool) (interface{}, error) {
	eventName := event.GetName()
	
	listeners := ed.getListeners(eventName)
	
	for _, listener := range listeners {
		result := listener.Handle(event)
		if halt(result) {
			return result, nil
		}
	}
	
	return nil, nil
}

func (ed *DefaultEventDispatcher) Push(eventName string, payload []interface{}) {
	ed.mutex.Lock()
	defer ed.mutex.Unlock()
	
	ed.pushed[eventName] = append(ed.pushed[eventName], payload...)
}

func (ed *DefaultEventDispatcher) Flush(eventName string) {
	ed.mutex.Lock()
	payloads := ed.pushed[eventName]
	delete(ed.pushed, eventName)
	ed.mutex.Unlock()
	
	for _, payload := range payloads {
		if event, ok := payload.(Event); ok {
			ed.Dispatch(event)
		}
	}
}

func (ed *DefaultEventDispatcher) Subscribe(subscriber EventSubscriber) {
	subscriber.Subscribe(ed)
}

func (ed *DefaultEventDispatcher) Forget(eventName string) {
	ed.mutex.Lock()
	defer ed.mutex.Unlock()
	
	delete(ed.listeners, eventName)
	
	for pattern := range ed.wildcards {
		if ed.matchesWildcard(pattern, eventName) {
			delete(ed.wildcards, pattern)
		}
	}
}

func (ed *DefaultEventDispatcher) ForgetPushed() {
	ed.mutex.Lock()
	defer ed.mutex.Unlock()
	
	ed.pushed = make(map[string][]interface{})
}

func (ed *DefaultEventDispatcher) HasListeners(eventName string) bool {
	return len(ed.getListeners(eventName)) > 0
}

func (ed *DefaultEventDispatcher) getListeners(eventName string) []Listener {
	ed.mutex.RLock()
	defer ed.mutex.RUnlock()
	
	var allListeners []Listener
	
	if listeners, exists := ed.listeners[eventName]; exists {
		allListeners = append(allListeners, listeners...)
	}
	
	for pattern, wildcardListeners := range ed.wildcards {
		if ed.matchesWildcard(pattern, eventName) {
			allListeners = append(allListeners, wildcardListeners...)
		}
	}
	
	return allListeners
}

func (ed *DefaultEventDispatcher) isWildcard(eventName string) bool {
	return eventName[len(eventName)-1] == '*'
}

func (ed *DefaultEventDispatcher) matchesWildcard(pattern, eventName string) bool {
	if !ed.isWildcard(pattern) {
		return false
	}
	
	prefix := pattern[:len(pattern)-1]
	return len(eventName) >= len(prefix) && eventName[:len(prefix)] == prefix
}

type UserRegistered struct {
	*BaseEvent
	User interface{}
}

func NewUserRegistered(user interface{}) *UserRegistered {
	return &UserRegistered{
		BaseEvent: NewBaseEvent("user.registered"),
		User:      user,
	}
}

type UserLoggedIn struct {
	*BaseEvent
	User interface{}
}

func NewUserLoggedIn(user interface{}) *UserLoggedIn {
	return &UserLoggedIn{
		BaseEvent: NewBaseEvent("user.logged_in"),
		User:      user,
	}
}

type UserLoggedOut struct {
	*BaseEvent
	User interface{}
}

func NewUserLoggedOut(user interface{}) *UserLoggedOut {
	return &UserLoggedOut{
		BaseEvent: NewBaseEvent("user.logged_out"),
		User:      user,
	}
}

type OrderCreated struct {
	*BaseEvent
	Order interface{}
}

func NewOrderCreated(order interface{}) *OrderCreated {
	return &OrderCreated{
		BaseEvent: NewBaseEvent("order.created"),
		Order:     order,
	}
}

type SendWelcomeEmailListener struct{}

func (swel *SendWelcomeEmailListener) Handle(event Event) error {
	if userRegistered, ok := event.(*UserRegistered); ok {
		fmt.Printf("Sending welcome email to user: %+v\n", userRegistered.User)
		
		if globalApp != nil {
			emailJob := NewSendEmailJob("user@example.com", "Welcome!", "Welcome to our platform!")
			return DispatchJob(emailJob)
		}
	}
	return nil
}

type LogUserActivityListener struct{}

func (lual *LogUserActivityListener) Handle(event Event) error {
	switch e := event.(type) {
	case *UserLoggedIn:
		fmt.Printf("User logged in: %+v\n", e.User)
	case *UserLoggedOut:
		fmt.Printf("User logged out: %+v\n", e.User)
	}
	return nil
}

type SendOrderConfirmationListener struct{}

func (socl *SendOrderConfirmationListener) Handle(event Event) error {
	if orderCreated, ok := event.(*OrderCreated); ok {
		fmt.Printf("Sending order confirmation for: %+v\n", orderCreated.Order)
		
		emailJob := NewSendEmailJob("customer@example.com", "Order Confirmation", "Your order has been confirmed!")
		return DispatchJob(emailJob)
	}
	return nil
}

type EventServiceProvider struct {
	events map[string][]string
}

func NewEventServiceProvider() *EventServiceProvider {
	return &EventServiceProvider{
		events: make(map[string][]string),
	}
}

func (esp *EventServiceProvider) Boot(dispatcher EventDispatcher) {
	esp.registerEventListeners(dispatcher)
}

func (esp *EventServiceProvider) registerEventListeners(dispatcher EventDispatcher) {
	dispatcher.Listen("user.registered", &SendWelcomeEmailListener{})
	dispatcher.Listen("user.logged_in", &LogUserActivityListener{})
	dispatcher.Listen("user.logged_out", &LogUserActivityListener{})
	dispatcher.Listen("order.created", &SendOrderConfirmationListener{})
	
	dispatcher.ListenFunc("user.*", func(event Event) error {
		fmt.Printf("User event occurred: %s\n", event.GetName())
		return nil
	})
	
	dispatcher.ListenFunc("order.*", func(event Event) error {
		fmt.Printf("Order event occurred: %s\n", event.GetName())
		return nil
	})
}

type Observable struct {
	observers []Observer
	mutex     sync.RWMutex
}

type Observer interface {
	Update(subject interface{}, event string, data interface{})
}

func (o *Observable) Attach(observer Observer) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	o.observers = append(o.observers, observer)
}

func (o *Observable) Detach(observer Observer) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	
	for i, obs := range o.observers {
		if obs == observer {
			o.observers = append(o.observers[:i], o.observers[i+1:]...)
			break
		}
	}
}

func (o *Observable) Notify(subject interface{}, event string, data interface{}) {
	o.mutex.RLock()
	defer o.mutex.RUnlock()
	
	for _, observer := range o.observers {
		observer.Update(subject, event, data)
	}
}

type ModelObserver struct {
	name string
}

func NewModelObserver(name string) *ModelObserver {
	return &ModelObserver{name: name}
}

func (mo *ModelObserver) Update(subject interface{}, event string, data interface{}) {
	fmt.Printf("[%s] Model %s: %s with data: %+v\n", mo.name, reflect.TypeOf(subject).Name(), event, data)
}

func DispatchEvent(event Event) error {
	if globalApp != nil {
		dispatcher, _ := globalApp.Container().Make("events")
		if ed, ok := dispatcher.(EventDispatcher); ok {
			return ed.Dispatch(event)
		}
	}
	return fmt.Errorf("event dispatcher not configured")
}

func ListenForEvent(eventName string, listener Listener) {
	if globalApp != nil {
		dispatcher, _ := globalApp.Container().Make("events")
		if ed, ok := dispatcher.(EventDispatcher); ok {
			ed.Listen(eventName, listener)
		}
	}
}

func ListenForEventFunc(eventName string, listenerFunc ListenerFunc) {
	if globalApp != nil {
		dispatcher, _ := globalApp.Container().Make("events")
		if ed, ok := dispatcher.(EventDispatcher); ok {
			ed.ListenFunc(eventName, listenerFunc)
		}
	}
}

func (c *Context) DispatchEvent(event Event) error {
	dispatcher, _ := c.app.Container().Make("events")
	if ed, ok := dispatcher.(EventDispatcher); ok {
		return ed.Dispatch(event)
	}
	return fmt.Errorf("event dispatcher not configured")
}

func EventMiddleware() MiddlewareFunc {
	return func(c *Context) error {
		event := NewBaseEvent("request.received")
		c.DispatchEvent(event)
		
		err := c.Next()
		
		if err != nil {
			errorEvent := NewBaseEvent("request.failed")
			c.DispatchEvent(errorEvent)
		} else {
			successEvent := NewBaseEvent("request.completed")
			c.DispatchEvent(successEvent)
		}
		
		return err
	}
}