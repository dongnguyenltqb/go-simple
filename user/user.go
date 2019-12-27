package user

import (
	"context"
	"github.com/dongnguyenltqb/go-rabbit/database"
	"github.com/dongnguyenltqb/go-rabbit/utils"
	"encoding/json"
	"fmt"
	"time"
	"strings"
	"github.com/gin-gonic/gin"
	"github.com/olivere/elastic"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Email     string             `json:"email,omitempty" bson:"email"`
	Password  string             `json:"password,omitempty" bson:"password"`
	FirstName string             `json:"first_name,omitempty" bson:"first_name"`
	LastName  string             `json:"last_name,omitempty" bson:"last_name,omitempty"`
}

type ESUser struct {
	Email     string `json:"email,omitempty" bson:"email"`
	FirstName string `json:"first_name,omitempty" bson:"first_name"`
	LastName  string `json:"last_name,omitempty" bson:"last_name,omitempty"`

}

type UserQuery struct {
	Page    int    `json:"page" form:"page"`
	Size    int    `json:"size" form:"size"`
	Keyword string `json:"keyword" form:"keyword"`
}

func (u *User) CacheToRedis() error {
	cli := database.RedisCLI
	key := "USER_"+u.Email
	_,err := cli.HSet(key,"email",u.Email).Result()
	if err != nil {
		utils.Logger("error",err)
		return err
	}
	_,err = cli.HSet(key,"first_name",u.FirstName).Result()
	if err != nil {
		utils.Logger("error",err)
		return err
	}
	_,err = cli.HSet(key,"last_name",u.LastName).Result()
	if err != nil {
		utils.Logger("error",err)
		return err
	}
	return err
}

func (u *User) AddIfNotExist() error  {
	isExist,err := database.Models.Users.CountDocuments(context.Background(),bson.M{
		"email":u.Email,
	})
	fmt.Println()
	if err != nil {
		utils.Logger("error",err)
		return  err
	}
	if isExist >0 {
		return  nil
	}
	_,err = database.Models.Users.InsertOne(context.Background(),u)
	if err != nil {
		utils.Logger("error",err)
		return  err
	}
	esUser := ESUser{
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
	}
	_,err = database.ES6.Index().Index("users").Type("_doc").Id(u.ID.Hex()).BodyJson(esUser).Do(context.Background())
	if err != nil {
		utils.Logger("error",err)
		return  err
	}
	fmt.Println(utils.ApplyStyle("bold","blue","Added user to database"))
	return  nil
}


func CreateNewUser(c *gin.Context) {
	var u User
	if err := c.BindJSON(&u); err != nil {
		utils.Logger("error", err)
		c.JSON(400, gin.H{
			"ok":      false,
			"message": "Bad Request",
		})
		return
	}
	hashedByte, err := bcrypt.GenerateFromPassword([]byte(u.Password), 10)
	if err != nil {
		c.JSON(500, gin.H{
			"ok":      false,
			"message": err.Error(),
		})
		return
	}
	u.Password = string(hashedByte)
	ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
	users := database.Models.Users
	count, err := users.CountDocuments(ctx, bson.M{
		"email": u.Email,
	})
	if err != nil {
		c.JSON(500, utils.ApiResponse{
			Ok:      false,
			Message: err.Error(),
			Data:    nil,
		})
		return
	}
	if count > 0 {
		c.JSON(400, utils.ApiResponse{
			Ok:      false,
			Message: "Email was existed",
			Data:    nil,
		})
		return
	}
	insertResult, err := users.InsertOne(ctx, u)
	if err != nil {
		c.JSON(500, utils.ApiResponse{
			Ok:      false,
			Message: err.Error(),
			Data:    nil,
		})
		return
	}
	esdoc := ESUser{
		FirstName: u.FirstName,
		LastName:  u.LastName,
	}
	_, err = database.ES6.Index().
		Index("users").
		Type("_doc").
		Id(insertResult.InsertedID.(primitive.ObjectID).Hex()).
		BodyJson(esdoc).
		Do(context.Background())
	if err != nil {
		utils.Logger("error", err)
		c.JSON(500, utils.ApiResponse{
			Ok:      false,
			Message: err.Error(),
			Data:    nil,
		})
		return
	}

	c.JSON(200, utils.ApiResponse{
		Ok:      true,
		Message: "Created User.",
		Data:    insertResult.InsertedID,
	})
}

