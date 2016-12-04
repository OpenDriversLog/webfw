package webfw

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/net/context"

	"github.com/Compufreak345/alice"
	gorillactx "github.com/gorilla/context"

	"github.com/Compufreak345/dbg"
	"github.com/OpenDriversLog/redistore"
	"github.com/OpenDriversLog/goodl-lib/translate"
	"sync"
)

const hTag = "webfw/handlers.go"

type MVCBinder struct {
	Ctrl Controller
}

var SessionStoreKey = []byte{152, 193, 128, 220, 47, 161, 2, 237, 144, 57,
	129, 100, 22, 48, 152, 232, 13, 223, 169, 201, 120, 81, 173, 196, 137, 86, 72, 30, 75, 51, 194, 42, 180, 67, 88, 231,
	114, 77, 162, 106, 30, 46, 125, 151, 135, 229, 75, 139, 17, 55, 120, 248, 236, 194, 115, 187, 200,
	117, 95, 164, 3, 209, 143, 20,
}
var SessionStore *redistore.RediStore
var storeInited = false
var MVCBinders map[string]MVCBinder
var V *ViewEngine
var FileCache = struct {
	sync.RWMutex
	m map[string][]byte
}{m: make(map[string][]byte)}


var MyErrorController ErrorController
var defaultTranslater *translate.Translater

// ErrorViewDataPolishFunc - If Non-MVC-Handlers throw an error (=DirectShowError_NoVD called), this function is used to polish the page
var ErrorViewDataPolishFunc func(*ViewData, context.Context, *http.Request, string) string

// DefaultTranslater returns the default translater (if not set, currently "de-DE")
func DefaultTranslater() *translate.Translater {

	if defaultTranslater != nil {
		return defaultTranslater
	}
	defaultTranslater = GetTranslater("de-DE")
	return defaultTranslater
}

// SetDefaultTranslater sets the default translater.
func SetDefaultTranslater(t *translate.Translater) {
	defaultTranslater = t
}

// init initializes the MVCBinders-map.
func init() {
	MVCBinders = make(map[string]MVCBinder)
}

/* Handlers as final step in a chain */

// GetMvcHandler returns a an alice.CtxHandler that is able to serve MVC-pages.
func GetMvcHandler(binderKey string, viewDataPolishFunc func(*ViewData, context.Context, *http.Request, string) string) alice.CtxHandler {
	return alice.CtxHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		MvcHandler(ctx, w, r, binderKey, viewDataPolishFunc)
	})
}

func GetProvideFolderContentHandler(folderRelative string, partToRemoveFromUrlPath string) alice.CtxHandler {
	return alice.CtxHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		ProvideFolderContentHandler(ctx, w, r, folderRelative, partToRemoveFromUrlPath, false, "")
	})
}

func DirectShowError_NoVD(ctx context.Context, w http.ResponseWriter, r *http.Request, err error, errorMessage string, errorType int, notStyled ...bool) {
	var T *translate.Translater
	if ctx != nil {
		if ctx.Value("T") != nil {
			T = ctx.Value("T").(*translate.Translater)
		}
	}
	if T == nil {
		T = DefaultTranslater()
	}

	vd := ViewData{
		Model:          &Model{},
		T:              T,
		ErrorMessage:   errorMessage,
		ErrorType:      errorType,
		NoStyleOnError: len(notStyled) > 0 && notStyled[0],
	}
	if ErrorViewDataPolishFunc != nil {
		// TODO : Find a better solution than passing the views/odl.html that should be unknown!
		ErrorViewDataPolishFunc(&vd, ctx, r, "views/odl.html")

	}
	DirectShowError(vd, err, w)
}

func ProvideFolderContentHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, folderRelative string, partToRemoveFromUrlPath string, asDownload bool, folderAbsolute string) (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			//showError(ctx, "File not found", w, r)
			DirectShowError_NoVD(ctx, w, r, errors.New(fmt.Sprintf("Error in Recover ProvideFolderContentHandler : ", rec)), http.StatusText(404), 404, true)

			dbg.I(hTag, "ProvideFolderContentHandler recovered ")
			err = errors.New("ProvideFolderContentHandler recovered ")
			return
		}
	}()

	var dir http.Dir
	if folderAbsolute == "" {
		dir = http.Dir(Config().RootDir + "/" + folderRelative)
	} else {
		dir = http.Dir(folderAbsolute)
	}
	u, err := url.Parse(r.RequestURI)
	if err != nil {
		DirectShowError_NoVD(ctx, w, r, err, http.StatusText(404), 404, true)
		return err
	}

	FileCache.RLock()
	cached, ok := FileCache.m[u.Path]
	FileCache.RUnlock()
	fileCached := bool(!dbg.Develop) && ok

	if fileCached {
		if asDownload {
			w.Header().Set("Content-Disposition", "attachment")
		}
		http.ServeContent(w, r, u.Path, time.Time{}, bytes.NewReader(cached))
		return
	}

	p := u.Path
	if config.SubDir != "" {
		p = strings.Replace(p, config.SubDir, "", 1)
	}
	path := filepath.Clean(strings.Replace(p, partToRemoveFromUrlPath, "", 1))

	f, err := dir.Open(path)

	var b bytes.Buffer
	bWriter := bufio.NewWriter(&b)
	if err == nil {
		defer f.Close()
		buf := make([]byte, 1024)
		for {
			n, err := f.Read(buf)
			if err != nil && err != io.EOF {
				DirectShowError(ViewData{ErrorType: 500, NoStyleOnError: true}, err, w)
				dbg.W(hTag, "ProvideFolderContentHandler - error reading file ", err)

				return err
			}
			if n == 0 {
				break
			}
			//fmt.Fprint(w, string(buf[:n]))
			fmt.Fprint(bWriter, string(buf[:n]))

		}
		if err == nil || err == io.EOF {
			bWriter.Flush()
			by := b.Bytes()
			FileCache.Lock()
			FileCache.m[u.Path] = by
			FileCache.Unlock()
			if asDownload {
				w.Header().Set("Content-Disposition", "attachment")
			}
			http.ServeContent(w, r, path, time.Time{}, bytes.NewReader(by))
			return

		}
	}
	DirectShowError_NoVD(ctx, w, r, nil, http.StatusText(404), 404, true)
	dbg.W(hTag, " File failed to read : ", path, "-", err)
	if err == nil {
		err = errors.New("Unknown failure while reading file")
	}
	return
}

// func AddMessagesToViewData takes FormValues statusMessage, warningMessage, errorMessage from the rewuest
// and appends them to the current Status/Error/Warning-Message - if ignoreIfOther is true it will not append
// if there is already another message
func AddMessagesToViewData(vd *ViewData, r *http.Request, ignoreIfOther bool) {

	if status := r.FormValue("statusMessage"); status != "" {
		msg := vd.StatusMessage
		vd.StatusMessage = template.HTML(appendMessage(fmt.Sprintf("%v", msg), status, ignoreIfOther))
	}
	if warning := r.FormValue("warningMessage"); warning != "" {
		msg := vd.WarningMessage
		vd.WarningMessage = template.HTML(appendMessage(fmt.Sprintf("%v", msg), warning, ignoreIfOther))
	}
	if erro := r.FormValue("errorMessage"); erro != "" {
		msg := vd.ErrorMessage
		vd.ErrorMessage = template.HTML(appendMessage(fmt.Sprintf("%v", msg), erro, ignoreIfOther))
	}

}

func appendMessage(origmsg string, newmsg string, ignoreIfOther bool) string {

	if origmsg == "<nil>" {
		origmsg = ""
	}
	iAmNotFirst := origmsg != ""
	if iAmNotFirst && ignoreIfOther {
		return origmsg
	}
	if iAmNotFirst {
		origmsg += "<br/>"
	}

	origmsg += newmsg
	return origmsg
}

