package consumer

import (
	"encoding/json"
	mystorage "github.com/dongnguyenltqb/go-rabbit/storage"
	"github.com/dongnguyenltqb/go-rabbit/utils"
	"github.com/streadway/amqp"
	"os"
)

var C2 *amqp.Channel
var tasks_upload <- chan amqp.Delivery

func uploadWorker(pool <- chan amqp.Delivery){
	for{
		task := <- pool
		var object mystorage.ObjectAddress
		if err := json.Unmarshal(task.Body,&object); err != nil{
			task.Reject(true)
			utils.Logger("error",err)
			continue
		}
		finished := make(chan bool)
		go mystorage.UploadToGCloudStorage(object.FileName,finished)
		if result := <- finished ; result == true {
			task.Ack(false)
		} else {
			task.Reject(true)
		}
	}
}

func runUploadWorker(numWorker int){
	for i:=1;i<numWorker;i++{
		go uploadWorker(tasks_upload)
	}
}  

func InitUploadConsumer(){
	conn,err := amqp.Dial(os.Getenv("AMQP_URL"))
	C2,err = conn.Channel()
	if err!=nil {
		panic(err)
	}
	C2.QueueDeclare("UploadImageToGCloud",true,false,false,false,nil)
	C2.QueueBind("UploadImageToGCloud","","UploadImage",true, map[string]interface{}{
		"type":"image/jpeg",
		"job":"upload",
	})
	tasks_upload,_ = C2.Consume("UploadImageToGCloud","UploadImage",false,true,false,false,nil)
	go runUploadWorker(NumWorker)
}