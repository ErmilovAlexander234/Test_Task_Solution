package transporthttp

import (
	"net/http"

	"github.com/gorilla/mux"

	swaggerdocs "example.com/taskservice/internal/transport/http/docs"
	httphandlers "example.com/taskservice/internal/transport/http/handlers"
)

func NewRouter(taskHandler *httphandlers.TaskHandler, docsHandler *swaggerdocs.Handler) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)

	//----------------свага-------------------------
	router.HandleFunc("/swagger/openapi.json", docsHandler.ServeSpec).Methods(http.MethodGet)
	router.HandleFunc("/swagger/", docsHandler.ServeUI).Methods(http.MethodGet)
	router.HandleFunc("/swagger", docsHandler.RedirectToUI).Methods(http.MethodGet)

	api := router.PathPrefix("/api/v1").Subrouter()

	//------------------круд-------------------
	api.HandleFunc("/tasks", taskHandler.Create).Methods(http.MethodPost)
	api.HandleFunc("/tasks", taskHandler.List).Methods(http.MethodGet)
	api.HandleFunc("/tasks/{id:[0-9]+}", taskHandler.GetByID).Methods(http.MethodGet)
	api.HandleFunc("/tasks/{id:[0-9]+}", taskHandler.Update).Methods(http.MethodPut)
	api.HandleFunc("/tasks/{id:[0-9]+}", taskHandler.Delete).Methods(http.MethodDelete)

	//-----------------повторяющиеся таски--------------------------
	api.HandleFunc("/tasks/recurring", taskHandler.CreateRecurring).Methods(http.MethodPost)

	//-----------------правила повтора--------------------------------
	api.HandleFunc("/recurrence-rules", taskHandler.ListRecurrenceRules).Methods(http.MethodGet)
	api.HandleFunc("/recurrence-rules/{id:[0-9]+}", taskHandler.GetRecurrenceRule).Methods(http.MethodGet)
	api.HandleFunc("/recurrence-rules/{id:[0-9]+}/tasks", taskHandler.ListTasksByRuleID).Methods(http.MethodGet)

	//-----------------удаление (правила+по правилу повтора)------------------
	api.HandleFunc("/recurrence-rules/{id:[0-9]+}/tasks", taskHandler.DeleteRuleTasks).Methods(http.MethodDelete)
	api.HandleFunc("/recurrence-rules/{id:[0-9]+}", taskHandler.DeleteRecurrenceRule).Methods(http.MethodDelete)

	return router
}
