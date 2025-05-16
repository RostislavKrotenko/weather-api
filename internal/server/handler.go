package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type handler struct {
	db *sql.DB
}

type errorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorResponse{
		Code:    status,
		Message: msg,
	})
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

func (h *handler) GetWeather(w http.ResponseWriter, r *http.Request) {
	city := r.URL.Query().Get("city")
	if city == "" {
		writeError(w, http.StatusBadRequest, "city parameter is required")
		return
	}

	apiKey := os.Getenv("OPENWEATHER_API_KEY")
	if apiKey == "" {
		log.Fatal("missing OPENWEATHER_API_KEY")
	}

	endpoint := fmt.Sprintf(
		"https://api.openweathermap.org/data/2.5/weather?q=%s&appid=%s&units=metric",
		url.QueryEscape(city),
		apiKey,
	)

	resp, err := http.Get(endpoint)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to fetch weather")
		return
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		writeError(w, http.StatusNotFound, "City not found")
		return
	default:
		writeError(w, http.StatusBadGateway, "weather provider returned "+resp.Status)
		return
	}

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
		writeError(w, http.StatusInternalServerError, "invalid weather response")
		return
	}

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

func (h *handler) Subscribe(w http.ResponseWriter, r *http.Request) {
	var req subscribeReq
	if err := r.ParseForm(); err == nil && r.FormValue("email") != "" {
		req.Email = r.FormValue("email")
		req.City = r.FormValue("city")
		req.Frequency = r.FormValue("frequency")
	} else {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid input")
			return
		}
	}

	if req.Email == "" || req.City == "" || req.Frequency == "" {
		writeError(w, http.StatusBadRequest, "Invalid input")
		return
	}
	if req.Frequency != "hourly" && req.Frequency != "daily" {
		writeError(w, http.StatusBadRequest, "frequency must be hourly or daily")
		return
	}

	var exists bool
	if err := h.db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM subscriptions WHERE email=$1 AND city=$2)",
		req.Email, req.City,
	).Scan(&exists); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check existing subscription")
		return
	}
	if exists {
		writeError(w, http.StatusConflict, "Email already subscribed")
		return
	}

	id := uuid.New().String()
	token := uuid.New().String()
	if _, err := h.db.Exec(
		`INSERT INTO subscriptions (id, email, city, frequency, token, confirmed, created_at)
		 VALUES ($1,$2,$3,$4,$5,false,$6)`,
		id, req.Email, req.City, req.Frequency, token, time.Now(),
	); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save subscription")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(subscribeRes{
		Message: "Subscription successful. Confirmation email sent.",
		Token:   token,
	})
}

func (h *handler) ConfirmSubscription(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
    var tokenRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	if token == "" || !tokenRegex.MatchString(token) {
		writeError(w, http.StatusBadRequest, "Invalid token")
		return
	}

	res, err := h.db.Exec(`UPDATE subscriptions SET confirmed=true WHERE token=$1`, token)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to confirm subscription")
		return
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		writeError(w, http.StatusNotFound, "Token not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Subscription confirmed successfully",
	})
}

func (h *handler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
    var tokenRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	if token == "" || !tokenRegex.MatchString(token){
		writeError(w, http.StatusBadRequest, "Invalid token")
		return
	}

	res, err := h.db.Exec(`DELETE FROM subscriptions WHERE token=$1`, token)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to unsubscribe")
		return
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		writeError(w, http.StatusNotFound, "Token not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Unsubscribed successfully",
	})
}
