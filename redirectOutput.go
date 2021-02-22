package main

import (
	"github.com/gin-gonic/gin"
	"log"
	"os"
	"time"
)

func redirectLogFile() {
	var f *os.File = nil
	for {
		t := time.Now().String()[0:10]
		fileDir := `log/` + t[0:7]
		if info, err := os.Stat(fileDir); os.IsNotExist(err) || !info.IsDir() {
			err = os.MkdirAll(fileDir, 0755)
			if err != nil {
				log.Println("can't make dir " + fileDir)
				continue
			}
		}

		if s, err := f.Stat(); err != nil || s.Name()[0:10] != t {
			f.Close()
			fileName := fileDir + `/` + t + `.log`
			f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
			if err != nil {
				log.Println(err)
			}

			gin.DefaultWriter = f
			os.Stdout = f
			log.SetOutput(f)

		}

		time.Sleep(time.Minute * 20)
	}
	if f != nil {
		f.Close()
	}
}

func redirectErrorFile() {
	var f *os.File = nil
	for {
		t := time.Now().String()[0:7]

		if s, err := f.Stat(); err != nil || s.Name()[0:7] != t {
			f.Close()
			fileName := `error/` + t + `.log`
			f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
			if err != nil {
				log.Println(err)
			}

			os.Stderr = f
		}

		time.Sleep(time.Hour * 6)
	}
	if f != nil {
		f.Close()
	}
}
