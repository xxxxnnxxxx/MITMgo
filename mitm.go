package main

import (
	"log"
	"mitmgo/src/manage"
)

func main() {

	mitmManager := manage.NewMITMManager()

	ret, err := mitmManager.ParseOpt()
	if err != nil {
		log.Println(err)
		return
	}

	if !ret {
		return
	}

	err = mitmManager.Initialize()
	if err != nil {
		log.Println(err)
		return
	}

	err = mitmManager.Do()
	if err != nil {
		log.Println(err)
		return
	}

}
