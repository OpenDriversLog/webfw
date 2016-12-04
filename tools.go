package webfw

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/Compufreak345/dbg"
	"github.com/OpenDriversLog/goodl-lib/translate"
)

const tTag = dbg.Tag("webfw/tools.go")

var translaters map[string]translate.Translater

// RecoverForTag shows an error page in case of an unknown panic.
func RecoverForTag(tag dbg.Tag, funcName string, r *http.Request, err error, noStyleOnError bool) (vd ViewData, viewPath string, vShared string, err2 error) {

	return GetErrorViewData(tag, 500, dbg.E(tag, "Unknown panic : %v  in method %v for request : %v", err, funcName, dbg.GetRequest(r)), "", err, noStyleOnError)

}

// GetErrorViewData returns ViewData displaying an error
func GetErrorViewData(source dbg.Tag, errtype int, timeKey int64, clientMessage string, err error, noStyleOnError bool) (vd ViewData, vPath string, vSharedTemplate string, err2 error) {
	dbg.I(tTag, "GetErrorViewData called")
	vd = ViewData{
		ErrorType:      errtype,
		ErrorSource:    string(source),
		ErrorMessage:   clientMessage,
		NoStyleOnError: noStyleOnError,
	}
	if clientMessage == "" {
		vd.ErrorMessage = http.StatusText(errtype)
	}
	vd.ErrorMessage = fmt.Sprintf("%v", vd.ErrorMessage)
	if err == nil {
		err = errors.New("Unknown error in webfw/tools.go/GetErrorViewData")
	}
	return vd, "", "", err
}

// GetTranslater gets a translater for the given language (e.g. "de-DE")
func GetTranslater(mainLang string) (translater *translate.Translater) {
	if translaters == nil {
		translaters = make(map[string]translate.Translater)
	}
	var trans = translaters[mainLang]
	if trans.DefaultLang == "" {
		lang := ""
		fallbackLang := "en-US"
		switch mainLang {
		case "de":
			lang = "de-DE"
		case "en":
			{
				lang = "en-US"
				fallbackLang = "de-DE"
			}
		default:
			lang = "de-DE"
		}
		trans = translate.Translater{
			DefaultLang:  lang,
			FallbackLang: fallbackLang,
			UrlLang:      mainLang,
		}
		translaters[mainLang] = trans
	}
	return &trans
}
