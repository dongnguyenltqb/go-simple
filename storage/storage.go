package mystorage

import (
	"cloud.google.com/go/storage"
	"context"
	"encoding/json"
	"fmt"
	"github.com/dongnguyenltqb/go-rabbit/database"
	"github.com/dongnguyenltqb/go-rabbit/publisher"
	"github.com/dongnguyenltqb/go-rabbit/user"
	"github.com/dongnguyenltqb/go-rabbit/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/streadway/amqp"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/api/iterator"
	"io"
	"log"
	"net/url"
	"os"
)

var Client *storage.Client
var Bucket *storage.BucketHandle


var UploadPool chan ObjectAddress

type ObjectAddress struct {
	ID primitive.ObjectID `json:"_id" bson:"_id"`
	FileName string `json:"fileName" bson:"fileName"`
}

func LoadBucket() {
	var err error
	BucketName := os.Getenv("STORAGE_BUCKET_NAME")
	Client, err = storage.NewClient(context.Background())
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	Bucket = Client.Bucket(BucketName)
	q := storage.Query{}
	Objects := Bucket.Objects(context.Background(), &q)
	fmt.Println(utils.ApplyStyle("bold", "yellow", "=======>  STORAGE STATUS <======="))
	for {
		object, err := Objects.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			utils.Logger("error", err)
			return
		}
		fmt.Println(object.Name, object.ContentType)
	}
}

func UploadToGCloudStorage(fileName string, finished chan bool) {
	f, err := os.Open("storage/image/" + fileName)
	if err != nil {
		utils.Logger("error", err)
		return
	}
	defer f.Close()
	object := Bucket.Object(fileName)
	wc := object.NewWriter(context.Background())
	if _, err = io.Copy(wc, f); err != nil {
		utils.Logger("error", err)
		return
	}
	if err := wc.Close(); err != nil {
		utils.Logger("error", err)
		return
	}
	fmt.Println(utils.ApplyStyle("bold", "yellow", "Upload to Cloud successfully...."))
	object.ACL().Set(context.Background(),storage.AllUsers,storage.RoleReader)
	objectAddress := ObjectAddress{
		ID:primitive.NewObjectID(),
		FileName: fileName,
	}
	_,err = database.Models.Object.InsertOne(context.Background(),objectAddress)
	if err != nil {
		utils.Logger("error",err)
	}
	finished <- true
}


func HandleUploadForm(c *gin.Context) {
	form,_ := c.MultipartForm()
	files := form.File["files"]
	var fileNames []string
	for _, file := range files {
		fileName :=fmt.Sprintf("%v",uuid.New()) + file.Filename
		fileNames = append(fileNames,fileName)
		c.SaveUploadedFile(file, "storage/image/"+fileName)
	}
	c.JSON(200, utils.ApiResponse{
		Ok:      true,
		Message: "Uploaded successfully.....",
		Data:    fileNames,
	})
	for _,fileName := range fileNames{
		go func(fileName string) {
			go PushTaskToExchangeUploadImage(ObjectAddress{FileName:fileName})
			go PushTaskToExchangeProcessImage(ObjectAddress{FileName:fileName})
		}(fileName)
	}
}

func GetObject(c *gin.Context) {
	fileName := c.Query("fileName")
	fileName,_ = url.QueryUnescape(fileName)
	object := Bucket.Object(fileName)
	objectAttrs,err := object.Attrs(context.Background())
	if err != nil {
		utils.Logger("error",err)
		return
	}
	c.Redirect(302,objectAttrs.MediaLink)
}

func PushTaskToExchangeProcessImage( object ObjectAddress){
	task,_ := json.Marshal(object)
	message := amqp.Publishing{
		Headers:         map[string]interface{}{
			"type":"image/jpeg",
			"job":"resize",
		},
		Body:	task,
	}
	publisher.C.Publish("ProcessImage","", false,false,message)
}


func PushTaskToExchangeUploadImage(object ObjectAddress){
	task,_ := json.Marshal(object)
	message := amqp.Publishing{
		Headers:map[string]interface{}{
			"type":"image/jpeg",
			"job":"upload",
		},
		Body:task,
	}
	publisher.C.Publish("UploadImage","",false,false,message)
}

func RegisterStorageController(app *gin.Engine) {
	app.POST("/upload", user.IsAuthenticated, HandleUploadForm)
	app.GET("/object",GetObject)
}
