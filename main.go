package main

import (
	"encoding/gob"
	"fmt"
	"golang.org/x/net/websocket"
	"gopkg.in/yaml.v3"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"time"
	"github.com/kballard/go-shellquote"
)

type Endpoint struct {
	Path         string
	Call         string
	Arguments    []string
	ResponseType string `yaml:"response-type"`
	ContentEncoding string `yaml:"content-encoding"`
	Variables    map[string]string
}

type Configuration struct {
	Endpoints map[string]Endpoint `yaml:"endpoints,flow"`
	Variables map[string]string
	Host      string
}

func printConfiguration(conf Configuration) {
	log.Printf("Configuration:\n")
	log.Printf(" Host: %v\n", conf.Host);
	log.Printf(" Endpoints: %v\n", len(conf.Endpoints));
	for k, e := range conf.Endpoints {
		log.Printf("  %s: %s\n", k, e.Call)
		if e.ResponseType != "" {
			log.Printf("     Response Type: %s\n", e.ResponseType)
		}
		if e.ContentEncoding != "" {
			log.Printf("     Content Encoding: %s\n", e.ContentEncoding)
		}
		if len(e.Variables) > 0 {
			log.Printf("     Variables:\n")
			for key, value := range e.Variables {
				log.Printf("       %s = %s\n", key, value)
			}
		}
	}
	log.Printf("Global Variables:\n")
	for key, value := range conf.Variables {
		log.Printf("   %s = %s\n", key, value)
	}
}

func serializeArgs(wd io.Writer, m Endpoint) {
	e := gob.NewEncoder(wd)
	err := e.Encode(m)
	if err != nil {
		fmt.Println(`failed gob Encode`, err)
	}
}

func deserializeArgs(rd io.Reader) Endpoint {
	var m Endpoint
	d := gob.NewDecoder(rd)
	err := d.Decode(&m)
	if err != nil {
		fmt.Println(`failed gob Decode`, err)
	}
	return m
}

func generic_handle_request(url *url.URL, config *Configuration, reader io.Reader, writer io.Writer, extraArgs []string) error {
	if writer == nil {
		writer = io.Discard
	}
	path := url.Path

	args2, ok := config.Endpoints[path]
	if !ok {
		return fmt.Errorf("Failed call %s\n", url.String())
	}
	log.Printf("Opened %s :%+v  - %s\n", url.String(), args2, args2.Call)
	
	args := args2.Arguments
	var cmd = exec.Command(args[0], args[1:]...)
	q := url.Query()

	tlen := 0
	for _, v := range q {
		tlen = tlen + len(v)
	}
	env2 := make([]string, tlen+len(args2.Variables)+len(config.Variables))
	j := 0
	for k, v := range q {
		for i := 0; i < len(v); i++ {
			env2[j] = fmt.Sprintf("%s=%s", k, v[i])
			j += 1
		}
	}
	for k, v := range args2.Variables {
		env2[j] = fmt.Sprintf("%s=%s", k, v)
		j += 1
	}
	for k, v := range config.Variables {
		env2[j] = fmt.Sprintf("%s=%s", k, v)
		j += 1
	}
	cmd.Env = append(append(env2, extraArgs...), cmd.Env...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("{}\n", err)
	}

	{
		stderr, err := cmd.StderrPipe()
		if err != nil {
			log.Printf("{}", err)
		}
		go func() {
			all, eeeer := ioutil.ReadAll(stderr)
			if eeeer != nil {
				log.Printf("{}\n", eeeer)
			}
			if all != nil && len(all) > 0 {
				log.Printf("Stderr: %s", all)
			}
			stderr.Close()
		}()
	}

	if reader != nil {
		stdin, err := cmd.StdinPipe()
		if err != nil {
			log.Printf("Unable to open stdin: %s\n", err.Error())
		}
		go func() {
			io.Copy(stdin, reader)
			stdin.Close()
		}()
	}

	var e = cmd.Start()

	io.Copy(writer, stdout)
	cmd.Wait()
	if e != nil {
		log.Printf("Request complete  error: {}\n", e)
	} else {

		log.Printf("closing.\n")
	}
	return nil
}

func handle_ws_request(conn *websocket.Conn, config *Configuration) {
	url := conn.Config().Location
	generic_handle_request(url, config, conn, conn, []string{fmt.Sprintf("request_path=%v", url.Path)})
	conn.Close()
}

