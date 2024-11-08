package controller

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/waiwen1001/bike/models"

	_ "github.com/lib/pq"
)

var ApiServer *APIServer

func TestMain(m *testing.M) {
	err := godotenv.Load("../.env")
	if err != nil {
		log.Fatal("cannot load .env file ", err)
	}

	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	connStr := fmt.Sprintf("user=%v dbname=%v password=%v sslmode=disable", dbUser, dbName, dbPassword)
	db, err := sql.Open("postgres", connStr)

	if err != nil {
		log.Fatal("cannot connect to db ", err)
	}

	testPostgres := &models.PostgresStore{
		Db: db,
	}

	defer db.Close()

	ApiServer = NewAPIServer(":8080", testPostgres)

	exitCode := m.Run()
	os.Exit(exitCode)
}

func TestFetchIndegoData(t *testing.T) {
	req, err := http.NewRequest("POST", "/api/v1/indego-data-fetch-and-store-it-db", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(makeHttpHandleFunc(ApiServer.FetchIndegoData))
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Indego data fetch and store successfully")
}

func TestGetStations(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/v1/stations?at=2024-11-08T07:30:11.051Z", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(makeHttpHandleFunc(ApiServer.GetStations))
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Success")
}

func TestGetStation(t *testing.T) {
	r := mux.NewRouter()
	r.HandleFunc("/api/v1/stations/{kioskId}", makeHttpHandleFunc(ApiServer.GetStation)).Methods("GET")
	req, err := http.NewRequest("GET", "/api/v1/stations/3005?at=2024-11-08T07:30:11.051Z", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Success")
}
