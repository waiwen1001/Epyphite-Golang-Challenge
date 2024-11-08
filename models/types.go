package models

import "time"

type APIResponse struct {
	Status  int
	Message string
	Data    any
}

type IndegoRes struct {
	LastUpdated string    `json:"last_updated"`
	Features    []Feature `json:"features"`
}

type Feature struct {
	Geometry   Geometry   `json:"geometry"`
	Properties Properties `json:"properties"`
	Type       string     `json:"type"`
}

type Geometry struct {
	Coordinates []float64 `json:"coordinates"`
	Type        string    `json:"type"`
}

type Properties struct {
	Id                     int64      `json:"id"`
	Name                   string     `json:"name"`
	Coordinates            []float64  `json:"coordinates"`
	TotalDocks             int64      `json:"totalDocks"`
	DocksAvailable         int64      `json:"docksAvailable"`
	BikesAvailable         int64      `json:"bikesAvailable"`
	ClassicBikesAvailable  int64      `json:"classicBikesAvailable"`
	SmartBikesAvailable    int64      `json:"smartBikesAvailable"`
	ElectricBikesAvailable int64      `json:"electricBikesAvailable"`
	RewardBikesAvailable   int64      `json:"rewardBikesAvailable"`
	RewardDocksAvailable   int64      `json:"rewardDocksAvailable"`
	KioskStatus            string     `json:"kioskStatus"`
	KioskPublicStatus      string     `json:"kioskPublicStatus"`
	KioskConnectionStatus  string     `json:"kioskConnectionStatus"`
	KioskType              int64      `json:"kioskType"`
	AddressStreet          string     `json:"addressStreet"`
	AddressCity            string     `json:"addressCity"`
	AddressState           string     `json:"addressState"`
	AddressZipCode         string     `json:"addressZipCode"`
	Bikes                  []Bike     `json:"bikes"`
	CloseTime              *time.Time `json:"closeTime"`
	EventEnd               *time.Time `json:"eventEnd"`
	EventStart             *time.Time `json:"eventStart"`
	IsEventBased           bool       `json:"isEventBased"`
	IsVirtual              bool       `json:"isVirtual"`
	KioskId                int64      `json:"kioskId"`
	Notes                  string     `json:"notes"`
	OpenTime               *time.Time `json:"openTime"`
	PublicText             string     `json:"publicText"`
	TimeZone               string     `json:"timeZone"`
	TrikesAvailable        int64      `json:"trikesAvailable"`
	Latitude               float64    `json:"latitude"`
	Longitude              float64    `json:"longitude"`
}

type DbProperties struct {
	Uid          int64
	GeometryType string
	StationType  string
	UpdatedAt    string
	CreatedAt    time.Time
	Properties
}

type Bike struct {
	DockNumber  *int64 `json:"dockNumber"`
	IsElectric  *bool  `json:"isElectric"`
	IsAvailable *bool  `json:"isAvailable"`
	Battery     *int64 `json:"battery"`
}

type DbBike struct {
	StationId int64
	Bike
}

type MinProperties struct {
	Uid       int64
	KioskId   int64
	UpdatedAt string
}

type BikeResult struct {
	At       string  `json:"at"`
	Stations Feature `json:"stations"`
	Weather  Weather `json:"weather"`
}

type Weather struct {
	Coord      WeatherCoord  `json:"coord"`
	Weather    []WeatherInfo `json:"weather"`
	Base       string        `json:"base"`
	Main       WeatherMain   `json:"main"`
	Visibility int64         `json:"visibility"`
	Wind       WeatherWind   `json:"wind"`
	Clouds     WeatherCloud  `json:"clouds"`
	Dt         int64         `json:"dt"`
	Sys        WeatherSys    `json:"sys"`
	Timezone   int64         `json:"timezone"`
	Id         int64         `json:"id"`
	Name       string        `json:"name"`
	Cod        int64         `json:"cod"`
}

type WeatherResult struct {
	Weather
	Index int
}

type WeatherCoord struct {
	Lon float64 `json:"lon"`
	Lat float64 `json:"lat"`
}

type WeatherInfo struct {
	Id          int64  `json:"id"`
	Main        string `json:"main"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

type WeatherMain struct {
	Temp      float64 `json:"temp"`
	FeelsLike float64 `json:"feels_like"`
	TempMin   float64 `json:"temp_min"`
	TempMax   float64 `json:"temp_max"`
	Pressure  float64 `json:"pressure"`
	Humidity  float64 `json:"humidity"`
	SeaLevel  float64 `json:"sea_level"`
	GrndLevel float64 `json:"grnd_level"`
}

type WeatherWind struct {
	Speed float64 `json:"speed"`
	Deg   int64   `json:"deg"`
	Gust  float64 `json:"gust"`
}

type WeatherCloud struct {
	All int64 `json:"all"`
}

type WeatherSys struct {
	Type    int64  `json:"type"`
	Id      int64  `json:"id"`
	Country string `json:"country"`
	Sunrise int64  `json:"sunrise"`
	Sunset  int64  `json:"sunset"`
}
