package main

import (
	"context"
	"time"

	"github.com/moooll/cat-service-mongo/internal/handler"
	"github.com/moooll/cat-service-mongo/internal/repository"
	rediscache "github.com/moooll/cat-service-mongo/internal/repository/rediscache"

	"log"

	"github.com/go-redis/cache/v8"
	"github.com/go-redis/redis/v8"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)

	defer cancel()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(repository.DatabaseURI))
	if err != nil {
		log.Print("could not connect to the db\n", err.Error())
	}

	defer func() {
		err = mongoClient.Disconnect(ctx)
		if err != nil {
			log.Print("could not disconnect from the db\n", err.Error())
		}
	}()

	collection := mongoClient.Database("catalog").Collection("cats2")
	dbs, err := mongoClient.ListDatabases(context.Background(), bson.M{})
	if err != nil {
		log.Print("error listing dbs ", err.Error())
	}
	collections, _ := mongoClient.Database("catalog").ListCollectionNames(context.Background(), bson.M{})
	log.Print(collections)
	log.Print(dbs)
	ring := redis.NewRing(&redis.RingOptions{
		Addrs: map[string]string{
			"server": ":6379",
		},
	})

	redisC := cache.New(&cache.Options{
		Redis:      ring,
		LocalCache: cache.NewTinyLFU(1000, time.Minute),
	})

	service := handler.NewService(repository.NewCatalog(collection), rediscache.NewRedisCache(redisC))
	e := echo.New()
	e.POST("/cats", service.AddCat)
	e.GET("/cats", service.GetAllCats)
	e.GET("/cats/:id", service.GetCat)
	e.PUT("/cats", service.UpdateCat)
	e.DELETE("/cats/:id", service.DeleteCat)
	e.GET("/cats/get-rand-cat", handler.GetRandCat)
	if err := e.Start(":8081"); err != nil {
		log.Print("could not start server\n", err.Error())
	}
}
