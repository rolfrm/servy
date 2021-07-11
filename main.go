package main

import (
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"runtime/pprof"
	"strconv"
	"strings"
	"syscall"
	"sync"
	"golang.org/x/net/websocket"
)

func serializeArgs(wd io.Writer, m []string) {
	e := gob.NewEncoder(wd)
	err := e.Encode(m)
	if err != nil {
		fmt.Println(`failed gob Encode`, err)
	}
}

func deserializeArgs(rd io.Reader) []string {
	var m []string
	d := gob.NewDecoder(rd)
	err := d.Decode(&m)
	if err != nil {
		fmt.Println(`failed gob Decode`, err)
	}
	return m
}

func SetupCloseHandler(srv *http.Server) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		srv.Close()
		os.Exit(0) // disable this to do profiling
	}()
}

func handle_ws_request(conn *websocket.Conn, ep map[string][]string){
	// this is extremely similar to handle_request
	url := conn.Config().Location;
	args, ok := ep[url.Path]
	if !ok {
		log.Printf("Failed websocket %s\n", url.Path)
		conn.Close();
		return;
	}

	log.Printf("Websocket Opened %s :%v\n", url.Path, args)
	var cmd = exec.Command(args[0], args[1:]...)
	q := url.Query()

	tlen := 0
	for _, v := range q {
		tlen = tlen + len(v)
	}
	env2 := make([]string, tlen)
	j := 0
	for k, v := range q {
		for i := 0; i < len(v); i++ {
			env2[j] = fmt.Sprintf("%s=%s", k, v[i])
			j += 1
		}
	}
	servy_args := os.Getenv("SERVY_ARGS")
	env3 := strings.Split(servy_args,";")
	cmd.Env = append(append(cmd.Env, env3...), env2...)
	
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("{}", err)
		return
	}
	
	{
		stderr, err := cmd.StderrPipe()
		if err != nil {
			log.Printf("{}", err)
			return
		}
		go func() {
			ioutil.ReadAll(stderr)
			stderr.Close()
		}()
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Printf("{}", err)
		return
	}
	go func() {
		io.Copy(stdin, conn)
		stdin.Close()
	}()


	var e = cmd.Start()
	io.Copy(conn, stdout)
	cmd.Wait()
	if e != nil {
		log.Printf("Request complete  error: {}", e)
	} else {

		log.Printf("Websocket closing.")
	}
	
	conn.Close()
}

func handle_request(w http.ResponseWriter, req *http.Request, ep map[string][]string) {
	url := req.URL;
	args, ok := ep[url.Path]
	if ok == false || len(args) == 0 {
		log.Printf("Failed request %s\n", url.Path)
		fmt.Fprintf(w, "Endpoint not found.")
		
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Printf("Request %s :%v\n", url.Path, args)
	var cmd = exec.Command(args[0], args[1:]...)
	q := url.Query()

	tlen := 0
	for _, v := range q {
		tlen = tlen + len(v)
	}
	env2 := make([]string, tlen + 1)
	j := 0
	for k, v := range q {
		for i := 0; i < len(v); i++ {
			env2[j] = fmt.Sprintf("%s=%s", k, v[i])
			j += 1
		}
	}
	env2[j] = fmt.Sprintf("request_length=%v",req.ContentLength)
	servy_args := os.Getenv("SERVY_ARGS")
	env3 := strings.Split(servy_args,";")
	cmd.Env = append(append(cmd.Env, env3...), env2...)
	
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("{}", err)
		return
	}

	{
		stderr, err := cmd.StderrPipe()
		if err != nil {
			log.Printf("{}", err)
			return
		}
		go func() {
			ioutil.ReadAll(stderr)
			stderr.Close()
		}()
	}

	if req.Method != http.MethodGet {
		stdin, err := cmd.StdinPipe()
		if err != nil {
			log.Printf("{}", err)
			return
		}
		go func() {
			io.Copy(stdin, req.Body)
			stdin.Close()
		}()
	}

	var e = cmd.Start()
	io.Copy(w, stdout)
	cmd.Wait()
	if e != nil {
		fmt.Fprintf(w, "{}", e)
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("Request complete  error: {}", e)
	} else {

		log.Printf("Request complete successfully.")
	}
}

func make_request_handler(url *url.URL, endpoints map[string][]string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		handle_request(w, req, endpoints)
	}
}

