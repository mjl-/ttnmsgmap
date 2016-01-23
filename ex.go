package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
)

func ex(fn http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				switch e := err.(type) {
				case HttpUserError:
					http.Error(w, e.Error(), http.StatusBadRequest)
				case HttpServerError:
					http.Error(w, e.Error(), http.StatusInternalServerError)
				case HttpError:
					http.Error(w, http.StatusText(int(e)), int(e))
				default:
					// xxx only do this when no header was written yet
					http.Error(w, http.StatusText(500), 500)
					panic(err)
				}
			}
		}()
		fn.ServeHTTP(w, r)
	}
}


type HttpError int
type HttpUserError error
type HttpServerError error

func (e HttpError) Error() string {
	return fmt.Sprintf("%d %s", int(e), http.StatusText(int(e)))
}

func needmethod(r *http.Request, methods ...string) {
	for _, m := range methods {
		if r.Method == m {
			return
		}
	}
	abort(405)
}

func needpost(r *http.Request) {
	needmethod(r, "POST")
}

func needget(r *http.Request) {
	needmethod(r, "GET")
}

func abort(e int) {
	panic(HttpError(e))
}

func abortUserError(s string) {
	panic(HttpUserError(errors.New(s)))
}

func abortServerError(s string) {
	panic(HttpServerError(errors.New(s)))
}

func check(err error) {
	if err != nil {
		log.Println(err)
		abort(500)
	}
}

