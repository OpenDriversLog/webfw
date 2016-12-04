// webfw is a small MVC-web-framework for handling webrequests with chained controllers, inspired by this tutorial :
// https://github.com/QLeelulu/goku and http://nicolasmerouze.com/build-web-framework-golang
package webfw

import (
	"net/http"

	"golang.org/x/net/context"
)

// Controller is the core controller determing which view to use, what data to display and how to handle the data.
type Controller interface {
	GetViewData(ctx context.Context, r *http.Request) (vd ViewData, vPath string, vSharedTemplate string, err error)
}

// ErrorController is used when an error is used, to show an error message.
type ErrorController interface {
	GetViewData(vd ViewData, errIn error) (vdNew ViewData, vPath string, vSharedTemplate string, err error)
}
