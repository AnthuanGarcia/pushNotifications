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
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type Ambient struct {
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
	HeatIndex   float64 `json:"heatIndex"`
	Movement    int     `json:"move"`
}

var app *firebase.App

func sendPushNotification(ambient Ambient) (err error) {

	credentials := os.Getenv("FILENAME_CREDENTIALS")
	opts := []option.ClientOption{option.WithCredentialsFile(credentials)}

	ctx := context.Background()

	if app == nil {

		app, err = firebase.NewApp(ctx, nil, opts...)

		if err != nil {
			return
		}

	}

	fcmClient, err := app.Messaging(ctx)

	if err != nil {
		return
	}

	dbClient, err := app.Firestore(ctx)

	if err != nil {
		return
	}

	data := make(map[string]string)

	data["Title"] = "Alerta de Ambiente"
	data["Body"] = fmt.Sprintf(
		"Temperatura: %.2f°C<br>Humedad: %.0f%%<br>Indice de Calor: %.2f°C",
		ambient.Temperature,
		ambient.Humidity,
		ambient.HeatIndex,
	)
	data["Temp"] = ""

	if ambient.Movement > 0 {

		data["Title"] = "¡Alguien ha entrado al site!"
		data["Body"] = "Se han detectado lecturas de movimiento."
		data["Move"] = ""
		delete(data, "Temp")

	}

	deviceTokens := []string{}
	tokens := dbClient.Collection("tokens").Documents(ctx)

	for {

		token, err := tokens.Next()

		if err == iterator.Done {
			break
		}

		if err != nil {
			return err
		}

		deviceTokens = append(deviceTokens, token.Data()["token"].(string))

	}

	_, err = fcmClient.SendMulticast(ctx, &messaging.MulticastMessage{
		Data:    data,
		Tokens:  deviceTokens,
		Android: &messaging.AndroidConfig{Priority: "high"},
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

	if err := sendPushNotification(*ambient); err != nil {

		log.Println("Error:", err)
		w.WriteHeader(http.StatusBadRequest)
		return

	}

}

func main() {

	port := os.Getenv("PORT")

	if port == "" {
		port = "8000"
	}

	http.HandleFunc("/sendAll", sendAll)

	fmt.Printf("Running in %s...\n", port)

	log.Fatal(
		http.ListenAndServe(fmt.Sprintf(":%s", port), nil),
	)

}
