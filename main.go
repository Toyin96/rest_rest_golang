package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
)

var database *sql.DB

type City struct {
	ID  int `json:"id"`
	Name string `json:"name"`
	CountryCode string `json:"country_code"`
	District string `json:"district"`
	Population int `json:"population"`
}

type UpdateDB struct {
	NumOfRowsAffected int64
	ID int64
}

func DbConnect(){
	db, err := sql.Open("mysql", "toyin96:m18job,,@tcp(127.0.0.1:3306)/world")
	if err != nil{
		log.Fatalln("error connecting to the database: ", err)
	}

	fmt.Println("successfully connected to the database")
	//defer db.Close()
	database = db
}

func Homepage(w http.ResponseWriter, r *http.Request){

	type User struct {
		Name string `json:"name"`
		Time time.Time `json:"time"`
	}

	user1 := User{
		Name: "Toyin",
		Time: time.Now(),
	}
	t, err := template.ParseFiles("templates/home.html")
	if err != nil{
		log.Fatalln("error parsing the home template: ", err)
	}
	err = t.Execute(w, user1)
	if err != nil{
		log.Panicln("error executing the template with the data: ", err)
	}
}

func CityList(w http.ResponseWriter, r *http.Request){

	jsonCities := dbCityList()

	//sending http response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonCities)
}

func dbCityList()[] byte{
	cities := []City{}
	city := City{}
	//querying the database
	result, err := database.Query("SELECT * FROM CITY")
	if err != nil{
		log.Fatalln("error querying the database for dbCityList: ", err)
	}
	defer database.Close()

	for result.Next(){
		err := result.Scan(&city.ID, &city.Name, &city.CountryCode, &city.District, &city.Population)
		if err != nil{
			log.Fatalln("error parsing files from dbCityList: ", err)
		}

		cities = append(cities, city)
	}

	jsonCities, err := json.Marshal(cities)
	if err != nil{
		log.Fatalln("error converting citylist struct into json: ", err)
	}
	return jsonCities
}

func CityInfo(w http.ResponseWriter, r *http.Request){
	vars := mux.Vars(r)
	cityId, err := strconv.Atoi(vars["id"])
	if err != nil{
		log.Panicln("error parsing cityID: ", err)
	}
	jsonCity := dbCityInfo(cityId)

	//sending a response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonCity)
}

func dbCityInfo(id int) []byte{
	city := City{}
	err := database.QueryRow("SELECT id, name, countrycode, district, population from CITY where ID=?", id).Scan(&city.ID, &city.Name, &city.CountryCode, &city.District, &city.Population)
	if err != nil {
		log.Fatalln("could not fetch the particular city: ", err)
	}

	jsonCity, err := json.Marshal(city)
	if err != nil{
		log.Panicln("error converting city to json: ", err)
	}
	return jsonCity
}

func DeleteCity(w http.ResponseWriter, r *http.Request){
	vars := mux.Vars(r)
	cityId, err := strconv.Atoi(vars["id"])
	if err != nil{
		log.Panicln("error parsing cityID: ", err)
	}

	deletedCity := dbDeleteCity(cityId)

	//returning the response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(deletedCity)
}

func dbDeleteCity(id int)[]byte{
	deletedCity := UpdateDB{}
	//preparing to delete city from db

	Stmt, err := database.Prepare("DELETE from city where id=?")
	if err != nil{
		log.Panicln("couldn't prepare statement to delete city: ", err)
	}

	res, err := Stmt.Exec(id)
	if err != nil{
		log.Panicln("couldn't delete city: ", err)
	}

	deletedCity.NumOfRowsAffected, err = res.RowsAffected()
	deletedCity.ID, err = res.LastInsertId()

	delCityRes, err := json.Marshal(deletedCity)
	if err != nil{
		log.Panicln("couldn't convert updated deleted city struct: ", err)
	}

	return delCityRes
}

func CityAdd(w http.ResponseWriter, r *http.Request){
	var city City

	requestBody, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		r.Body.Close()
		log.Fatalln("couldn't read from request body: ", err)
	}

	// convert the json in the request a to Go's type using the json.unmarshall method

	if err := json.Unmarshal(requestBody, &city); err != nil{
		w.Header().Set("Content-Type", "application.json")
		w.WriteHeader(422) //couldn't process

		if err := json.NewEncoder(w).Encode(err); err !=nil{
			log.Panicln(err)
		}
	}

	// call the function that'll add the city to our database

	jsonCity := dbCityAdd(city)

	// send the response to the client

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonCity)
}

func dbCityAdd(city City)[]byte{
	var update UpdateDB

	/* first, we'll prepare the city to protect against
	sql injection before executing the insert command
	 */
	stmt, err := database.Prepare("INSERT INTO CITY(name, countrycode, district, population) VALUES(?, ?, ?, ?)")

	if err != nil{
		log.Panicln(err)
	}

	body, err := stmt.Exec(city.Name, city.CountryCode, city.District, city.Population)
	if err != nil{
		log.Panicln("error inserting city into database: ", err)
	}

	insertId, err := body.LastInsertId()
	if err != nil{
		log.Fatalln(err)
	}

	rowCnt, err := body.RowsAffected()
	if err != nil{
		log.Fatalln(err)
	}

	update.ID = insertId
	update.NumOfRowsAffected = rowCnt

	updateJson, err := json.Marshal(update)
	if err != nil{
		log.Fatalln("couldn't convert update to json format: ", err)
	}

	return updateJson
}


func main(){
	router := mux.NewRouter()

	DbConnect()

	router.HandleFunc("/", Homepage)
	router.HandleFunc("/city", CityList)
	router.HandleFunc("/cityadd", CityAdd)
	router.HandleFunc("/city/{id}", CityInfo)
	router.HandleFunc("/citydel/{id}", DeleteCity)

	log.Fatalln(http.ListenAndServe(":8999", router))
}
