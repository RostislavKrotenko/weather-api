package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type handler struct {
    db *sql.DB
}

type subscribeReq struct {
    Email     string `json:"email" form:"email"`
    City      string `json:"city"  form:"city"`
    Frequency string `json:"frequency" form:"frequency"`
}

type subscribeRes struct {
    Message string `json:"message"`
    Token   string `json:"token"`
}

type WeatherResponse struct {
    Temperature float64 `json:"temperature"`
    Humidity    float64 `json:"humidity"`
    Description string  `json:"description"`
}

func (h *handler) Subscribe(w http.ResponseWriter, r *http.Request) {
    var req subscribeReq
    if err := r.ParseForm(); err == nil && r.FormValue("email") != "" {
        req.Email = r.FormValue("email")
        req.City = r.FormValue("city")
        req.Frequency = r.FormValue("frequency")
    } else {
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "invalid request body", http.StatusBadRequest)
            return
        }
    }

    id := uuid.New().String()
    token := uuid.New().String()
    _, err := h.db.Exec(
        `INSERT INTO subscriptions (id, email, city, frequency, token, confirmed, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
        id, req.Email, req.City, req.Frequency, token, false, time.Now(),
    )
    if err != nil {
        http.Error(w, "failed to save subscription", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    resp := subscribeRes{
        Message: "Subscription successful. Confirmation email sent.",
        Token:   token,
    }
    json.NewEncoder(w).Encode(resp)
}

func (h *handler) ConfirmSubscription(w http.ResponseWriter, r *http.Request) {
    token := chi.URLParam(r, "token")
    res, err := h.db.Exec(`UPDATE subscriptions SET confirmed = true WHERE token = $1`, token)
    if err != nil {
        http.Error(w, "failed to confirm subscription", http.StatusInternalServerError)
        return
    }
    if count, _ := res.RowsAffected(); count == 0 {
        http.Error(w, "token not found", http.StatusNotFound)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(`{"message":"Subscription confirmed successfully."}`))
}

func (h *handler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
    token := chi.URLParam(r, "token")
    res, err := h.db.Exec(`DELETE FROM subscriptions WHERE token = $1`, token)
    if err != nil {
        http.Error(w, "failed to unsubscribe", http.StatusInternalServerError)
        return
    }
    if count, _ := res.RowsAffected(); count == 0 {
        http.Error(w, "token not found", http.StatusNotFound)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(`{"message":"Unsubscribed successfully."}`))
}

func (h *handler) GetWeather(w http.ResponseWriter, r *http.Request) {
    city := r.URL.Query().Get("city")
    if city == "" {
        http.Error(w, "query param 'city' is required", http.StatusBadRequest)
        return
    }

    apiKey := os.Getenv("OPENWEATHER_API_KEY")
    if apiKey == "" {
        log.Fatal("missing OPENWEATHER_API_KEY")
    }

    // Формуємо URL для запиту
    endpoint := fmt.Sprintf(
        "https://api.openweathermap.org/data/2.5/weather?q=%s&appid=%s&units=metric",
        url.QueryEscape(city),
        apiKey,
    )

    resp, err := http.Get(endpoint)
    if err != nil {
        http.Error(w, "failed to fetch weather", http.StatusBadGateway)
        return
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        http.Error(w, "weather provider returned "+resp.Status, http.StatusBadGateway)
        return
    }

    // Парсимо відповідь OpenWeatherMap
    var owm struct {
        Main struct {
            Temp     float64 `json:"temp"`
            Humidity float64 `json:"humidity"`
        } `json:"main"`
        Weather []struct {
            Description string `json:"description"`
        } `json:"weather"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&owm); err != nil {
        http.Error(w, "invalid weather response", http.StatusInternalServerError)
        return
    }

    // Формуємо власну модель відповіді
    result := WeatherResponse{
        Temperature: owm.Main.Temp,
        Humidity:    owm.Main.Humidity,
        Description: func() string {
            if len(owm.Weather) > 0 {
                return owm.Weather[0].Description
            }
            return ""
        }(),
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(result)
}