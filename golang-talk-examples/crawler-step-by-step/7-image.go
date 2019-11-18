package main // define package

import (
	"fmt"
	"math/rand"
	"sync"
)                  // import necessary package to print to console
import "net/http"  // import part of the net package to serve http requests and make requests
import "bufio" // we'll switch from io's reader to bufio's scanner to read the body line by line
import "os" // we'll use it to create file writers
import "html/template"

const addr = "localhost:55555" // define an address to listen on, this will not change so should be const
var urls chan string
var links []string
var linksLock sync.RWMutex
var tmplt *template.Template

func main() { // define main function
	urls = make(chan string, 10)
	var err error
	tmplt = template.New("index.html")       //create a new template with some name
	tmplt, err = tmplt.ParseFiles("index.html") //parse some content and generate a template, which is an internal representation
	if err != nil {
		fmt.Println(err)
	}
	http.HandleFunc("/", handle) // add a handler to the default ServeMux


	f, err := os.OpenFile("data.txt", os.O_APPEND | os.O_CREATE | os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(f) // create a new scanner instance
	for scanner.Scan() { // while there is a new line
		links = append(links, scanner.Text())
	}
	if err = scanner.Err(); err != nil {
		fmt.Println("Scanner error", err)
	}
	fmt.Println("Loaded links", links)
	go func() {
		for {
			select {
			case url := <- urls:
				_, err := fmt.Fprintf(f, url + "\n")
				if err != nil {
					fmt.Println(err)
				}
				linksLock.Lock()
				links = append(links, url)
				linksLock.Unlock()
				fmt.Println("Adding", url, len(links))
			default:


			}

		}
	}()

	err = http.ListenAndServe(addr, nil) // start listening on the addres and instruct to use the default ServeMux
	fmt.Println(err.Error()) // ListenAndServe blocks execution unless an error occurs, so we log that here
}

func handle (w http.ResponseWriter, r *http.Request) { // define a function that will handle requests
	fmt.Println("request from", r.RemoteAddr, "method", r.Method) // log an incoming request

	if r.Method == http.MethodPost { // we work with a body supplied by post request
		scanner := bufio.NewScanner(r.Body) // create a new scanner instance

		linksLock.RLock()
		defer linksLock.RUnlock()
		var newLinks []string
		for scanner.Scan() { // while there is a new line
			newLinks = append(newLinks, scanner.Text()) // send the line to the jobs channel, would block execution if channel becomes full
		}

		for _, url := range links {
			for _, newUrl := range newLinks {
				if newUrl == url {
					w.WriteHeader(http.StatusConflict)
					return
				}
			}
		}

		for _, url := range newLinks {
			urls <- url
		}

		w.WriteHeader(http.StatusOK)  // write the header to the outgoing socket with 200 status code
	} else if r.Method == http.MethodGet {
		linksLock.RLock()
		defer linksLock.RUnlock()
		linksLen := len(links)
		if linksLen == 0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		index := rand.Intn(len(links) )
		err := tmplt.Execute(w, links[index]) //merge template ‘t’ with content of ‘p’
		if err != nil {
			fmt.Println(err)
		}
	}	else { // other HTTP methods
		w.WriteHeader(http.StatusMethodNotAllowed) // we don't accept anything other than POST
	}
	// response is automatically finished when the handle function returns
}