func startEndPoint(url *url.URL, endpoints map[string][]string) {
	if (url.Scheme == "wss" || url.Scheme == "ws") {
		handler := websocket.Handler(func (conn *websocket.Conn) {
			handle_ws_request(conn, endpoints)})
		http.Handle(url.Path, handler)
	} else {
		http.HandleFunc(url.Path, make_request_handler(url, endpoints))
	}
}

func startListenAndServe(srv *http.Server, u *url.URL) error {
	if u.Scheme == "https" {
		certPem := os.Getenv("SERVY_CERT_FILE")
		keyPem := os.Getenv("SERVY_KEY_FILE")
		log.Printf("SSL: %s %s\n", certPem, keyPem)
		return srv.ListenAndServeTLS(certPem, keyPem)
	} else {
		return srv.ListenAndServe()
	}

}

func attachToServer(u *url.URL, args []string) error {
	log.Printf("Host: '%s' %s\n", u.Host, u.Scheme)

	// this means that we could not create a server.
	// a new servy can be started by connecting to the localhost mux1.

	scheme := u.Scheme
	if(scheme == "wss") {
		scheme = "https";
	}
	if(scheme == "ws") {
		scheme = "http";
	}

	
	resp, err2 := http.Get(fmt.Sprintf("%s://%s/servy-conf", scheme, u.Host))
	var port uint
	if err2 == nil {

		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		port2, _ := strconv.ParseUint(string(body), 10, 32)
		port = uint(port2)
	} else {
		return err2
	}

	conn, _ := net.Dial("tcp", fmt.Sprintf("localhost:%v", port))
	defer conn.Close()

	serializeArgs(conn, args)
	buf := make([]byte, 1)
	_, err3 := conn.Read(buf)
	if err3 != nil && err3.Error() == "EOF" {
		return errors.New("Server Connection Lost")
	}
	return err3
}




func main_server(args []string, u *url.URL) error {
	rest := args[1:]

	ep := make(map[string][]string)

	var port uint
	newConnection := func(w http.ResponseWriter, req *http.Request) {
		// when somebody ask, respond with the right port to connect
		fmt.Fprintf(w, "%v", port)
	}
	log.Printf("Listing for requests at {}\n", args[0])

	srv := &http.Server{Addr: u.Host}
	SetupCloseHandler(srv)
	var mu sync.Mutex
	handleConnection := func(conn net.Conn) {
		var args = deserializeArgs(conn)
		defer conn.Close()
		
		u, e := url.Parse(args[0])
		if e != nil {
			log.Fatal(e)
			return
		}
		_, ok := ep[u.Path]
		mu.Lock()
		ep[u.Path] = args[1:]
		mu.Unlock()
		if ok == false {
			// start a new handler as the endpoint has not previosly been seen
			
			go startEndPoint(u, ep)
		}
		buf := make([]byte, 1)
		conn.Read(buf)
		ep[u.Path] = make([]string, 0)
		log.Printf("Closing con {}\n", args)
	}

	ep[u.Path] = rest
	startEndPoint(u, ep)
	log.Printf("Path: %s\n", u.Path)
	http.HandleFunc("/servy-conf", newConnection)

	go func() {
		ln, err2 := net.Listen("tcp", ":")
		if err2 != nil {
			log.Fatal(err2)
		}
		addr := ln.Addr()
		log.Printf("%v", uint(addr.(*net.TCPAddr).Port))
		port = uint(addr.(*net.TCPAddr).Port)
		for {
			conn, err := ln.Accept()
			log.Printf("Got connection!\n")
			if err != nil {
				log.Fatal(err) // handle error
			}
			go handleConnection(conn)
		}
	}()
	for {
		err := startListenAndServe(srv, u)
		if strings.Contains(err.Error(), "address already in use") == false {

			log.Printf("startListenAndServe Error: %v\n", err.Error())
			return err
		}
		err2 := attachToServer(u, args)
		if strings.Contains(err2.Error(), "Server Connection Lost") == false {
			log.Printf("attach To Process Error: %v\n", err2.Error())
			return err2
		}
	}

	return nil
}

func main() {

	if false {
		f, _ := os.Create("./prof")
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	//log.SetFlags(0)
	//log.SetOutput(ioutil.Discard)
	args := os.Args[1:]
	if len(args) < 2 {
		log.Fatal("At least two arguments must be supplied. [endpoint] and command.")
		return
	}
	u, err := url.Parse(args[0])
	if err != nil {
		log.Fatal(err)
		return
	}

	e := main_server(args, u)
	if e != nil {
		log.Printf("Error: %v\n", e)
	}
}