func handle_http_request(w http.ResponseWriter, req *http.Request, config *Configuration) {
	url := req.URL

	stdinData := req.Body
	stdOutData := w
	if req.Method == http.MethodGet {
		stdinData = nil
	}
	if req.Method == http.MethodPut {
		stdOutData = nil
	}
	args2, ok := config.Endpoints[url.Path]

	if !ok {
		w.WriteHeader(http.StatusNotFound)
		io.WriteString(w, "Not found")
		return
	}
	if ok {
		if args2.ResponseType != "" {
			w.Header().Add("Content-Type", args2.ResponseType)
		}
		if args2.ContentEncoding != "" {
			w.Header().Add("Content-Encoding", args2.ContentEncoding)
		}
	}
	
	err := generic_handle_request(url, config, stdinData, stdOutData,
		[]string{
			fmt.Sprintf("request_length=%v", req.ContentLength),
			fmt.Sprintf("request_path=%v", url.Path),
		})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, "Bad Request")

		return
	}
}

func make_request_handler(config *Configuration, ws http.Handler) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		// check if its a websocket request hidden inside a http request.
		upgrade, ok := req.Header["Upgrade"]
		if ok && upgrade[0] == "websocket" {
			ws.ServeHTTP(w, req)
		} else {
			handle_http_request(w, req, config)
		}
	}
}

func splitCall(call string) []string {
	m, _:= shellquote.Split(call);	
	return m
}

func mergeConfigFiles(a, b Configuration) Configuration {
	c := Configuration{
		Endpoints: make(map[string]Endpoint),
		Variables: make(map[string]string),
		Host: a.Host,
	}
	if c.Host == "" {
		c.Host = b.Host
	}

	for k,v := range a.Endpoints {
		c.Endpoints[k] = v
	}
	for k,v := range b.Endpoints {
		c.Endpoints[k] = v
	}
	for k,v := range a.Variables {
		c.Variables[k] = v
	}
	for k,v := range b.Variables {
		c.Variables[k] = v
	}
	
	return c;
}

func readConfigFile(path string) Configuration {
	var x Configuration
	dat, err := os.Open(path)
	if err != nil {
		log.Printf("%v\n", err)
		return x
	}
	dec := yaml.NewDecoder(dat)
	err = dec.Decode(&x)
	if err != nil {
		log.Printf("%v\n", err)
		return x
	}
	
	for i, ep := range x.Endpoints {
		callargs, err := shellquote.Split(ep.Call);
		if err != nil {
			log.Printf("warning processing %s: %s", i, err.Error());
		}
		ep.Arguments = callargs;
		
		ep.Path = i;
		if ep.Variables == nil {
			ep.Variables = make(map[string]string);
		}
		for k,v := range x.Variables {
			ep.Variables[k] = v 
		}

		
		x.Endpoints[i] = ep
		if i[0] != '/' {
			x.Endpoints[fmt.Sprintf("/%s", i)] = ep
		}
	}
	return x
}

func readConfigFiles(paths[] string) Configuration {
	x := readConfigFile(paths[0])
	for i := 1; i < len(paths); i++{
		x2 := readConfigFile(paths[i])
		x = mergeConfigFiles(x, x2)
	}
	return x
}
func main() {
	
	args := os.Args[1:]
	if len(args) >= 1 {
		var path = args[0]
		x := readConfigFiles(args)
		
		printConfiguration(x)

		for k,v := range x.Variables {
			os.Setenv(k, v);
		}
		
		s1, err := os.Stat(args[0])
		if err != nil {
			return
		}
		ws := websocket.Handler(func(conn *websocket.Conn) {
			handle_ws_request(conn, &x)
		})
		http.HandleFunc("/", make_request_handler(&x, ws))
		go func() {
			for true {
				time.Sleep(200 * time.Millisecond)
				s2, _ := os.Stat(path)
				if s2 == nil {
					continue;
				}

				if s2.Size() == s1.Size() && s2.ModTime() == s1.ModTime() {
					continue
				}
				s1 = s2
				log.Printf("Reloading file\n")
				x = readConfigFiles(args)
				
			}
		}()
		srv := &http.Server{Addr: x.Host}
		err = srv.ListenAndServe()
		certPem := os.Getenv("SERVY_CERT_FILE")
		keyPem := os.Getenv("SERVY_KEY_FILE")

		if certPem != "" {
			log.Printf("Starting with TLS enabled")
			srv.ListenAndServeTLS(certPem, keyPem)
		} else {
			srv.ListenAndServe()
		}
		if err != nil {
			log.Printf("startListenAndServe Error: %v\n", err.Error())
		}
	}
}
