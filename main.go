package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/api/option"
)

type Ambient struct {
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
	HeatIndex   float64 `json:"heatIndex"`
	Movement    int     `json:"move"`
}

const DBFile = "./tokens.db"
const Table = `CREATE TABLE IF NOT EXISTS Tokens(token VARCHAR(64) NOT NULL,time DATETIME NOT NULL);`

func sendPushNotification(deviceTokens []string, data Ambient) (err error) {

	authKey := []byte(os.Getenv("FIREBASE_AUTH_KEY"))

	opts := []option.ClientOption{option.WithCredentialsJSON(authKey)}

	ctx := context.Background()
	app, err := firebase.NewApp(ctx, nil, opts...)

	if err != nil {
		return
	}

	fcmClient, err := app.Messaging(ctx)

	if err != nil {
		return
	}

	_, err = fcmClient.SendMulticast(ctx, &messaging.MulticastMessage{
		Notification: &messaging.Notification{
			Title: "Hola", Body: "Prueba",
		},
		Tokens: deviceTokens,
	})

	if err != nil {
		return
	}

	return nil

}

func sendAll(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid Method"))
		return
	}

	decoder := json.NewDecoder(r.Body)
	ambient := &Ambient{}

	if err := decoder.Decode(ambient); err != nil {

		log.Println("Error:", err)
		w.WriteHeader(http.StatusBadRequest)
		return

	}

	db, err := sql.Open("sqlite3", DBFile)

	if err != nil {
		log.Println("Error:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	rows, err := db.Query("SELECT token FROM Tokens;")

	if err != nil {
		log.Println("Error:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	deviceTokens := []string{}

	for rows.Next() {

		token := ""
		err = rows.Scan(&token)

		if err != nil {
			log.Println("Error:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		deviceTokens = append(deviceTokens, token)

	}

	if err := sendPushNotification(deviceTokens, *ambient); err != nil {

		log.Println("Error:", err)
		w.WriteHeader(http.StatusBadRequest)
		return

	}

}

func registerToken(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid Method"))
		return
	}

	token := r.URL.Query().Get("token")

	if token == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Token not proportinated"))
		return
	}

	db, err := sql.Open("sqlite3", DBFile)

	if err != nil {
		log.Println("Error(Open):", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if _, err = db.Exec(Table); err != nil {
		log.Println("Error(Create Table):", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = db.Exec(
		"INSERT INTO Tokens VALUES(?, ?);",
		token, time.Now().Format(time.RFC3339),
	)

	if err != nil {
		log.Println("Error(Exec):", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

}

func main() {

	port := os.Getenv("PORT")

	if port == "" {
		port = "8000"
	}

	http.HandleFunc("/sendAll", sendAll)
	http.HandleFunc("/registerToken", registerToken)

	fmt.Printf("Running in %s...\n", port)

	log.Fatal(
		http.ListenAndServe(fmt.Sprintf(":%s", port), nil),
	)

}