func GetUser(c *gin.Context) {
	ID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.Logger("error", err)
		c.JSON(400, utils.ApiResponse{
			Ok:      false,
			Message: "Wrong User Id",
			Data:    nil,
		})
		return
	}
	var u User
	ctx, _ := context.WithTimeout(context.Background(), time.Second)
	err = database.Models.Users.FindOne(ctx, bson.M{
		"_id": ID,
	}, &options.FindOneOptions{
		Projection: bson.M{
			"password": false,
		},
	}).Decode(&u)
	if err != nil {
		utils.Logger("error", err)
		c.JSON(404, utils.ApiResponse{
			Ok:      false,
			Message: "User not found",
			Data:    nil,
		})
		return
	}
	c.JSON(200, utils.ApiResponse{
		Ok:      true,
		Message: "User was found",
		Data: bson.M{
			"_id":  ID,
			"_doc": u,
		},
	})
}

func SearchUser(c *gin.Context) {
	var q UserQuery
	if err := c.BindQuery(&q); err != nil {
		utils.Logger("error", err)
		c.JSON(200, utils.ApiResponse{
			Ok:      false,
			Message: "Bad request check your query parammeter",
		})
		return
	}

	q.Keyword =  strings.ToLower(q.Keyword)
	boolQuery := elastic.NewBoolQuery()
	wc1 := elastic.NewTermQuery("first_name", q.Keyword)
	wc2 := elastic.NewTermQuery("last_name", q.Keyword)
	boolQuery.Should(wc1)
	boolQuery.Should(wc2)
	es := database.ES6

	searchResult, err := es.Search().Index("users").Type("_doc").Query(boolQuery).From((q.Page - 1) * q.Size).Size(q.Size).Do(context.Background())
	if err != nil {
		utils.Logger("error", err)
		c.JSON(400, utils.ApiResponse{
			Ok:      false,
			Message: err.Error(),
		})
		return
	}
	data := make([]bson.M, 0)
	for _, hit := range searchResult.Hits.Hits {
		source, _ := hit.Source.MarshalJSON()
		var original ESUser
		_ = json.Unmarshal(source, &original)
		data = append(data, bson.M{
			"_id":hit.Id,
			"_source":original,
		})
	}
	c.JSON(200, utils.ApiResponse{
		Ok:      true,
		Message: "Search successfully.",
		Data:   data,
	})
}

func ReIndexUserESSearch(c *gin.Context){
	go ChangeUserPassWord()
	ctx := context.Background()
	users := database.Models.Users
	cursor,err := users.Find(ctx,bson.M{})
	if err != nil {
		utils.Logger("error",err)
		c.JSON(500,utils.ApiResponse{
			Ok:false,
			Message:err.Error(),
		})
		return
	}
	defer cursor.Close(ctx)
	ES := database.ES6
	for cursor.Next(ctx){
		var u User
		cursor.Decode(&u)
		es_doc := ESUser{
			FirstName:u.FirstName,
			LastName:u.LastName,
		}
		ES.Index().Index("users").Type("_doc").Id(u.ID.Hex()).BodyJson(es_doc).Do(context.Background())
	}
	c.JSON(200,utils.ApiResponse{
		Ok:true,
		Message:"Done....",
	})
}

func ChangeUserPassWord(){
	users :=  database.Models.Users
	ctx := context.Background()
	hashedByte, err := bcrypt.GenerateFromPassword([]byte("123"), 10)
	if err != nil {
		utils.Logger("error",err)
	}
	users.UpdateMany(ctx,bson.M{},bson.M{
		"$set":bson.M{
			"password":string(hashedByte),
		},
	})
}

func RegistryUserController(app *gin.Engine) {
	app.POST("/login", Login)
	app.GET("/login/google", LoginWithGoogle)
	app.GET("/login/google/callback", LoginWithGoogleCallBack)

	r := app.Group("/user")
	r.PATCH("",ReIndexUserESSearch)
	r.POST("", CreateNewUser)
	r.GET("", IsAuthenticated,SearchUser)
	r.GET("/:id", IsAuthenticated,GetUser)
}