// func DirectShowError() Displays http error response with data provided in ViewData.**
func DirectShowError(vd ViewData, err error, w http.ResponseWriter) {

	if err != nil && vd.ErrorType == 0 {
		vd.ErrorType = 500

	}
	if vd.ErrorSource == "" {
		dbg.W(hTag, "Hmm, no ErrorSource provided - it would be better if you add a ErrorSource to your viewdata")
		vd.ErrorSource = hTag
	}
	if vd.ErrorMessage == "" {
		vd.ErrorMessage = http.StatusText(vd.ErrorType)
	}

	dbg.I(hTag, "Error page shown: \t Source : %v \r\n\t Type : %v \r\n\t DebugMessage : %v \r\n\t ClientMessage : %v \r\n\t Error : %v",
		vd.ErrorSource, vd.ErrorType, vd.ErrorMessage, err)

	if MyErrorController == nil || vd.NoStyleOnError {
		http.Error(w, fmt.Sprintf("%v", vd.ErrorMessage), vd.ErrorType)
	} else {
		var b bytes.Buffer
		vd, vPath, vSharedTemplate, err := MyErrorController.GetViewData(vd, err)
		if err != nil {
			dbg.E(hTag, "Error getting ErrorController ViewData : ", err)
			http.Error(w, fmt.Sprintf("%v", vd.ErrorMessage), vd.ErrorType)
			return
		}
		tpl, err := V.GetTemplate(vPath, Config().RootDir+"/"+vPath, vSharedTemplate)
		if err != nil {
			dbg.E(hTag, "Error getting template for ErrorController ViewData : ", err)
			http.Error(w, fmt.Sprintf("%v", vd.ErrorMessage), vd.ErrorType)
		}
		err = V.RenderWriter(vd, tpl, &b, vd.ViewName)
		if err != nil {
			dbg.E(hTag, "Error rendering ErrorController ViewData : ", err)
			http.Error(w, fmt.Sprintf("%v", vd.ErrorMessage), vd.ErrorType)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(vd.ErrorType)
		// It might be redundant to first write it to a buffer and then to the response writer, but...
		// I might say it's more secure for errors,... Or I just thought it differently and now it's implemented this way ;)
		fmt.Fprintln(w, b.String())
	}
}

// MvcHandler is the entry point for handling requests - bind this (as last part of a chain) e.g. to http.Handle("/*")
func MvcHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, binderKey string, viewDataPolishFunc func(*ViewData, context.Context, *http.Request, string) string) {
	dbg.D(hTag, "Start MvcHandler")

	binder, foundTpl := MVCBinders[binderKey]

	if foundTpl {
		vd, vPath, vShared, err := binder.Ctrl.GetViewData(ctx, r)

		if vd.ViewName == "" {
			vd.ViewName = binderKey
		}
		if viewDataPolishFunc != nil {
			vPath = viewDataPolishFunc(&vd, ctx, r, vPath)
		}
		if vd.ErrorType != 0 || err != nil {
			// An error was returned - display http error code
			DirectShowError(vd, err, w)
			return
		}
		tpl, err := V.GetTemplate(vPath, Config().RootDir+"/"+vPath, vShared)
		foundTpl = err == nil
		if foundTpl {
			dbg.V(hTag, "Start render")
			V.RenderHttpResp(vd, tpl, w, r, "")
			dbg.V(hTag, "End render")
		} else {
			if strings.Contains(err.Error(), "no such file") {
				dbg.I("Template not found : ", vPath, err)
				DirectShowError_NoVD(ctx, w, r, err, http.StatusText(404), 404)
				return
			}
			dbg.E(hTag, "Error loading template %s : %v", vPath, err)
		}
	}
	if !foundTpl {
		dbg.E(hTag, "No view found for "+binderKey)
		DirectShowError_NoVD(ctx, w, r, nil, http.StatusText(500), 500)

	}
	dbg.D(hTag, "End MvcHandler")
}

// ClearCacheHandler will clear the cached files.
func ClearCacheHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	dbg.D(hTag, "Clearing cache")
	V.ClearCache()
	FileCache.Lock()
	FileCache.m = make(map[string][]byte)
	FileCache.Unlock()

}

/* Handlers for before final handlers */

