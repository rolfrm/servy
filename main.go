package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)
type configuration struct{
	endpoint string
	

}

func main() {
	args := os.Args[1:]
	
	if len(args) == 1 {
		//json

	}
	if len(args) < 2 {
		log.Fatal("!!!")
		return
	}
        
	url := args[0]
	rest := args[1:]
	helloHandler := func(w http.ResponseWriter, req *http.Request) {
		fmt.Printf("Request starts")
		var cmd = exec.Command(rest[0], rest[1:]...)
		var env = os.Environ()
		log.Print(env)

		stdout, err := cmd.StdoutPipe()

		if err != nil {
			log.Fatal(err)
		}
		stderr, err2 := cmd.StderrPipe()
		if err2 != nil {
			log.Fatal(err2)
		}

		cstd := make(chan []byte, 1)
		cerr := make(chan []byte, 1)

		go func() {
			d, _ := ioutil.ReadAll(stdout)
			cstd <- d
		}()

		go func() {
			d, _ := ioutil.ReadAll(stderr)
			cerr <- d
		}()
		var e = cmd.Start()
		if e != nil {
			fmt.Fprintf(w, "{}", e)
		} else {
			w.Write(<-cstd)
			w.Write(<-cerr)
			cmd.Wait()
		}

		fmt.Printf("Request complete")
	}
	
	log.Printf("Listing for requests at {}\n", url)
	base := strings.Split(url, "/")[0]
	endpoint := url[len(base):]
	http.HandleFunc(endpoint, helloHandler)
	
	log.Fatal(http.ListenAndServe(base, nil))
}
