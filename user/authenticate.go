package user

import (
	"context"
	"encoding/json"
	"github.com/dongnguyenltqb/go-rabbit/database"
	"github.com/dongnguyenltqb/go-rabbit/utils"
	//"encoding/binary"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/golang/protobuf/proto"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

type Claims struct {
	Email string `json:"email"`
	ID    string `json:"_id"`
	jwt.StandardClaims
}

type Credential struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type GoogleUserDebugType struct {
	Email      string `json:"email"`
	Picture    string `json:"picture"`
	Name       string `json:"name"`
	GivenName  string `json:"given_name"`
	FamilyName string `json:"family_name"`
}

func GenerateResponseToken(u User) (string, error) {
	claim := Claims{
		Email: u.Email,
		ID:    u.ID.Hex(),
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(100000000000000000).Unix(),
		},
	}
	jwtKey := []byte(os.Getenv("JWT_SECRET_KEY"))
	tokenString := jwt.NewWithClaims(jwt.SigningMethodHS256, &claim)
	return tokenString.SignedString(jwtKey)
}

func Login(c *gin.Context) {
	var body Credential
	if err := c.BindJSON(&body); err != nil {
		utils.Logger("error", err)
		c.JSON(400, utils.ApiResponse{
			Ok:      false,
			Message: "Bad request",
			Data:    nil,
		})
		return
	}
	var u User
	ctx, _ := context.WithTimeout(context.Background(), time.Second)
	err := database.Models.Users.FindOne(ctx, bson.M{
		"email": body.Email,
	}).Decode(&u)
	if err != nil {
		utils.Logger("error", err)
		c.JSON(400, utils.ApiResponse{
			Ok:      false,
			Message: "Bad request",
			Data:    nil,
		})
		return
	}
	err = bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(body.Password))
	if err != nil {
		utils.Logger("error", err)
		c.JSON(400, utils.ApiResponse{
			Ok:      false,
			Message: err.Error(),
			Data:    nil,
		})
		return
	}

	responseToken, err := GenerateResponseToken(u)
	if err != nil {
		utils.Logger("error", err)
		c.JSON(400, utils.ApiResponse{
			Ok:      false,
			Message: err.Error(),
			Data:    nil,
		})
		return
	}
	c.JSON(200, utils.ApiResponse{
		Ok:      true,
		Message: "Login successfully",
		Data:    responseToken,
	})

}

func IsAuthenticated(c *gin.Context) {
	tokenString := c.GetHeader("x-auth-token")
	var claim Claims
	token, err := jwt.ParseWithClaims(tokenString, &claim, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("jWT_SECRET_KEY")), nil
	})
	if err != nil || !token.Valid {
		c.AbortWithStatus(401)
		utils.Logger("error", err)
		return
	}
	c.Set("user", claim)
	c.Next()
}

func LoginWithGoogle(c *gin.Context) {
	conf := &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_SECRET_KEY"),
		Endpoint:     google.Endpoint,
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		Scopes:       []string{"profile", "email", "openid"},
	}
	url := conf.AuthCodeURL("profile")
	c.Redirect(302, url)
}

func LoginWithGoogleCallBack(c *gin.Context) {
	conf := &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_SECRET_KEY"),
		Endpoint:     google.Endpoint,
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		Scopes:       []string{"profile", "email", "openid"},
	}
	token, err := conf.Exchange(context.Background(), c.Query("code"))
	idToken := token.Extra("id_token")
	if err != nil {
		utils.Logger("error", err)
		c.JSON(400, utils.ApiResponse{
			Ok:      false,
			Message: "Bad Request.",
		})
		return
	}
	resp, _ := http.Get(os.Getenv("GOOGLE_TOKEN_DEBUG_URL") + idToken.(string))
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	var decodedInfo GoogleUserDebugType
	if err = json.Unmarshal(body, &decodedInfo); err != nil {
		utils.Logger("error", err)
		c.JSON(400, utils.ApiResponse{
			Ok:      false,
			Message: "Bad Request.",
		})
		return
	}

	u := User{
		ID:        primitive.NewObjectID(),
		Email:     decodedInfo.Email,
		FirstName: decodedInfo.GivenName,
		LastName:  decodedInfo.FamilyName,
	}
	u.CacheToRedis()
	if err = u.AddIfNotExist(); err != nil {
		utils.Logger("error", err)
		c.JSON(400, utils.ApiResponse{
			Ok:      false,
			Message: "Bad Request.",
		})
		return
	}
	responseToken, _ := GenerateResponseToken(u)
	//
	v := &UserProtobuf{
		FirstName: "Dong",
		LastName:  "Nguyen",
		Contact: []*ContactProtobuf{
			&ContactProtobuf{
				PhoneNumber: "039 390 1228",
				Country:     "Viet Name",
			},
		},
	}
	fmt.Println(utils.ApplyStyle("bold", "yellow", fmt.Sprintf("%+v", v)))
	binary, _ := proto.Marshal(v)
	fmt.Println(binary)
	//
	originalBinary := []int64{10, 4,
		68,
		111,
		110,
		103,
		18,
		10,
		78,
		103,
		117,
		121,
		101,
		110,
		32,
		72,
		117,
		117,
		26,
		24,
		100,
		111,
		110,
		103,
		110,
		103,
		117,
		121,
		101,
		110,
		108,
		116,
		113,
		98,
		64,
		103,
		109,
		97,
		105,
		108,
		46,
		99,
		111,
		109,
		34,
		16,
		42,
		10,
		48,
		51,
		57,
		51,
		57,
		48,
		49,
		50,
		50,
		56,
		50,
		2,
		86,
		110}

	var buf []byte
	for _, value := range originalBinary {
		buf = append(buf, byte(value))
	}
	c.JSON(200, utils.ApiResponse{
		Ok:      true,
		Message: "Successfully",
		Data: bson.M{
			"token":          responseToken,
			"user":           decodedInfo,
			"originalBinary": buf,
		},
	})

}
