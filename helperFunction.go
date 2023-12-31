package main

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/dgrijalva/jwt-go"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/mgo.v2/bson"
)

var client *mongo.Client

func isValidEmail(email string) bool {
	regex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	match, _ := regexp.MatchString(regex, email)
	return match
}
func isValidPhoneNumber(phone string) bool {
	regex := `^\d{10}$`
	match, _ := regexp.MatchString(regex, phone)
	return match
}
func checkMandatoryFields(fields []string, requestBody map[string]interface{}) error {
	for _, field := range fields {
		if val, ok := requestBody[field]; !ok || val == "" {
			return fmt.Errorf("%s is required", field)
		}
	}
	return nil
}
func checkDuplicate(field, value string, w http.ResponseWriter, collection *mongo.Collection) bool {
	filter := bson.M{field: value}
	var existingUser map[string]interface{}
	err := collection.FindOne(context.Background(), filter).Decode(&existingUser)
	if err == nil {
		http.Error(w, field+" is already registered", http.StatusBadRequest)
		return true
	} else if err != mongo.ErrNoDocuments {
		http.Error(w, "Error checking "+field+" availability: "+err.Error(), http.StatusInternalServerError)
		return true
	}
	return false
}

func initDB() {
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	var err error
	client, err = mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		panic(err)
	}

	err = client.Ping(context.Background(), nil)
	if err != nil {
		panic(err)
	}
}

func GenerateJWT(username string) (string, error) {
	var mySigningKey = []byte("secretkey")
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)

	claims["authorized"] = true
	claims["username"] = username
	claims["exp"] = time.Now().Add(time.Minute * 10).Unix()

	tokenString, err := token.SignedString(mySigningKey)

	if err != nil {
		fmt.Printf("something went wrong: %s", err.Error())
		return "", err
	}
	return tokenString, nil
}

func ExtractUsernameFromJWT(tokenString string) (string, error) {
	mySigningKey := []byte("secretkey")
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return mySigningKey, nil
	})

	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		username := claims["username"].(string)
		return username, nil
	} else {
		return "", fmt.Errorf("invalid token")
	}
}
