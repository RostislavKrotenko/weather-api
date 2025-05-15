package server

import (
	"database/sql"
    "net/http"

    "github.com/go-chi/chi/v5"
)

func New(dbConn *sql.DB) *http.Server {
    r := chi.NewRouter()

    r.Handle("/swagger.yaml", http.FileServer(http.Dir(".")))
    r.Handle("/docs/*", http.StripPrefix("/docs/", http.FileServer(http.Dir("swagger-ui"))))

    h := &handler{db: dbConn}
    r.Route("/api", func(r chi.Router) {
        r.Get("/weather", h.GetWeather)

        r.Post("/subscribe", h.Subscribe)
        r.Get("/confirm/{token}", h.ConfirmSubscription)
        r.Get("/unsubscribe/{token}", h.Unsubscribe)
    })

    fs := http.FileServer(http.Dir("./public"))
    r.Handle("/*", fs)

    return &http.Server{Handler: r}
}