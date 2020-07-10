package wart

import (
	"fmt"
	"html"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/robertkrimen/otto"
	log "github.com/sirupsen/logrus"
)

type EndpointMeta struct {
	Key     string
	Stopped bool
	vm      *otto.Otto
}

func (tm *EndpointMeta) getVM() *otto.Otto {
	return tm.vm
}

func (tm *EndpointMeta) getSource(w *Wart) (source string) {
	source = w.Client.HGet(ctx, tm.Key, "Source").Val()
	return
}

func getEndpoint(w *Wart, path string) (em *EndpointMeta) {
	em = &EndpointMeta{}
	em.Key = w.Cluster + ":Endpoints:" + html.EscapeString(path)
	if em.getSource(w) == "" {
		em = nil
	} else {
		em.vm = otto.New()
		applyLibrary(w, em)
	}
	///TODO - Add some checks to see if the endpoint is enabled or not.
	return
}

func (em *EndpointMeta) run(wart *Wart, w http.ResponseWriter, r *http.Request) {
	source := em.getSource(wart)
	output := ""
	if source != "" {
		b, _ := ioutil.ReadAll(r.Body)
		errorThrown := false
		em.vm.Set("request", map[string]interface{}{
			"Method": r.Method,
			"Path":   html.EscapeString(r.URL.Path),
			"Query":  r.URL.Query(),
			"GetHeader": func(key string) otto.Value {
				value, _ := otto.ToValue(r.Header.Get(key))
				return value
			},
			"Body": string(b),
		})
		em.vm.Set("response", map[string]interface{}{
			"Write": func(value string) {
				output += value
			},
			"SetContentType": func(value string) {
				w.Header().Set("Content-Type", value)
			},
			"SetHeader": func(key string, value string) {
				w.Header().Set(key, value)
			},
			"Error": func(errorString string, status int) {
				http.Error(w, errorString, status)
				errorThrown = true
			},
		})

		//Split the script up
		inputS := strings.Split(source, "<?")
		for i := 0; i < len(inputS); i++ {
			if strings.Contains(inputS[i], "?>") {
				s := strings.Split(inputS[i], "?>")
				script := s[0]
				afterScript := s[1]
				_, err := em.vm.Run(script)

				if err != nil {
					wart.Client.HSet(ctx, em.Key, "Error", err.Error())
					wart.Client.HSet(ctx, em.Key, "ErrorTime", time.Now())
					log.WithError(err).Error("Syntax error in script.")
					errorThrown = true
					http.Error(w, err.Error(), http.StatusInternalServerError)
					break
				}

				if len(afterScript) > 0 {
					output += afterScript
				}
			} else {
				output += inputS[i]
			}
		}
		if !errorThrown {
			fmt.Fprintf(w, output)
		}
	} else {
		http.Error(w, "Endpoint not found", http.StatusNotFound)
	}
}
