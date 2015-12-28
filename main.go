package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"

	_ "github.com/lib/pq"
)

var stmt *sql.Stmt

// GeoJSON Feature Collection's pieces
type FeatureCollection struct {
	Features []Feature `json:"features"`
	Type     string    `json:"type"`
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
	Bathrooms int    `json:"bathrooms"`
	Bedrooms  int    `json:"bedrooms"`
	ID        string `json:"id"`
	Price     int    `json:"price"`
	SqFt      int    `json:"sq_ft"`
	Street    string `json:"street"`
}

func main() {
	port := flag.Int("port", 3000, "Port to run the server on")
	databaseString := flag.String("database", "",
		"Postgresql database connection string")

	flag.Parse()

	db, err := sql.Open("postgres", *databaseString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if setStmt, err := db.Prepare(
		`SELECT id, street, price, bedrooms, bathrooms, sq_ft, lat, lng
		FROM listings WHERE price BETWEEN $1 AND $2 AND bedrooms BETWEEN
		$3 AND $4 AND bathrooms BETWEEN $5 AND $6`); err != nil {
		log.Fatal(err)
	} else {
		// set global var stmt, TODO replace with handler context
		stmt = setStmt
	}
	defer stmt.Close()

	http.HandleFunc("/listings", listingsHandler)

	log.Printf("Starting server on port %d\n", *port)
	err = http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func listingsHandler(w http.ResponseWriter, r *http.Request) {
	vals := map[string]int{"min_price": 0, "max_price": 300000, "min_bed": 0,
		"max_bed": 5, "min_bath": 0, "max_bath": 3}

	// overwrite the above default values with any provided parameters
	for key := range vals {
		if fval := r.FormValue(key); fval != "" {
			if val, err := strconv.Atoi(fval); err != nil &&
				err.(*strconv.NumError).Err == strconv.ErrSyntax {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, "%s is not a number", key)
				return
			} else {
				vals[key] = val
			}
		}
	}

	featColl, err := getFeatureCollection(stmt, vals["min_price"], vals["max_price"],
		vals["min_bed"], vals["max_bed"], vals["min_bath"], vals["max_bath"])
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "We encountered a problem:\n", err)
		return
	}

	geoJson, err := json.Marshal(featColl)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "We encountered a problem:\n", err)
		return
	}

	// optionally set header to regular json, not GeoJSON
	if _, ok := r.Form["json"]; ok {
		w.Header().Set("Content-Type", "application/json")
	} else {
		w.Header().Set("Content-Type", "application/vnd.geo+json")
	}

	w.Write(geoJson)
}

// obtain a feature collection using the inclusive min and max options
func getFeatureCollection(stmt *sql.Stmt, minPrice, maxPrice, minBed, maxBed,
	minBath, maxBath interface{}) (fc FeatureCollection, e error) {

	rows, err := stmt.Query(minPrice, maxPrice, minBed, maxBed, minBath, maxBath)
	if e = err; err != nil {
		return
	}
	defer rows.Close()

	features := make([]Feature, 0)
	for rows.Next() {
		var (
			id, street               string
			price, beds, baths, sqft int
			lat, lng                 float64
		)
		err := rows.Scan(&id, &street, &price, &beds, &baths, &sqft, &lat, &lng)
		if e = err; err != nil {
			return
		}

		properties := Properties{Bathrooms: baths, Bedrooms: beds, ID: id,
			Price: price, SqFt: sqft, Street: street}
		geometry := Geometry{Coordinates: []float64{lng, lat}, Type: "Point"}
		features = append(features, Feature{Geometry: geometry,
			Properties: properties, Type: "Feature"})
	}

	fc.Type = "FeatureCollection"
	fc.Features = features
	return
}
