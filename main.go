package main

import (
	"bitbucket.org/mjl/asset"
	"bitbucket.org/mjl/httpvfs"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	mqtt "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	ttnshared "github.com/TheThingsNetwork/server-shared"
	"golang.org/x/tools/godoc/vfs"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type Config struct {
	Addr    string
	Verbose bool
}

var config Config

func init() {
	flag.StringVar(&config.Addr, "addr", "localhost:8000", "Address to listen on")
	flag.BoolVar(&config.Verbose, "verbose", false, "Verbose logging")
}

var fs vfs.FileSystem
var packetMux *Mux
var gatewayMux *Mux

func mqttConnect() *mqtt.Client {
	var mqttClient *mqtt.Client

	clientId := func() string {
		id := make([]byte, 6)
		_, err := rand.Read(id)
		if err != nil {
			log.Fatal(err)
		}
		return fmt.Sprintf("ttnmsgmap-%x", id)
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker("tcp://croft.thethings.girovito.nl:1883")
	opts.SetClientID(clientId())
	opts.SetKeepAlive(20)
	opts.SetOnConnectHandler(func(client *mqtt.Client) {
		tokenc := make(chan mqtt.Token)
		subscribe := func(topic string, handler mqtt.MessageHandler) {
			token := mqttClient.Subscribe(topic, 0, handler)
			token.Wait()
			tokenc <- token
		}
		go subscribe("nodes/+/packets", packetHandler)
		go subscribe("gateways/+/status", gatewayHandler)
		t0 := <-tokenc
		t1 := <-tokenc
		if t0.Error() != nil {
			log.Println("error subscribing:", t0.Error())
		}
		if t1.Error() != nil {
			log.Println("error subscribing:", t1.Error())
		}
	})
	opts.SetConnectionLostHandler(func(client *mqtt.Client, err error) {
		log.Println("mqtt connection lost (reconnecting):", err)
	})
	mqttClient = mqtt.NewClient(opts)

	// making this non-fatal. maybe the library will connect again later?
	go func() {
		if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
			log.Println("connecting to mqtt:", token.Error())
		}
	}()

	return mqttClient
}

func packetHandler(client *mqtt.Client, msg mqtt.Message) {
	var packet ttnshared.RxPacket
	err := json.Unmarshal(msg.Payload(), &packet)
	if err != nil {
		log.Println("unmarshalling packet payload:", err)
		return
	}

	// Decode payload
	data, err := base64.StdEncoding.DecodeString(packet.Data)
	if err != nil {
		log.Println("base64-decoding data:", err)
		return
	}

	value := map[string]interface{}{
		"devAddr":    packet.NodeEui,
		"gatewayEui": packet.GatewayEui,
		"time":       packet.Time,
		"frequency":  *packet.Frequency,
		"dataRate":   packet.DataRate,
		"rssi":       *packet.Rssi,
		"snr":        *packet.Snr,
		"data":       fmt.Sprintf("%s", data),
		"dataHex":    fmt.Sprintf("%x", data),
	}

	if config.Verbose {
		log.Printf("packet %#v", value)
	}
	packetMux.Input <- value
}

func gatewayHandler(client *mqtt.Client, msg mqtt.Message) {
	v := map[string]interface{}{}
	err := json.Unmarshal(msg.Payload(), &v)
	if err != nil {
		log.Println("unmarshal gateway payload:", err)
		return
	}

	if config.Verbose {
		log.Printf("gateway %#v", v)
	}
	gatewayMux.Input <- v
}

func makeSubscribe(mux *Mux, path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.Method != "GET" {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != path {
			http.NotFound(w, r)
			return
		}

		f, ok := w.(http.Flusher)
		if !ok {
			log.Fatal("flushing http responses not supported")
		}

		h := w.Header()
		h.Set("content-type", "text/event-stream")
		h.Set("cache-control", "no-cache, max-age=0")
		w.WriteHeader(200)

		fmt.Fprint(w, "retry: 1000\n")
		f.Flush()

		msgc := make(chan interface{}, 2)
		responsec := make(chan int64)
		mux.Register <- Register{Message: msgc, Response: responsec}
		id := <-responsec

		defer func() {
			if err := recover(); err != nil {
				mux.Unregister <- id
			}
		}()

		sse := func(v interface{}) {
			data, err := json.Marshal(v)
			if err != nil {
				log.Fatal("could not marshal to json: " + err.Error())
			}
			_, err = fmt.Fprintf(w, "data: %s\n\n", data)
			if err != nil {
				return
			}
			// xxx this should return an error...
			f.Flush()
		}

		for {
			msg := <-msgc
			sse(msg)
		}
	}
}

func indexhtml(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.Method != "GET" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	f, err := fs.Open("/index.html")
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer f.Close()
	h := w.Header()
	h.Set("content-type", "text/html")
	h.Set("cache-control", "no-cache, max-age=0")
	_, _ = io.Copy(w, f)
}

var gatewaysBuf []byte
var gatewaysEnd time.Time

// for speed, and so we can run on https (ttn websites doesn't have https).
func gateways(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.Method != "GET" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.URL.Path != "/gateways/" {
		http.NotFound(w, r)
		return
	}

	c := make(chan []byte)
	ttnGateways <- c
	buf := <-c
	if buf == nil {
		http.Error(w, "could not fetch gateways", http.StatusInternalServerError)
		return
	}
	h := w.Header()
	h.Set("content-type", "application/json")
	h.Set("cache-control", "max-age=60")
	_, _ = w.Write(buf)
}

var mqttClient *mqtt.Client

var ttnGateways chan chan []byte

func main() {
	packetMux = NewMux()
	gatewayMux = NewMux()

	ttnGateways = make(chan chan []byte)
	go func() {
		var buf []byte
		bufc := make(chan []byte)
		bufEnd := time.Now()

		fetch := func() {
			resp, err := http.Get("http://thethingsnetwork.org/api/v0/gateways/")
			if err != nil {
				log.Println("fetching ttn gateways", err)
				bufc <- nil
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				log.Printf("unexpected status %d from ttn api", resp.StatusCode)
				bufc <- nil
				return
			}
			data, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Println("error reading gateways from ttn api", err)
				bufc <- nil
				return
			}
			bufc <- data
		}

		for {
			go fetch()
			buf = <-bufc
			if buf != nil {
				bufEnd = time.Now().Add(time.Second * 120)
				break
			}
			time.Sleep(time.Second)
		}

		fetching := false
		for {
			select {
			case c := <-ttnGateways:
				c <- buf

				if !fetching && bufEnd.After(time.Now()) {
					go fetch()
					fetching = true
				}

			case nbuf := <-bufc:
				fetching = false
				if nbuf != nil {
					buf = nbuf
					bufEnd = time.Now().Add(time.Second * 120)
				}
			}
		}
	}()

	flag.Parse()
	if len(flag.Args()) != 0 {
		log.Fatal("bad usage, no arguments allowed")
	}

	fs = asset.Fs()
	if err := asset.Error(); err != nil {
		log.Println("using local assets")
		fs = vfs.OS("assets")
	}

	mqttClient = mqttConnect()

	http.HandleFunc("/", indexhtml)
	http.Handle("/s/", http.FileServer(httpvfs.New(fs)))
	http.HandleFunc("/sub/packets/", makeSubscribe(packetMux, "/sub/packets/"))
	http.HandleFunc("/sub/gateways/", makeSubscribe(gatewayMux, "/sub/gateways/"))
	http.HandleFunc("/gateways/", gateways)

	log.Println("listening on", config.Addr)
	log.Fatal(http.ListenAndServe(config.Addr, nil))
}
