package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func encryptData(data string) *string {
	sum := sha256.Sum256([]byte(data)) // we encript the data
	v := hex.EncodeToString(sum[:])
	return &v
}

// i need more comments for do something because i cant die aaa
// get a simple connection
func getConnection() (*sql.DB, error) {

	db, err := sql.Open("sqlite3", "./database/iotcameradata.db")
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return db, nil
}
func exist(user string, ip string, w http.ResponseWriter, sizeChan chan int) {
	q := `SELECT COUNT(*) 
		FROM usercameras 
		WHERE username=?1 OR ip =?2 ;`
	var size int
	// GET A CONNECTION
	db, _ := getConnection()

	HowMany, err := db.Query(q, user, ip)
	if err != nil {
		log.Println(err)
		close(sizeChan)
		http.Error(w, "internal server error", 500)
	}
	defer HowMany.Close()

	for HowMany.Next() {
		err = HowMany.Scan(&size)
		if err != nil {
			log.Println(err)
			close(sizeChan)
			http.Error(w, "internal server error", 500)

		}

	}

	sizeChan <- size

}

// register func
func registerUserCameraDatabase(user registerCamera, w http.ResponseWriter, okChan chan bool) {
	if len(user.Username) == 0 || len(user.Password) == 0 {
		close(okChan)
		http.Error(w, "some value is empty", 406)

	}
	sizeChan := make(chan int)
	// we check if the username of the camera already exist
	go exist(user.Username, *encryptData(user.IP), w, sizeChan)
	if <-sizeChan > 0 {
		close(okChan)
		http.Error(w, "sorry but that user has already registered", 406)

	}

	// the query for insert the data
	q := `INSERT INTO 
	usercameras(
		ip,
		password,
		username,
		last_time_login
	)
	VALUES(?1,?2,?3,?4) `
	// we get the connection
	db, err := getConnection()
	if err != nil {
		log.Println(err)
		close(okChan)
		http.Error(w, "internal server error", 500) // 500 = internal server error
	}

	defer db.Close()
	//we use stm to avoid attacks

	stm, err := db.Prepare(q)
	if err != nil {
		log.Println(err)
		close(okChan)
		http.Error(w, "internal server error", 500) // 500 = internal server error

	}
	defer stm.Close()
	//then we run the query
	r, err := stm.Exec(
		encryptData(user.IP),
		encryptData(user.Password),
		user.Username,
		time.Now().UnixNano()/int64(time.Hour),
	)
	if err != nil {
		log.Println(err)
		close(okChan)
		http.Error(w, "internal server error", 500)
		return
	}
	// if more than one file is affected we return an error
	i, _ := r.RowsAffected()
	if i != 1 {
		close(okChan)
		log.Printf("idk why a row has been afected lol\n the query was %s \n the ip was %s \n the password was %s \n the username was %s", q,
			*encryptData(user.IP),
			*encryptData(user.Password),
			user.Username,
		)

		return
	}
	okChan <- true

}

// login func
func loginUserCameraDatabase(user registerCamera, w http.ResponseWriter, validChan chan bool) {
	q := `SELECT COUNT(*) FROM usercameras  
	WHERE username = ?1 AND password= ?2;`
	// get the connection
	db, _ := getConnection()

	defer db.Close()
	// make the consult
	// encript the data
	valid, err := db.Query(q, user.Username,
		encryptData(user.Password))
	if err != nil {

		log.Println(err)
		validChan <- false
		http.Error(w, "internal server error", 500)

	}
	// review the results
	var i int
	for valid.Next() {
		// change the value of i
		err = valid.Scan(&i)
		if err != nil {
			log.Println(err)
			http.Error(w, "internal server error", 500)

		}
	}

	validChan <- i > 0

}

// we generate the token
func generateToken(user registerCamera, w http.ResponseWriter, tokenChan chan string) {
	q := `UPDATE usercameras SET token = ?1 WHERE username = ?2;`
	// we get a connection
	db, err := getConnection()
	if err != nil {
		close(tokenChan)
		log.Println(err)
		http.Error(w, "internal error server", 500)
		return
	}

	token := *encryptData(fmt.Sprintf("%s%d", (*encryptData(user.Password) +
		*encryptData(user.Username)),
		rand.Int(),
	))
	defer db.Close()
	stm, _ := db.Prepare(q)
	stm.Exec(token, user.Username)

	tokenChan <- token
	// prepare the database with a stm

}

// we update the last time that he send somethings
func updateUsages(user registerCamera, w http.ResponseWriter) {
	q := `UPDATE usercameras SET  last_time_login = ?1 WHERE username = ?2;`
	db, err := getConnection()
	if err != nil {
		log.Println(err)
		http.Error(w, "internal server error", 500)
		return
	}
	defer db.Close()
	db.Exec(q, time.Now().UnixNano()/int64(time.Hour), user.Username)

}
