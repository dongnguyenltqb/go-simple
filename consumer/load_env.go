package consumer

import (
	"os"
	"strconv"
)

func LoadEnv(){
	var err error
	NumWorker,err = strconv.Atoi(os.Getenv("NUM_WORKER"))
	if err != nil {
		panic(err)
	}
}