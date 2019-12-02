package goat

import (
	// "errors"
	"html"
	"log"
	"net/http"
	"strings"

	"github.com/emuggie/goat/plugin"
	// plugin "gff/plugin"
)

type App struct {
	basePath string
	beanMap  map[string]interface{}
}

func NewApp(basePath string) *App {
	var app = new(App)
	app.beanMap = make(map[string]interface{})
	app.basePath = basePath
	return app
}

func (this *App) AddBean(name string, bean interface{}) bool {
	log.Println(bean)
	if this.beanMap[name] != nil {
		return false
	}
	this.beanMap[name] = bean
	return true
}

func (this *App) GetBean(name string) interface{} {
	return this.beanMap[name]
}

func (this *App) RemoveBean(name string) {
	delete(this.beanMap, name)
}

/**
Isolated goroutine
*/
func (this *App) Handle(req *http.Request, res http.ResponseWriter) {

	requestPath := strings.Trim(req.URL.Path, "/")

	//lookup
	targetContext := plugin.RequestContextNew(this.basePath + "/" + requestPath)
	if targetContext == nil || !targetContext.Has(strings.ToUpper(req.Method)) {
		return
	}

	//Preparation : Context initialization
	scopedBean := make(map[string]interface{})
	scopedBean["Request"] = req
	scopedBean["Response"] = res

	//Copy Beans from app
	for k, v := range this.beanMap {
		scopedBean[k] = v
	}

	//Depth chain
	dirPath := strings.Split(requestPath, "/")
	dirPath = dirPath[0 : len(dirPath)-1]
	var contextStack = make([]*plugin.RequestContext, len(dirPath))
	lookupPath := ""
	for i, dir := range dirPath {
		lookupPath += "/" + html.EscapeString(dir)
		pathContext := plugin.RequestContextNew(this.basePath + lookupPath)
		if pathContext == nil || (!pathContext.Has(plugin.PreHandleFunction) && !pathContext.Has(plugin.PostHandleFunction)) {
			continue
		}
		// Inject context
		pathContext.InjectAll(&scopedBean)
		if pathContext.Has(plugin.PreHandleFunction) {
			pathContext.Invoke(plugin.PreHandleFunction)
		}
		contextStack[i] = pathContext
	}

	targetContext.InjectAll(&scopedBean)
	if targetContext.Has(plugin.PreHandleFunction) {
		targetContext.Invoke(plugin.PreHandleFunction)
	}
	// Handle
	targetContext.Invoke(strings.ToUpper(req.Method))

	if targetContext.Has(plugin.PostHandleFunction) {
		targetContext.Invoke(plugin.PostHandleFunction)
	}

	// Post handle from stack
	for i := len(contextStack) - 1; i >= 0; i-- {
		if contextStack[i] == nil {
			continue
		}
		if contextStack[i].Has(plugin.PostHandleFunction) {
			contextStack[i].Invoke(plugin.PostHandleFunction)
		}
	}
}
