package webfw

import (
	"errors"
	"html/template"
	"io/ioutil"
	"net/http"

	"fmt"
	"io"

	"github.com/Compufreak345/dbg"
	"github.com/OpenDriversLog/goodl-lib/translate"
	"sync"
)

const vTag = dbg.Tag("webfw/viewengine.go")

// ViewEngine is used to display views by applying templates.
type ViewEngine struct {
	templateCache   map[string]*template.Template
	SharedTemplates map[string]*template.Template
	tempCacheMutex *sync.Mutex
	sharedMutex *sync.Mutex
}

// ViewData determines which view will be shown and what context it uses.
type ViewData struct {
	Data           map[string]interface{}
	Model          IModel
	Globals        map[string]interface{}
	T              *translate.Translater
	ErrorType      int
	ErrorSource    string
	ErrorMessage   interface{}
	WarningMessage interface{}
	StatusMessage  interface{}
	Debug          bool
	ViewName       string
	Redirect       string
	NoStyleOnError bool
	//Body    interface{} // if in layout template, this will set
}

// NewViewEngine returns a new ViewEngine.
func NewViewEngine() *ViewEngine {
	sharedDir := Config().RootDir + "/views/shared/"
	e := &ViewEngine{templateCache: make(map[string]*template.Template),
		SharedTemplates: make(map[string]*template.Template),
		tempCacheMutex:&sync.Mutex{},
		sharedMutex:&sync.Mutex{},
	}
	files, _ := ioutil.ReadDir(sharedDir)
	e.sharedMutex.Lock()
	for _, file := range files {
		e.SharedTemplates[file.Name()] = template.Must(template.New(file.Name()).Delims("{[{", "}]}").ParseGlob(sharedDir + file.Name()))
	}
	e.sharedMutex.Unlock()

	return e
}

// GetTemplate gets the template with the given key.
func (v *ViewEngine) GetTemplate(key string, path string, sharedTemplateToUse string) (t *template.Template, err error) {

	dbg.D(vTag, "Start GetTemplate for %s,%s,%s", key, path, sharedTemplateToUse)
	key = key + "-_-" + sharedTemplateToUse
	v.tempCacheMutex.Lock()
	mt, ok := v.templateCache[key]
	v.tempCacheMutex.Unlock()
	if !dbg.Develop && ok {
		t = mt
		dbg.D(vTag, "End GetTemplate (template cached)")
		return
	}

	tmplTxt, err := ioutil.ReadFile(path)
	if err != nil {
		dbg.W(vTag, "End GetTemplate with error %v", err)
		return
	}
	// TODO : Maybe allow multiple templates
	var x *template.Template
	if sharedTemplateToUse != "" {
		v.sharedMutex.Lock()
		x, _ = v.SharedTemplates[sharedTemplateToUse].Clone()
		v.sharedMutex.Unlock()
	} else {
		x = template.New(key)
	}
	x.Delims("{[{", "}]}")
	t = template.Must(x.Parse(string(tmplTxt)))

	v.tempCacheMutex.Lock()
	v.templateCache[key] = t
	v.tempCacheMutex.Unlock()
	dbg.D(vTag, "End GetTemplate ")
	return

}

// ClearCache clears the cached files, initializing a new ViewEngine
func (v *ViewEngine) ClearCache() {
	v = NewViewEngine()
}

// RenderHttpResp renders the given view.
func (v *ViewEngine) RenderHttpResp(vd ViewData, t *template.Template, w http.ResponseWriter, r *http.Request, name string) (err error) {
	vd.Debug = bool(dbg.Debugging)
	dbg.D(vTag, "Start Render ")

	defer func() {
		if r := recover(); r != nil {
			dbg.W(vTag, "Error rendering template ", r)
			http.Error(w, http.StatusText(500), 500)
			err = errors.New("Unknown error rendering template")
			return
		}
	}()
	if vd.Redirect != "" {
		http.Redirect(w, r, vd.Redirect, 307)
	}
	err = v.RenderWriter(vd, t, w, name)
	if err != nil {
		dbg.W(vTag, "Error rendering template : ", err)
		DirectShowError(vd, err, w)
		return
	}
	dbg.D(vTag, "End Render ")
	return
}

// RenderWriter writes the given view to the output writer.
func (v *ViewEngine) RenderWriter(vd ViewData, t *template.Template, w io.Writer, name string) (err error) {
	vd.Debug = bool(dbg.Debugging)
	dbg.D(vTag, "Start RenderWriter ")

	defer func() {
		if r := recover(); r != nil {
			dbg.W(vTag, "Error rendering template ", r)
			err = errors.New(fmt.Sprintf("", r))
			return
		}
	}()
	// t = t.Delims("{[{", "}]}")

	if name == "" {
		err = t.Execute(w, interface{}(vd))
	} else {
		err = t.ExecuteTemplate(w, name, interface{}(vd))

	}
	if err != nil {
		dbg.W(vTag, "Error rendering template : ", err)
		return
	}
	dbg.D(vTag, "End RenderWriter ")
	return
}

// HTML calls the Translator and returns the answer as template.HTML
func (v *ViewData) HTML(s string, args ...interface{}) template.HTML {
	return template.HTML(v.T.T(s, args...))
}
