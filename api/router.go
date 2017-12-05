package api

import (
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/rancher/go-rancher/api"
	"github.com/rancher/go-rancher/client"
)

type HandleFuncWithError func(http.ResponseWriter, *http.Request) error

//HandleError handle error from operation
func handleError(s *client.Schemas, t func(http.ResponseWriter, *http.Request) (int, error)) http.Handler {
	return api.ApiHandler(s, http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if errorCode, err := t(rw, req); err != nil {
			logrus.Errorf("Got Error: %v", err)
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(errorCode)

			e := Error{
				Resource: client.Resource{
					Type: "error",
				},
				Status:   errorCode,
				Msg:      err.Error(),
				BaseType: "error",
			}
			api.GetApiContext(req).Write(&e)
		}
	}))
}

func NewRouter(s *Server) *mux.Router {
	schemas := newSchema()
	r := mux.NewRouter().StrictSlash(true)
	f := handleError

	versionsHandler := api.VersionsHandler(schemas, "v1")
	versionHandler := api.VersionHandler(schemas, "v1")

	//framework route
	r.Methods(http.MethodGet).Path("/").Handler(versionsHandler)
	r.Methods(http.MethodGet).Path("/v1").Handler(versionHandler)
	r.Methods(http.MethodGet).Path("/v1/apiversions").Handler(versionsHandler)
	r.Methods(http.MethodGet).Path("/v1/schemas").Handler(api.SchemasHandler(schemas))
	r.Methods(http.MethodGet).Path("/v1/schemas/{id}").Handler(api.SchemaHandler(schemas))

	//alert config route
	r.Methods(http.MethodGet).Path("/v1/config").Handler(f(schemas, s.getAlertConfig))
	r.Methods(http.MethodGet).Path("/v1/configs").Handler(f(schemas, s.getAlertConfig))

	//recipient route
	r.Methods(http.MethodGet).Path("/v1/recipient").Handler(f(schemas, s.listRecipient))
	r.Methods(http.MethodGet).Path("/v1/recipients").Handler(f(schemas, s.listRecipient))
	r.Methods(http.MethodPost).Path("/v1/recipients").Handler(f(schemas, s.createRecipient))
	r.Methods(http.MethodPost).Path("/v1/recipient").Handler(f(schemas, s.createRecipient))
	r.Methods(http.MethodGet).Path("/v1/recipients/{id}").Handler(f(schemas, s.getRecipient))
	r.Methods(http.MethodDelete).Path("/v1/recipients/{id}").Handler(f(schemas, s.deleteRecipient))
	r.Methods(http.MethodPut).Path("/v1/recipients/{id}").Handler(f(schemas, s.updateRecipient))

	//alert route
	r.Methods(http.MethodGet).Path("/v1/alert").Handler(f(schemas, s.listAlerts))
	r.Methods(http.MethodGet).Path("/v1/alerts").Handler(f(schemas, s.listAlerts))
	r.Methods(http.MethodPost).Path("/v1/alert").Handler(f(schemas, s.createAlert))
	r.Methods(http.MethodPost).Path("/v1/alerts").Handler(f(schemas, s.createAlert))
	r.Methods(http.MethodGet).Path("/v1/alerts/{id}").Handler(f(schemas, s.getAlert))
	r.Methods(http.MethodDelete).Path("/v1/alerts/{id}").Handler(f(schemas, s.deleteAlert))
	r.Methods(http.MethodPut).Path("/v1/alerts/{id}").Handler(f(schemas, s.updateAlert))

	alertConfigActions := map[string]http.Handler{
		"update": f(schemas, s.updateAlertConfig),
	}
	for name, actions := range alertConfigActions {
		r.Methods(http.MethodPost).Path("/v1/configs").Queries("action", name).Handler(actions)
	}

	/*
		alertActions := map[string]http.Handler{
			"enable":    f(schemas, s.activateAlert),
			"disable":   f(schemas, s.deactivateAlert),
			"silence":   f(schemas, s.silenceAlert),
			"unsilence": f(schemas, s.unsilenceAlert),
		}
		for name, actions := range alertActions {
			r.Methods(http.MethodPost).Path("/v1/alerts/{id}").Queries("action", name).Handler(actions)
		}
	*/

	return r
}
