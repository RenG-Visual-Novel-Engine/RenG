package object

import "sync"

type Environment struct {
	store map[string]Object
	outer *Environment
}

var (
	EnvMutex = &sync.RWMutex{}
)

func NewEnvironment() *Environment {
	s := make(map[string]Object)
	return &Environment{store: s, outer: nil}
}

func (e *Environment) Get(name string) (Object, bool) {
	EnvMutex.RLock()
	obj, ok := e.store[name]
	if !ok && e.outer != nil {
		EnvMutex.RUnlock()
		obj, ok = e.outer.Get(name)
	}
	EnvMutex.RUnlock()
	return obj, ok
}

func (e *Environment) Set(name string, val Object) Object {
	EnvMutex.Lock()
	e.store[name] = val
	EnvMutex.Unlock()
	return val
}

func (e *Environment) IsHere(name string) bool {
	EnvMutex.RLock()
	if _, ok := e.store[name]; ok {
		EnvMutex.RUnlock()
		return true
	}
	EnvMutex.RUnlock()
	return false
}

func NewEncloseEnvironment(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
	return env
}
