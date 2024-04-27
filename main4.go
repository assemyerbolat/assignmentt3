package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type Product struct {
	ID          int    json:"id"
	Name        string json:"name"
	Description string json:"description"
	Price       int    json:"price"
}

var (
	ctx         context.Context
	redisClient *redis.Client
	db          *sqlx.DB
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "assem254673"
	dbname   = "assem"
)

func init() {
	ctx = context.Background()

	redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	psqlconn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	var err error
	db, err = sqlx.Connect("postgres", psqlconn)
	if err != nil {
		log.Fatalf("Error connecting to PostgreSQL: %v", err)
	}
}

func getProductByIDHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	productIDStr := params["id"]
	productID, err := strconv.Atoi(productIDStr)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	cachedProduct, err := redisClient.Get(ctx, "product:"+productIDStr).Result()
	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(cachedProduct))
		return
	}

	var product Product
	err = db.Get(&product, "SELECT * FROM products WHERE id = $1", productID)
	if err != nil {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	productJSON, _ := json.Marshal(product)

	redisClient.Set(ctx, "product:"+productIDStr, string(productJSON), 24*time.Hour)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(productJSON)
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/products/{id}", getProductByIDHandler).Methods("GET")

	fmt.Println("Server is running...")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
