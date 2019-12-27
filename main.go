package  main

import (
	"github.com/dongnguyenltqb/go-rabbit/consumer"
	"github.com/dongnguyenltqb/go-rabbit/database"
	"github.com/dongnguyenltqb/go-rabbit/publisher"
	mystorage "github.com/dongnguyenltqb/go-rabbit/storage"
	"github.com/dongnguyenltqb/go-rabbit/user"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"log"
	"os"
)

func main(){
	app := gin.Default()
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	database.Init()
	user.RegistryUserController(app)
	mystorage.RegisterStorageController(app)
	publisher.JoinNetWork()
	go consumer.InitResizeConsumer()
	go consumer.InitUploadConsumer()
	go mystorage.LoadBucket()
	app.Run(":"+os.Getenv("PORT"))
}