package controller

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/robfig/cron/v3"
	"github.com/waiwen1001/bike/middleware"
	"github.com/waiwen1001/bike/models"
	"github.com/waiwen1001/bike/utils"
)

type APIServer struct {
	listenAddr string
	store      models.Storage
}

type apiFunc func(http.ResponseWriter, *http.Request) error

type APIResponse struct {
	Status  int
	Message string
	Data    any
}

var cache sync.Map

func ResponseJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

func makeHttpHandleFunc(f apiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			status := http.StatusBadRequest
			ResponseJSON(w, status, APIResponse{Status: status, Message: err.Error()})
		}
	}
}

func NewAPIServer(listenAddr string, store models.Storage) *APIServer {
	return &APIServer{
		listenAddr: listenAddr,
		store:      store,
	}
}

func (s *APIServer) Run() {
	router := mux.NewRouter()

	apiRouter := router.PathPrefix("/api/v1").Subrouter()
	// for login
	apiRouter.HandleFunc("/check-auth", makeHttpHandleFunc(s.CheckAuth)).Methods("GET")
	apiRouter.HandleFunc("/login", makeHttpHandleFunc(s.Login)).Methods("POST")
	apiRouter.HandleFunc("/logout", makeHttpHandleFunc(s.Logout)).Methods("POST")

	protectedRouter := apiRouter.NewRoute().Subrouter()
	protectedRouter.Use(middleware.TokenAuthMiddleware)

	protectedRouter.HandleFunc("/indego-data-fetch-and-store-it-db", makeHttpHandleFunc(s.FetchIndegoData)).Methods("POST")
	protectedRouter.HandleFunc("/stations", makeHttpHandleFunc(s.GetStations)).Methods("GET")
	protectedRouter.HandleFunc("/stations/{kioskId}", makeHttpHandleFunc(s.GetStation)).Methods("GET")

	router.MethodNotAllowedHandler = makeHttpHandleFunc(s.ShowAPIError)

	corsOptions := []handlers.CORSOption{
		handlers.AllowedOrigins([]string{"http://localhost:5173"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Token"}),
	}

	// initial cron job call fetch indego data every hour
	c := cron.New()
	_, err := c.AddFunc("0 * * * *", func() {
		err := s.FetchIndegoData(nil, nil)
		if err != nil {
			log.Printf("Error fetching and storing Indego data: %v", err)
		}
	})
	if err != nil {
		log.Fatalf("Error adding cron job: %v", err)
	}

	// Start the cron job
	c.Start()

	log.Println("API running at port", s.listenAddr)
	http.ListenAndServe(s.listenAddr, handlers.CORS(corsOptions...)(router))
}

func (s *APIServer) ShowAPIError(w http.ResponseWriter, r *http.Request) error {
	errMsg := "Error : Method not allowed"
	status := http.StatusMethodNotAllowed
	return ResponseJSON(w, status, APIResponse{Status: status, Message: errMsg})
}

func (s *APIServer) FetchIndegoData(w http.ResponseWriter, r *http.Request) error {
	now := time.Now()
	apiUrl := "https://bts-status.bicycletransit.workers.dev/phl"
	resp, err := http.Get(apiUrl)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	var data models.IndegoRes
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return err
	}

	log.Printf("Total time used for fetching Indego API %v", time.Since(now))

	now2 := time.Now()
	err = s.store.StoreIndegoData(&data)
	log.Printf("Total time used for storing data %v", time.Since(now2))
	log.Printf("Total function used time %v", time.Since(now))

	if w == nil && r == nil {
		// for cron job
		if err != nil {
			return err
		}
		return nil
	}

	status := http.StatusOK
	if err != nil {
		status = http.StatusBadRequest
		return ResponseJSON(w, status, APIResponse{Status: status, Message: "Fetch and store indego data failed"})
	}

	t, err := utils.ParseTime(data.LastUpdated)
	if err != nil {
		fmt.Println("Error parsing time:", err)
		status := http.StatusBadRequest
		return ResponseJSON(w, status, APIResponse{Status: status, Message: "Invalid time format"})
	}

	dateTime := t.Format("2006-01-02 15:04:05")

	return ResponseJSON(w, status, APIResponse{Status: status, Message: "Indego data fetch and store successfully", Data: dateTime})
}

func (s *APIServer) GetStations(w http.ResponseWriter, r *http.Request) error {
	now := time.Now()
	at := r.URL.Query().Get("at")
	log.Printf("Receive at : %v", at)
	if at == "" {
		errMsg := "Error : at cannot be empty"
		status := http.StatusNotFound
		return ResponseJSON(w, status, APIResponse{Status: status, Message: errMsg})
	}

	t, err := utils.ParseTime(at)
	if err != nil {
		fmt.Println("Error parsing time:", err)
		status := http.StatusBadRequest
		return ResponseJSON(w, status, APIResponse{Status: status, Message: "Invalid time format"})
	}

	dateTime := t.Format("2006-01-02 15:04:05")

	data, err := s.store.GetStationList(dateTime)
	log.Printf("Total time used for get all stations data %+v", time.Since(now))
	status := http.StatusOK
	if err != nil {
		status = http.StatusBadRequest
		return ResponseJSON(w, status, APIResponse{Status: status, Message: "Get stations failed"})
	}

	// capture station weather
	now2 := time.Now()
	api := os.Getenv("OPEN_WEATHER_APIKEY")
	var wg sync.WaitGroup
	ch := make(chan models.WeatherResult)
	for i, v := range data {
		wg.Add(1)
		p := v.Stations.Properties
		go s.FetchWeather(api, i, p.Latitude, p.Longitude, ch, &wg)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()
	log.Printf("Total time used for fetching weather %+v", time.Since(now2))
	// end

	for result := range ch {
		data[result.Index].Weather = result.Weather
	}

	response := APIResponse{Status: status, Message: "Success", Data: data}

	return ResponseJSON(w, status, response)
}

func (s *APIServer) GetStation(w http.ResponseWriter, r *http.Request) error {
	now := time.Now()
	vars := mux.Vars(r)
	at := r.URL.Query().Get("at")
	kioskId := vars["kioskId"]
	if at == "" {
		errMsg := "Error : at cannot be empty"
		status := http.StatusNotFound
		return ResponseJSON(w, status, APIResponse{Status: status, Message: errMsg})
	}

	t, err := utils.ParseTime(at)
	if err != nil {
		fmt.Println("Error parsing time:", err)
		status := http.StatusBadRequest
		return ResponseJSON(w, status, APIResponse{Status: status, Message: "Invalid time format"})
	}

	dateTime := t.Format("2006-01-02 15:04:05")

	data, err := s.store.GetStation(dateTime, kioskId)
	log.Printf("Total time used for get kioskId : %v stations data %+v", kioskId, time.Since(now))
	status := http.StatusOK
	if err != nil {
		if strings.Contains(err.Error(), "empty row") {
			return ResponseJSON(w, http.StatusOK, APIResponse{Status: status, Message: "Success"})
		}
		status = http.StatusBadRequest
		return ResponseJSON(w, status, APIResponse{Status: status, Message: "Get station failed"})
	}

	// capture station weather
	now2 := time.Now()
	api := os.Getenv("OPEN_WEATHER_APIKEY")
	var wg sync.WaitGroup
	ch := make(chan models.WeatherResult)
	wg.Add(1)
	p := data.Stations.Properties
	go s.FetchWeather(api, 0, p.Latitude, p.Longitude, ch, &wg)

	go func() {
		wg.Wait()
		close(ch)
	}()
	log.Printf("Total time used for fetching weather %+v", time.Since(now2))
	// end

	for result := range ch {
		data.Weather = result.Weather
	}

	response := APIResponse{Status: status, Message: "Success", Data: data}

	return ResponseJSON(w, http.StatusOK, response)
}

func (s *APIServer) FetchWeather(api string, index int, latitude float64, longitude float64, ch chan<- models.WeatherResult, wg *sync.WaitGroup) {
	defer wg.Done()
	apiUrl := fmt.Sprintf("https://api.openweathermap.org/data/2.5/weather?lat=%f&lon=%f&appid=%v", latitude, longitude, api)
	// if more than 10 seconds go timeout
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(apiUrl)
	if err != nil {
		log.Printf("Error fetching open weather API %v", err)
		return
	}

	defer resp.Body.Close()
	var data models.Weather
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Printf("Error parsing open weather response err: %v", err)
		return
	}

	ch <- models.WeatherResult{Index: index, Weather: data}
}

func (s *APIServer) CheckAuth(w http.ResponseWriter, r *http.Request) error {
	log.Printf("Check auth")
	if _, ok := cache.Load("user"); ok {
		return ResponseJSON(w, http.StatusOK, APIResponse{Status: http.StatusOK, Message: "Authorized"})
	}
	return ResponseJSON(w, http.StatusUnauthorized, APIResponse{Status: http.StatusUnauthorized, Message: "Unauthorized"})
}

func (s *APIServer) Login(w http.ResponseWriter, r *http.Request) error {
	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "user" && password == "123456" {
		cache.Store(username, true)
		return ResponseJSON(w, http.StatusOK, APIResponse{Status: http.StatusOK, Message: "Success"})
	}

	return ResponseJSON(w, http.StatusUnauthorized, APIResponse{Status: http.StatusUnauthorized, Message: "Login failed"})
}

func (s *APIServer) Logout(w http.ResponseWriter, r *http.Request) error {
	cache.Store("user", false)

	return ResponseJSON(w, http.StatusOK, APIResponse{Status: http.StatusOK, Message: "Success"})
}
