package models

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	_ "github.com/lib/pq"
)

type Storage interface {
	StoreIndegoData(*IndegoRes) error
	GetStationList(string) ([]BikeResult, error)
	GetStation(string, string) (BikeResult, error)
}

type PostgresStore struct {
	Db *sql.DB
}

func NewPostgresStore() (*PostgresStore, error) {
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	connStr := fmt.Sprintf("user=%v dbname=%v password=%v sslmode=disable", dbUser, dbName, dbPassword)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &PostgresStore{
		Db: db,
	}, nil
}

// table migration
func (s *PostgresStore) Init() error {
	if err := s.createStationTable(); err != nil {
		log.Printf("Station table err")
		return err
	}

	if err := s.createBikeTable(); err != nil {
		log.Printf("Station bike err")
		return err
	}

	return nil
}

func (s *PostgresStore) createStationTable() error {
	query := `CREATE TABLE IF NOT EXISTS stations (
		uid SERIAL PRIMARY KEY,
		id INT NOT NULL,
		name VARCHAR(255),
		latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    geometry_type VARCHAR(255),
		total_docks INT,
		docks_available INT,
		bikes_available INT,
    classic_bikes_available INT,
    smart_bikes_available INT,
    electric_bikes_available INT,
    reward_bikes_available INT,
    reward_docks_available INT,
    kiosk_status VARCHAR(255),
    kiosk_public_status VARCHAR(255),
    kiosk_connection_status VARCHAR(255),
    kiosk_type INT,
    address_street VARCHAR(255),
    address_city VARCHAR(255),
    address_state VARCHAR(255),
    address_zip_code VARCHAR(255),
    open_time VARCHAR(255),
    close_time VARCHAR(255),
    event_start VARCHAR(255),
    event_end VARCHAR(255),
    is_event_based BOOLEAN,
    is_virtual BOOLEAN,
    kiosk_id INT,
    notes TEXT,
    public_text TEXT,
    time_zone VARCHAR(255),
		trikes_available INT,
    station_type VARCHAR(255),
		updated_at TIMESTAMP,
		created_at TIMESTAMP
	)`

	_, err := s.Db.Exec(query)
	if err != nil {
		return err
	}

	_, err = s.Db.Exec(`CREATE INDEX IF NOT EXISTS idx_updated_at ON stations (updated_at)`)
	if err != nil {
		return err
	}

	_, err = s.Db.Exec(`CREATE INDEX IF NOT EXISTS idx_kiosk_id ON stations (kiosk_id)`)
	if err != nil {
		return err
	}

	return nil
}

func (s *PostgresStore) createBikeTable() error {
	query := `CREATE TABLE IF NOT EXISTS bikes (
		uid SERIAL PRIMARY KEY,
		station_id INT,
    dock_number INT,
    is_electric BOOLEAN,
    is_available BOOLEAN,
    battery INT,
		created_at TIMESTAMP,
		FOREIGN KEY (station_id) REFERENCES stations(uid)
	)`

	_, err := s.Db.Exec(query)
	return err
}

// end table migration

