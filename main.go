package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"net"
	"strconv"
	"encoding/gob"
	"io"
	"bufio"
	
)

type configuration struct{
	endpoint string
}

func serializeArgs(wd io.Writer, m []string)  {
	e := gob.NewEncoder(wd)
	err := e.Encode(m)
	if err != nil { fmt.Println(`failed gob Encode`, err) }
}

// go binary decoder
func deserializeArgs(rd io.Reader) []string {
	var m []string
	d := gob.NewDecoder(rd)
	err := d.Decode(&m)
	if err != nil { fmt.Println(`failed gob Decode`, err); }
	return m
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

	make_handler := func(rest []string) func(http.ResponseWriter, *http.Request) {
		
			handler := func(w http.ResponseWriter, req *http.Request) {
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
		return handler;
	}
	var port uint;
	newConnection := func(w http.ResponseWriter, req *http.Request) {
		log.Printf("Emit port!\n");
		fmt.Fprintf(w, "%v", port);
	}
	log.Printf("Listing for requests at {}\n", url)
	base := strings.Split(url, "/")[0]
	handleConnection := func(conn net.Conn){
		log.Printf("Handling Connection!\n");
		var args = deserializeArgs(conn);
		go http.HandleFunc(args[0], make_handler(args[1:]))
		buf := make([]byte, 1);
		conn.Read(buf);
		
		log.Printf("Closing con {}\n", args);
		conn.Close();
	}
	
	
	endpoint := url[len(base):]
	http.HandleFunc(endpoint, make_handler(rest))
	http.HandleFunc("/servy-conf", newConnection);

	go func(){
		ln, err2 := net.Listen("tcp", ":")
		if(err2 != nil){
			log.Fatal(err2);
		}
		addr := ln.Addr();
		log.Printf("%v", uint(addr.(*net.TCPAddr).Port));
		port = uint(addr.(*net.TCPAddr).Port);
		for {
			conn, err := ln.Accept()
			log.Printf("Got connection!\n");
			if err != nil {
				// handle error
			}
			go handleConnection(conn)
		}
	}()
	
	err := http.ListenAndServe(base, nil)
	if(err != nil){
		// this means that we could not create a server.
		// a new servy can be started by connecting to the localhost mux1.
		resp, err2 := http.Get(fmt.Sprintf("http://%s/servy-conf", base));
		if(err2 == nil){
			defer resp.Body.Close()
			body, err3 := ioutil.ReadAll(resp.Body)
			port2, _ := strconv.ParseUint(string(body), 10, 32)
			port = uint(port2)
			log.Printf(">>> %v {}\n", port, err3);
		}else{
			log.Fatal(err2);
			return;
		}

		conn,err3 := net.Dial("tcp", fmt.Sprintf("localhost:%v", port));
		serializeArgs(conn, append([]string{endpoint}, rest...));
		log.Printf("{} {}\n", conn, err3);
		defer conn.Close();
		bufio.NewReader(os.Stdin).ReadBytes('\n')
		
	}
}
