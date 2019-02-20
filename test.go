package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("Hello, playground")
//	c := make(chan bool)
	for i:=0;i<9000;i++ {
	 go func() {
	 	for {
                  time.Sleep(time.Duration(1)*time.Second)
                }	
	 }()
	}
	fmt.Println("xxx")
	for {
	   time.Sleep(time.Duration(1) * time.Second)
	}
}

