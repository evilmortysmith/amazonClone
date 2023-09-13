package main

import (
	"context"
	"encoding/json"
	"net/http"

	"gopkg.in/mgo.v2/bson"
)

func RemoveFromCart(w http.ResponseWriter, r *http.Request) {

	if r.Method != "DELETE" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusInternalServerError)
		return
	}

	username := r.FormValue("username")
	login_token := r.FormValue("login_token")
	product_id := r.FormValue("product_id")

	expectedKeysToAddToCart := []string{"product_id", "login_token", "username"}

	for key := range r.Form {
		if !contains(expectedKeysToAddToCart, key) {
			http.Error(w, "Unexpected key in form data: "+key, http.StatusBadRequest)
			return
		}
	}

	if username == "" {
		http.Error(w, "Username field is missing", http.StatusBadRequest)
		return
	}
	if login_token == "" {
		http.Error(w, "Login token is missing", http.StatusBadRequest)
		return
	}
	if product_id == "" {
		http.Error(w, "Product id is missing", http.StatusBadRequest)
		return
	}

	collectionUser := client.Database("amazon_db").Collection("users")

	var filter_username bson.M
	var user User

	if username != "" {
		filter_username = bson.M{"username": username}
		error_name := collectionUser.FindOne(context.TODO(), filter_username).Decode(&user)
		if error_name != nil {
			http.Error(w, "Username is not registered", http.StatusBadRequest)
			return
		}
	}
	if login_token != user.LoginToken {
		http.Error(w, "Incorrect login token", http.StatusBadRequest)
		return
	}

	var product CartItem
	var found bool = false

	if len(user.Cart) == 0 {
		http.Error(w, "Cart is empty", http.StatusBadRequest)
		return
	} else {
		for _, item := range user.Cart {
			if item.Product.ID == product_id {
				product = item
				found = true
			}
		}
	}

	if !found {
		http.Error(w, "Product is not in cart", http.StatusBadRequest)
		return
	}

	if product.Quantity == 1 {

		updatedCart := []CartItem{}
		for _, item := range user.Cart {
			if item.Product.ID != product_id {
				updatedCart = append(updatedCart, item)
			}
		}
		user.Cart = updatedCart
	} else {

		for i, item := range user.Cart {
			if item.Product.ID == product_id {
				user.Cart[i].Quantity--
				break
			}
		}
	}

	_, updateErr := collectionUser.UpdateOne(
		context.TODO(),
		filter_username,
		bson.M{"$set": bson.M{"cart": user.Cart}},
	)
	if updateErr != nil {
		http.Error(w, "Error updating user cart", http.StatusInternalServerError)
		return
	}

	userData, err := json.Marshal(user)
	if err != nil {
		http.Error(w, "Error converting user data to JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)
	w.Write(userData)

}