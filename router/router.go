package router

import (
	"github.com/gorilla/mux"
	"mj0lk.be/netwars/utils"
)

func NewRouter() *mux.Router {
	r := mux.NewRouter().StrictSlash(true)
	for prefix, routes := range routes {
		//check multiple paths (optional variables)
		subRouter := r.PathPrefix(prefix).Subrouter()
		for _, route := range routes {
			for _, path := range route.Path {
				handler := route.Handler
				if route.Auth {
					handler = utils.Validator(route.Handler)
				}
				subRouter.HandleFunc(path, handler).
					Methods(route.Method).
					Name(route.Name)
				if len(route.Headers) > 0 {
					subRouter.Headers(route.Headers...)
				}
			}
		}
	}
	return r
}
