package consumer

import (
	"encoding/json"
	"fmt"
	"github.com/disintegration/imaging"
	mystorage "github.com/dongnguyenltqb/go-rabbit/storage"
	"github.com/dongnguyenltqb/go-rabbit/utils"
	"github.com/streadway/amqp"
	"os"
	"strings"
)

var C *amqp.Channel
var ResizeTaskPool chan mystorage.ObjectAddress

func ResizeWorker(pool <-chan amqp.Delivery){
	for{
		task := <- pool
		fmt.Println(utils.ApplyStyle("bold","yellow","Resizing image"))
		var object mystorage.ObjectAddress
		if err := json.Unmarshal(task.Body,&object);err != nil {
			task.Reject(true)
			utils.Logger("error",err)
			continue
		}
		srcImage, err := imaging.Open("storage/image/"+object.FileName)
		if err != nil {
			task.Reject(true)
			utils.Logger("error",err)
			continue
		}
		dstImage200 := imaging.Resize(srcImage, 200, 0, imaging.Lanczos)
		resizedFileName := "200@"+strings.TrimLeft(object.FileName,"/image/")
		err = imaging.Save(dstImage200,"storage/image/"+resizedFileName)
		if err != nil {
			task.Reject(true)
			utils.Logger("error",err)
			continue
		}
		task.Ack(false)
		workerMessage := mystorage.WorkerMessage{
			FileName: resizedFileName,
			Resize:   false,
		}
		go func(workerMessage mystorage.WorkerMessage) {
				mystorage.UploadPool <- workerMessage
		}(workerMessage)
	}
}

func runResizeWorker(numWorker int){
	for i:=0;i<=numWorker;i++{
		go ResizeWorker(tasks)
	}
}

var tasks <- chan amqp.Delivery

func InitConsumer(){

	conn,err := amqp.Dial(os.Getenv("AMQP_URL"))
	C,err = conn.Channel()
	if err!=nil {
		panic(err)
	}
	C.QueueDeclare("ResizeImage",true,false,false,false,nil)
	C.QueueBind("ResizeImage","","ProcessImage",true, map[string]interface{}{
		"type":"image/jpeg",
	})
	tasks,_ = C.Consume("ResizeImage","ResizeImageWorker",false,true,false,false,nil)
	go runResizeWorker(100)
}