// GorillaClearHandler clears a Gorilla-context.
func GorillaClearHandler(ctx context.Context, next alice.CtxHandler) alice.CtxHandler {
	fn := func(ctx context.Context, w http.ResponseWriter, r *http.Request) {


		gorillactx.ClearHandler(alice.CtxHandlerToHandlerFunc(ctx, next)).ServeHTTP(w, r)

	}

	return alice.CtxHandlerFunc(fn)
}

// LoggingHandler logs when a request started & ended, including the requests URL
func LoggingHandler(ctx context.Context, next alice.CtxHandler) alice.CtxHandler {
	fn := func(ctx context.Context, w http.ResponseWriter, r *http.Request) {

		t1 := time.Now()
		next.ServeHTTP(ctx, w, r)
		t2 := time.Now()
		if dbg.Develop {
			dbg.I(hTag, "[%s] %q %v\n", r.Method, r.URL.String(), t2.Sub(t1))

		} else {
			dbg.I(hTag, "[%s] %q %v\n", r.Method, r.URL.Path, t2.Sub(t1))
		}
	}

	return alice.CtxHandlerFunc(fn)
}

var redisInited bool

// InitHandler inits every request by initializing a context
// and starting the process in a new goroutine - if it does not finish before Config.MaxResponseTime,
// we will print Config.TimeoutMessage and return.
func InitHandler(ctx context.Context, next alice.CtxHandler) alice.CtxHandler {
	fn := func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				dbg.E(hTag, "panic in InitHandler: %v for request : %v", err, dbg.GetRequest(r))
				DirectShowError(ViewData{ErrorType: 500}, errors.New(fmt.Sprintf("%s", err)), w)

			}
		}()
		if dbg.Develop {
			dbg.D(hTag, "My url : %s ", r.RequestURI)
		} else {
			dbg.D(hTag, "My url : %s ", r.URL.Path)
		}
		serveFn := func(ch chan struct{}, ctx context.Context) {
			next.ServeHTTP(ctx, w, r)
			ch <- struct{}{}

		}

		WrapWithInit(serveFn, w, 0)
	}

	return alice.CtxHandlerFunc(fn)
}

// Note by Compu : I don't like the non-elegance of this, but well... My brain... Arrg... This is needed for the
// new httprouter used in goodl.go (to get rid of the Inithandler, you know?)

// Wraps a function with initialization-stuff (redis init, context-init, timeout-wrapping)
func WrapWithInit(fn func(ch chan struct{}, ctx context.Context), w http.ResponseWriter, customTimeout time.Duration) {
	if !redisInited {
		var err error
		SessionStore, err = redistore.NewRediStore(10, "tcp", config.RedisAddress, "", SessionStoreKey)

		if err != nil {
			dbg.E(hTag, "Error while initialising redistore : ")
			panic(err)
		}
		redisInited = true
	}
	ctx, ctxCancel := context.WithCancel(context.Background())

	serveChan := make(chan struct{})
	go fn(serveChan, ctx)

	timeout := Config().MaxResponseTime
	if customTimeout > 0 {
		timeout = customTimeout
	}
	// wait until either timeout reached (ctx.Done-channel is closed) or serveHTTP finished (serveChan gets a signal)
	for {
		select {
		case <-serveChan:
			{ // Everything OK - clean context
				ctxCancel()
				return
			}
		case <-time.After(timeout):
			{ // timed out. present error.
				ctxCancel()
				fmt.Fprintln(w, Config().TimeoutMessage)
				return
			}
		}
	}
}

// RecoverHandler - in case of an error, prints an error message and logs the error
func RecoverHandler(ctx context.Context, next alice.CtxHandler) alice.CtxHandler {
	fn := func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				dbg.E(hTag, "panic in RecoverHandler: %v for request : %v", err, dbg.GetRequest(r))
				DirectShowError(ViewData{ErrorType: 500}, errors.New(fmt.Sprintf("%s", err)), w)

			}
		}()

		next.ServeHTTP(ctx, w, r)
	}

	return alice.CtxHandlerFunc(fn)
}