func (s *PostgresStore) StoreIndegoData(data *IndegoRes) error {
	tx, err := s.Db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}

	LastUpdated := data.LastUpdated

	query := `INSERT INTO stations
		(id, name, latitude, longitude, geometry_type, total_docks, docks_available, bikes_available, classic_bikes_available, smart_bikes_available, electric_bikes_available, reward_bikes_available, reward_docks_available, kiosk_status, kiosk_public_status, kiosk_connection_status, kiosk_type, address_street, address_city, address_state, address_zip_code, open_time, close_time, event_start, event_end, is_event_based, is_virtual, kiosk_id, notes, public_text, time_zone, trikes_available, station_type,  updated_at, created_at) VALUES `
	var values []interface{}

	for i, v := range data.Features {
		p := v.Properties
		g := v.Geometry
		station_type := v.Type
		coords := p.Coordinates
		if len(coords) != 2 {
			return fmt.Errorf("invalid coordinate")
		}
		lat := coords[1]
		lng := coords[0]
		qi := i * 35

		if i > 0 {
			query += ", "
		}

		query += fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)", qi+1, qi+2, qi+3, qi+4, qi+5, qi+6, qi+7, qi+8, qi+9, qi+10, qi+11, qi+12, qi+13, qi+14, qi+15, qi+16, qi+17, qi+18, qi+19, qi+20, qi+21, qi+22, qi+23, qi+24, qi+25, qi+26, qi+27, qi+28, qi+29, qi+30, qi+31, qi+32, qi+33, qi+34, qi+35)

		values = append(values, p.Id, p.Name, lat, lng, g.Type, p.TotalDocks, p.DocksAvailable, p.BikesAvailable, p.ClassicBikesAvailable, p.SmartBikesAvailable, p.ElectricBikesAvailable, p.RewardBikesAvailable, p.RewardDocksAvailable, p.KioskStatus, p.KioskPublicStatus, p.KioskConnectionStatus, p.KioskType, p.AddressStreet, p.AddressCity, p.AddressState, p.AddressZipCode, p.OpenTime, p.CloseTime, p.EventStart, p.EventEnd, p.IsEventBased, p.IsVirtual, p.KioskId, p.Notes, p.PublicText, p.TimeZone, p.TrikesAvailable, station_type, LastUpdated, time.Now())
	}

	_, err = tx.Exec(query, values...)
	if err != nil {
		tx.Rollback()
		log.Printf("Station Query execute error %v", err)
		return fmt.Errorf("failed to execute station bulk insert: %v", err)
	}

	// select for insert bikes
	sQuery := "SELECT uid, kiosk_id, updated_at FROM stations WHERE updated_at = $1"
	rows, err := tx.Query(sQuery, LastUpdated)

	if err != nil {
		tx.Rollback()
		log.Printf("Query select error %v", err)
		return fmt.Errorf("failed to select query: %v", err)
	}
	defer rows.Close()

	var bikes []DbBike
	for rows.Next() {
		mp := MinProperties{}
		if err := rows.Scan(&mp.Uid, &mp.KioskId, &mp.UpdatedAt); err != nil {
			tx.Rollback()
			log.Printf("Station scan error %v", err)
			return fmt.Errorf("failed to scan row: %v", err)
		}

		for _, v := range data.Features {
			p := v.Properties
			if p.KioskId == mp.KioskId && len(p.Bikes) > 0 {
				for _, b := range p.Bikes {
					bikes = append(bikes, DbBike{StationId: mp.Uid, Bike: b})
				}
				break
			}
		}
	}

	bQuery := `INSERT INTO bikes (station_id, dock_number, is_electric, is_available, battery, created_at) VALUES `
	var bValues []interface{}
	for i, b := range bikes {
		if i > 0 {
			bQuery += ", "
		}

		bi := i * 6
		bQuery += fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d)", bi+1, bi+2, bi+3, bi+4, bi+5, bi+6)
		bValues = append(bValues, b.StationId, b.DockNumber, b.IsElectric, b.IsAvailable, b.Battery, time.Now())
	}

	_, err = tx.Exec(bQuery, bValues...)
	if err != nil {
		tx.Rollback()
		log.Printf("Bike Query execute error %v", err)
		return fmt.Errorf("failed to execute bike bulk insert: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		log.Printf("Query transaction commit error %v", err)
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	return nil
}

func (s *PostgresStore) GetStationList(at string) (res []BikeResult, err error) {
	atFrom := fmt.Sprintf("%v.000", at)
	atTo := fmt.Sprintf("%v.999", at)
	query := "SELECT st.*, b.dock_number, b.is_electric, b.is_available, b.battery FROM stations as st LEFT JOIN bikes b ON st.uid = b.station_id WHERE st.updated_at >= $1 AND st.updated_at <= $2 limit 100"
	rows, err := s.Db.Query(query, atFrom, atTo)

	if err != nil {
		log.Printf("Query select error %v", err)
		return nil, fmt.Errorf("failed to select query: %v", err)
	}
	defer rows.Close()

	fMap := make(map[int64]*Feature)

	var rowCount int = 0
	for rows.Next() {
		rowCount++
		p := DbProperties{}
		b := Bike{}
		var uid int64
		if err := rows.Scan(&uid, &p.Id, &p.Name, &p.Latitude, &p.Longitude, &p.GeometryType, &p.TotalDocks, &p.DocksAvailable, &p.BikesAvailable, &p.ClassicBikesAvailable, &p.SmartBikesAvailable, &p.ElectricBikesAvailable, &p.RewardBikesAvailable, &p.RewardDocksAvailable, &p.KioskStatus, &p.KioskPublicStatus, &p.KioskConnectionStatus, &p.KioskType, &p.AddressStreet, &p.AddressCity, &p.AddressState, &p.AddressZipCode, &p.OpenTime, &p.CloseTime, &p.EventStart, &p.EventEnd, &p.IsEventBased, &p.IsVirtual, &p.KioskId, &p.Notes, &p.PublicText, &p.TimeZone, &p.TrikesAvailable, &p.StationType, &p.UpdatedAt, &p.CreatedAt, &b.DockNumber, &b.IsElectric, &b.IsAvailable, &b.Battery); err != nil {
			log.Printf("Station scan error %v", err)
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}

		f := Feature{}
		if _, exists := fMap[uid]; !exists {
			err := s.ConvertDBProperties(p, &f)
			if err != nil {
				log.Printf("Convert properties error %v", err)
				return nil, fmt.Errorf("failed to convert properties: %v", err)
			}

			fMap[uid] = &f
		}

		if b.DockNumber != nil || b.IsElectric != nil || b.IsAvailable != nil || b.Battery != nil {
			fMap[uid].Properties.Bikes = append(fMap[uid].Properties.Bikes, b)
		}
	}

	if rowCount == 0 {
		return nil, nil
	}

	var features []*Feature
	for _, f := range fMap {
		features = append(features, f)
	}

	sort.Slice(features, func(i, j int) bool {
		return features[i].Properties.Id < features[j].Properties.Id
	})

	for _, v := range features {
		bikeResult := BikeResult{}
		bikeResult.At = at
		bikeResult.Stations = *v

		res = append(res, bikeResult)
	}

	return
}

func (s *PostgresStore) GetStation(at string, kioskId string) (res BikeResult, err error) {
	atFrom := fmt.Sprintf("%v.000", at)
	atTo := fmt.Sprintf("%v.999", at)
	query := "SELECT st.*, b.dock_number, b.is_electric, b.is_available, b.battery FROM stations as st LEFT JOIN bikes b ON st.uid = b.station_id WHERE st.kiosk_id = $1 AND st.updated_at >= $2 AND st.updated_at <= $3"
	rows, err := s.Db.Query(query, kioskId, atFrom, atTo)

	if err != nil {
		log.Printf("Query select error %v", err)
		return BikeResult{}, fmt.Errorf("failed to select query: %v", err)
	}
	defer rows.Close()

	var rowCount int = 0
	f := Feature{}
	for rows.Next() {
		rowCount++
		p := DbProperties{}
		b := Bike{}
		if err := rows.Scan(&p.Uid, &p.Id, &p.Name, &p.Latitude, &p.Longitude, &p.GeometryType, &p.TotalDocks, &p.DocksAvailable, &p.BikesAvailable, &p.ClassicBikesAvailable, &p.SmartBikesAvailable, &p.ElectricBikesAvailable, &p.RewardBikesAvailable, &p.RewardDocksAvailable, &p.KioskStatus, &p.KioskPublicStatus, &p.KioskConnectionStatus, &p.KioskType, &p.AddressStreet, &p.AddressCity, &p.AddressState, &p.AddressZipCode, &p.OpenTime, &p.CloseTime, &p.EventStart, &p.EventEnd, &p.IsEventBased, &p.IsVirtual, &p.KioskId, &p.Notes, &p.PublicText, &p.TimeZone, &p.TrikesAvailable, &p.StationType, &p.UpdatedAt, &p.CreatedAt, &b.DockNumber, &b.IsElectric, &b.IsAvailable, &b.Battery); err != nil {
			log.Printf("Station scan error %v", err)
			return BikeResult{}, fmt.Errorf("failed to scan row: %v", err)
		}

		if rowCount == 1 {
			// only update when first time
			err = s.ConvertDBProperties(p, &f)
			if err != nil {
				log.Printf("Convert properties error %v", err)
				return BikeResult{}, fmt.Errorf("failed to convert properties: %v", err)
			}
		}

		if b.DockNumber != nil || b.IsElectric != nil || b.IsAvailable != nil || b.Battery != nil {
			f.Properties.Bikes = append(f.Properties.Bikes, b)
		}
	}

	if rowCount == 0 {
		return BikeResult{}, fmt.Errorf("empty row")
	}

	res.At = at
	res.Stations = f

	return
}

func (s *PostgresStore) ConvertDBProperties(dbp DbProperties, f *Feature) (err error) {
	p := Properties{}
	p.Id = dbp.Id
	p.Name = dbp.Name
	p.Coordinates = dbp.Coordinates
	p.TotalDocks = dbp.TotalDocks
	p.DocksAvailable = dbp.DocksAvailable
	p.BikesAvailable = dbp.BikesAvailable
	p.ClassicBikesAvailable = dbp.ClassicBikesAvailable
	p.SmartBikesAvailable = dbp.SmartBikesAvailable
	p.ElectricBikesAvailable = dbp.ElectricBikesAvailable
	p.RewardBikesAvailable = dbp.RewardBikesAvailable
	p.RewardDocksAvailable = dbp.RewardDocksAvailable
	p.KioskStatus = dbp.KioskStatus
	p.KioskPublicStatus = dbp.KioskPublicStatus
	p.KioskConnectionStatus = dbp.KioskConnectionStatus
	p.KioskType = dbp.KioskType
	p.AddressStreet = dbp.AddressStreet
	p.AddressCity = dbp.AddressCity
	p.AddressState = dbp.AddressState
	p.AddressZipCode = dbp.AddressZipCode
	p.Bikes = dbp.Bikes
	p.CloseTime = dbp.CloseTime
	p.EventEnd = dbp.EventEnd
	p.EventStart = dbp.EventStart
	p.IsEventBased = dbp.IsEventBased
	p.IsVirtual = dbp.IsVirtual
	p.KioskId = dbp.KioskId
	p.Notes = dbp.Notes
	p.OpenTime = dbp.OpenTime
	p.PublicText = dbp.PublicText
	p.TrikesAvailable = dbp.TrikesAvailable
	p.Latitude = dbp.Latitude
	p.Longitude = dbp.Longitude

	g := Geometry{}
	g.Coordinates = []float64{p.Longitude, p.Latitude}
	g.Type = dbp.GeometryType

	f.Geometry = g
	f.Properties = p
	f.Type = dbp.StationType

	return
}
