package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"net"
	"strconv"
	"encoding/gob"
	"io"
	"net/url"
	"runtime/pprof"
	//"runtime"
	"os/signal"
	"syscall"
)

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

func SetupCloseHandler(srv *http.Server) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		srv.Close();
		//os.Exit(0)
	}()
}

func main() {

	if(false){
		f, _ := os.Create("./prof")
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	
	//log.SetFlags(0)
	//log.SetOutput(ioutil.Discard)
	args := os.Args[1:]
	if len(args) < 2 {
		//log.Fatal("!!!")
		return
	}
        
	rest := args[1:]
	ep := make(map[string] []string)
	
	handler :=  func(w http.ResponseWriter, req *http.Request) {
		args, ok := ep[req.URL.Path]
		if ok == false || len(args) == 0 {
			w.WriteHeader(http.StatusBadRequest);
			return;
		}
		
		log.Printf("Request %s :%v\n", req.URL.Path, args)
		var cmd = exec.Command(args[0], args[1:]...)
		q := req.URL.Query()
		
		tlen := 0
		for _, v := range q {
			tlen = tlen + len(v);
		}
		env2 := make([]string, tlen)
		j := 0;
		for k, v := range q {
			for i := 0; i < len(v); i++{
				env2[j] = fmt.Sprintf("%s=%s", k, v[i]);
				j += 1;
			}
		}

		cmd.Env = append(cmd.Env, env2...);
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Printf("{}", err)
			return;
		}

		{
			stderr, err := cmd.StderrPipe()
			if err != nil {
				log.Printf("{}", err)
				return;
			}
			go func (){
				ioutil.ReadAll(stderr)
				stderr.Close();
			}();
		}
		
		if req.Method != http.MethodGet {
			stdin, err := cmd.StdinPipe()
			if err != nil {
				log.Printf("{}", err)
				return;
			}
			go func (){
				io.Copy(stdin, req.Body);
				stdin.Close()
			}();
		}
		
		var e = cmd.Start()
		io.Copy(w, stdout)
		cmd.Wait();
		if e != nil {
			fmt.Fprintf(w, "{}", e)
			w.WriteHeader(http.StatusBadRequest);
			log.Printf("Request complete  error: {}", e)
		}else{
		
			log.Printf("Request complete successfully.")
		}
	}
	
	var port uint;
	newConnection := func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "%v", port);
	}
	log.Printf("Listing for requests at {}\n", args[0])
	u, err := url.Parse(args[0])
	if err != nil {
		log.Fatal(err);
		return;
	}
	srv := &http.Server{Addr: u.Host}
	SetupCloseHandler(srv)
	
	handleConnection := func(conn net.Conn){
		var args = deserializeArgs(conn);
		defer conn.Close();
		u, e := url.Parse(args[0]);
		if e != nil {
			log.Fatal(e);
			return;
		}
		_, ok := ep[u.Path];
		ep[u.Path] = args[1:];
		if ok == false {
			// start a new handler as the endpoint has not previosly been seen
			go http.HandleFunc(u.Path, handler)
		}
		buf := make([]byte, 1);
		conn.Read(buf);
		ep[u.Path] = make([]string, 0);
		log.Printf("Closing con {}\n", args);
	}


	
	ep[u.Path] = rest;
	http.HandleFunc(u.Path, handler)
	log.Printf("Path: %s\n", u.Path);
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

	if u.Scheme == "https" {
		certPem := os.Getenv("SERVY_CERT_FILE");
		keyPem := os.Getenv("SERVY_KEY_FILE");
		log.Printf("SSL: %s %s\n", certPem, keyPem);
		err = srv.ListenAndServeTLS(certPem, keyPem)
	}else{
		err = srv.ListenAndServe()

	}
	
	log.Printf("Host: '%s' %s\n", u.Host, u.Scheme);
	log.Printf("Error: {}\n", err);
	if(err != nil){
		log.Printf("Error: '%v'\n", err.Error());
		if(err.Error() == "http: Server closed"){
			return;
		}
		//if(err.Error() == 
		// this means that we could not create a server.
		// a new servy can be started by connecting to the localhost mux1.
		
		resp, err2 := http.Get(fmt.Sprintf("%s://%s/servy-conf", u.Scheme, u.Host));
		if(err2 == nil){
			
			defer resp.Body.Close()
			body, _ := ioutil.ReadAll(resp.Body)
			port2, _ := strconv.ParseUint(string(body), 10, 32)
			port = uint(port2)
		}else{
			log.Fatal(err2);
			return;
		}

		conn,_ := net.Dial("tcp", fmt.Sprintf("localhost:%v", port));
		defer conn.Close();
		
		serializeArgs(conn, args);
		buf := make([]byte, 1);
		conn.Read(buf);
	}
}
