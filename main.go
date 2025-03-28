package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-redis/redis"
)

type SomeRepository interface {
	GetData(currency string) string
}

type SomeRepositoryImpl struct{}

func (r *SomeRepositoryImpl) GetData(currency string) string {
	resp, err := http.Get(fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?ids=bitcoin&vs_currencies=%s", currency))
	if err != nil {
		log.Println("CoinGecko request error:", err)
		return "CoinGecko request error"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Println("unexpected status:", resp.StatusCode)
		return "Data retrieval error"
	}

	var res map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		log.Println("Error decoding JSON:", err)
		return "Data retrieval error"
	}

	price := res["bitcoin"][currency]

	fmt.Println("Received from external source:", currency)
	return fmt.Sprintf("Price BTC: %.2f %s", price, strings.ToUpper(currency))
}

type SomeRepositoryProxy struct {
	repository SomeRepository
	cache      *redis.Client
}

func (r *SomeRepositoryProxy) GetData(currency string) string {
	cacheKey := fmt.Sprintf("bitcoin:%s", currency)

	res, err := r.cache.Get(cacheKey).Result()
	if err == redis.Nil {
		log.Println("Data not in cache:", err)
		data := r.repository.GetData(currency)

		err := r.cache.Set(cacheKey, data, 30*time.Second).Err()
		if err != nil {
			log.Println("Error saving data to cache:", err)
			return "Error saving data to redis"
		}
		log.Println("The query result is saved in cache")

		return data
	} else if err != nil {
		log.Println("Error getting data from cache:", err)
		return "Error getting data from redis"
	}

	return res
}

func main() {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	repo := &SomeRepositoryImpl{}
	proxy := &SomeRepositoryProxy{
		repository: repo,
		cache:      client,
	}

	fmt.Println(proxy.GetData("usd"))
	fmt.Println(proxy.GetData("usd"))
	fmt.Println(proxy.GetData("eur"))
	fmt.Println(proxy.GetData("eur"))
}
