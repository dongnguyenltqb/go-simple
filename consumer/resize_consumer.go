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

var C1 *amqp.Channel
var NumWorker int

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
		resizedObject := mystorage.ObjectAddress{
			FileName: resizedFileName,
		}
		go mystorage.PushTaskToExchangeUploadImage(resizedObject)
	}
}

func runResizeWorker(numWorker int){
	for i:=0;i<=numWorker;i++{
		go ResizeWorker(tasks_resize)
	}
}

var tasks_resize <- chan amqp.Delivery

func InitResizeConsumer(){
	conn,err := amqp.Dial(os.Getenv("AMQP_URL"))
	C1,err = conn.Channel()
	if err!=nil {
		panic(err)
	}
	C1.QueueDeclare("ResizeImage",true,false,false,false,nil)
	C1.QueueBind("ResizeImage","","ProcessImage",true, map[string]interface{}{
		"type":"image/jpeg",
		"job":"resize",
	})
	tasks_resize,_ = C1.Consume("ResizeImage","ResizeImageWorker",false,true,false,false,nil)
	go runResizeWorker(NumWorker)
}