package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/codingconcepts/env"
)

type response struct {
	Code    int    `json:"code"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type config struct {
	Port      int    `env:"PORT" default:"8080"`
	Directory string `env:"DIR" default:"."`
}

var c config

func saveJson(r io.Reader, fileName string) (err error) {
	js, err := ioutil.ReadAll(r)
	if err != nil {
		log.Println("Could not read body", err)
		return
	}

	f, err := os.Create(fileName)
	if err != nil {
		log.Println("Could not create file", err)
		return
	}
	defer f.Close()

	_, err = f.Write(js)
	if err != nil {
		log.Println("Could not write file", err)
		return
	}

	return
}

func writeResponse(w http.ResponseWriter, code int, message string) {
	resp := response{
		Code:    code,
		Message: message,
	}

	switch resp.Code {
	case http.StatusOK:
		resp.Status = "success"
	default:
		resp.Status = "failure"
	}

	w.WriteHeader(resp.Code)
	w.Header().Add("Content-Type", "application/json")
	js, _ := json.Marshal(resp)
	w.Write(js)

	log.Println(string(js))
}

func main() {
	c = config{}
	err := env.Set(&c)
	if err != nil {
		log.Println("Could not configure", err)
		os.Exit(1)
	}

	outFile := filepath.Join(c.Directory, "1password-credentials.json")
	port := fmt.Sprintf(":%d", c.Port)

	mux := http.NewServeMux()
	srv := http.Server{Addr: port, Handler: mux}
	ctx, cancel := context.WithCancel(context.Background())

	log.Printf("Listening on port %d, writing to %s\n", c.Port, outFile)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "PUT":
			for _, s := range r.Header["Content-Type"] {
				if s == "application/json" {
					err := saveJson(r.Body, outFile)
					if err != nil {
						writeResponse(w, http.StatusInternalServerError, err.Error())
					} else {
						writeResponse(w, http.StatusOK, "credentials received")
						cancel()
					}
					return
				}
			}
			writeResponse(w, http.StatusBadRequest, "only json accepted")
		default:
			writeResponse(w, http.StatusNotImplemented, "unsupported request method")
		}
	})

	go func() {
		err = srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Println("Could not start server", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	srv.Shutdown(context.Background())
}
