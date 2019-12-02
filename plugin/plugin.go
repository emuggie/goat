package plugin

import (
	"plugin"
	"reflect"
	"strings"
)

const TagPrefix = "Inject"
const MethodName = "New"
const PreHandleFunction = "Before"
const PostHandleFunction = "After"

var pluginCache = make(map[string](*plugin.Plugin))

func Open(path string) (*plugin.Plugin, error) {
	cached := pluginCache[path]
	if cached != nil {
		return cached, nil
	}
	pgn, err := plugin.Open(path)
	if err != nil {
		return nil, err
	}
	pluginCache[path] = pgn
	return pgn, nil
}

func Lookup(path string, name string) plugin.Symbol {
	if !strings.HasSuffix(path, ".so") {
		path += ".so"
	}
	pgn, err := Open(path)
	if err != nil {
		return nil
	}
	loaded, err := pgn.Lookup(name)
	if err != nil {
		return nil
	}
	return loaded
}

type RequestContext struct {
	handler  interface{}
	elements reflect.Value
}

func (this *RequestContext) Has(name string) bool {
	method := reflect.ValueOf(this.handler).MethodByName(name)
	if method.IsValid() {
		return true
	}
	return false
}

func (this *RequestContext) Invoke(name string) bool {
	method := reflect.ValueOf(this.handler).MethodByName(name)
	if method.IsValid() {
		method.Call(nil)
		return true
	}
	return false
}

func (this *RequestContext) Inject(tagName string, value interface{}) {
	for i := 0; i < this.elements.NumField(); i++ {
		fType := this.elements.Type().Field(i)
		key, ok := fType.Tag.Lookup("Inject")
		if !ok || key != tagName {
			continue
		}

		field := this.elements.Field(i)
		if !field.IsValid() || !field.CanSet() {
			// log.Println("Error : not Set" ,key, field.IsValid(),field.CanSet())
			panic("Error Injecting field : " + key)
		}
		field.Set(reflect.ValueOf(value))
	}
}

func (this *RequestContext) InjectAll(valueMap *map[string]interface{}) {
	for i := 0; i < this.elements.NumField(); i++ {
		fType := this.elements.Type().Field(i)
		key, ok := fType.Tag.Lookup("Inject")
		if !ok || (*valueMap)[key] == nil {
			continue
		}

		field := this.elements.Field(i)
		if field.IsValid() && field.CanSet() {
			field.Set(reflect.ValueOf((*valueMap)[key]))
		}
	}
}

func RequestContextNew(path string) *RequestContext {
	//lookup
	ScopeNew := Lookup(path, "New")
	if ScopeNew == nil {
		return nil
	}
	context := new(RequestContext)
	context.handler = ScopeNew.(func() interface{})()
	target := reflect.ValueOf(context.handler)
	context.elements = target.Elem()
	return context
}

func RequestContextExists(target interface{}) *RequestContext {
	context := new(RequestContext)
	context.handler = target
	t := reflect.ValueOf(context.handler)
	context.elements = t.Elem()
	return context
}
