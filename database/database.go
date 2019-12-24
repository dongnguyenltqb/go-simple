package database
// Use Elasticsearch v6.x
import (
	"context"
	"fmt"
	"github.com/dongnguyenltqb/go-rabbit/utils"
	"github.com/olivere/elastic"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"os"
	"time"
)

var ES6 *elastic.Client
var MongoCLI *mongo.Client

type ModelsType struct {
	Users *mongo.Collection
	Object *mongo.Collection
}

var Models ModelsType

func Init() {
	var err error
	ES6,err = elastic.NewClient(elastic.SetSniff(false))
	if err != nil {
		fmt.Println("da co loi ",err)
		panic(err)
		return
	}
	info, code, _ := ES6.Ping(os.Getenv("ELASTIC_URL")).Do(context.Background())
	fmt.Println(utils.ApplyStyle("bold", "yellow", fmt.Sprintf("%v", info)), code)
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	MongoCLI, _= mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGODB_URL")))
	Models.Users = MongoCLI.Database("dongnguyen").Collection("users")
	Models.Object = MongoCLI.Database("dongnguyen").Collection("storageObject")
}