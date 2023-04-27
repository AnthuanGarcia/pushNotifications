package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"google.golang.org/api/option"
)

type Ambient struct {
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
	HeatIndex   float64 `json:"heatIndex"`
	Movement    int     `json:"move"`
}

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

func main() {

	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/sendAll", func(w http.ResponseWriter, r *http.Request) {

		decoder := json.NewDecoder(r.Body)
		ambient := &Ambient{}

		if err := decoder.Decode(ambient); err != nil {

			log.Println("Error:", err)
			w.WriteHeader(http.StatusBadRequest)
			return

		}

		// TODO: Get deviceTokens
		if err := sendPushNotification(nil, *ambient); err != nil {

			log.Println("Error:", err)
			w.WriteHeader(http.StatusBadRequest)
			return

		}

		w.WriteHeader(http.StatusOK)

	})

	log.Fatal(
		http.ListenAndServe(fmt.Sprintf(":%s", port), nil),
	)

}
