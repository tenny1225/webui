package webui

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"golang.org/x/net/websocket"
)

type Window interface {
	HandleFunc(p string, f func(w http.ResponseWriter, r *http.Request))
	Run(fun func())
	RunAndBindPort(port string, fun func())
	Close()
	Navigation(htmlPath string)
	BindWithName(key string, obj interface{})
	Bind(obj interface{})
	RemoveBind(key string)
	Eval(js string, f interface{})
}

func NewWindow(title string, x, y, w, h int64, staticPath string) Window {
	return &window{
		title:         title,
		w:             w,
		h:             h,
		x:             x,
		y:             y,
		staticPath:    staticPath,
		closeChannel:  make(chan int, 1),
		wlWaitChannel: make(chan int, 1),
		args:          make([]string, 0),
	}
}
func NewCommandsWindow(title string, x, y, w, h int64, staticPath string, commands []string) Window {
	return &window{
		title:         title,
		w:             w,
		h:             h,
		x:             x,
		y:             y,
		staticPath:    staticPath,
		closeChannel:  make(chan int, 1),
		wlWaitChannel: make(chan int, 1),
		args:          commands,
	}
}

type window struct {
	title         string
	x             int64
	y             int64
	w             int64
	h             int64
	staticPath    string
	closeChannel  chan int
	currentUrl    string
	addr          string
	inited        bool
	wl            *webSocketListener
	wlWaitChannel chan int
	args          []string
	server        *http.Server
}

func (w *window) HandleFunc(p string, f func(w http.ResponseWriter, r *http.Request)) {
	http.HandleFunc(p, f)
}
func (w *window) Run(fun func()) {
	w.RunAndBindPort("8000", fun)
}
func (w *window) RunAndBindPort(addr string, fun func()) {
	fmt.Println("RunAndBindPort")
	if strings.HasPrefix(addr, ":") {
		addr = addr[1:]
	}
	w.addr = addr
	go w.startHttpService(addr)
	<-time.Tick(time.Second)
	if fun != nil {
		go fun()
	}
	<-w.closeChannel
}
func (win *window) startHttpService(addr string) {
	http.Handle("/ws", websocket.Handler(func(wc *websocket.Conn) {
		fmt.Println("ws listen 0")
		win.wl = NewWebSocketListener(wc)
		fmt.Println("ws listen1 ")
		//win.wlWaitChannel <- 1
		fmt.Println("ws listen")
		win.wl.listen()
	}))
	http.HandleFunc("/file", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		p := r.FormValue("p")
		fn := path.Base(p)
		f, e := os.Open(p)
		if e != nil {
			w.WriteHeader(500)
			w.Write([]byte(e.Error()))
			return
		}
		defer f.Close()
		w.WriteHeader(200)
		w.Header().Add("Content-Disposition", "attachment;filename="+fn)
		io.Copy(w, f)
	})
	http.HandleFunc("/"+path.Base(win.staticPath)+"/", func(w http.ResponseWriter, r *http.Request) {

		f := path.Base(r.URL.Path)
		fmt.Println("f=", f)

		var buf []byte
		if f == DEFAULT_HTML_NAME {
			html := fmt.Sprintf(DEFAULT_HTML,
				win.title,
				win.currentUrl,
				win.addr)
			buf = []byte(html)
			w.WriteHeader(200)
			w.Write(buf)
		} else if f == NOT_FOUND_CHROME_HTML_NAME {
			buf = []byte(NOT_FOUND_CHROME_HTML)
			w.WriteHeader(404)
			w.Write(buf)
		} else {
			var e error
			fn := path.Join(win.staticPath, strings.TrimPrefix(r.URL.Path, "/"+path.Base(win.staticPath)))
			buf, e = ioutil.ReadFile(fn)
			if e != nil {
				w.Write([]byte(e.Error()))
				return
			}
			if path.Ext(fn) == ".css" {
				w.Header().Set("Content-Type", "text/css; charset=utf-8")
			}
			w.WriteHeader(200)
			w.Write(buf)
		}

	})
	win.server = &http.Server{Addr: ":" + addr}
	win.server.ListenAndServe()
	//http.ListenAndServe(":"+addr, nil)
}
func (w *window) startChrome(url string) {
	commandReader := bytes.NewBuffer([]byte{})
	fmt.Println("uri", url)
	go func() {
		defer w.Close()
		bash, args := GetLocalChromeBash(w.x, w.y, w.w, w.h, url, w.staticPath, w.args)
		if bash == "" {
			uri := fmt.Sprintf("http://localhost:%s/%s/%s", w.addr, path.Base(w.staticPath), NOT_FOUND_CHROME_HTML_NAME)
			openDefaultWebView(uri)
			time.Sleep(time.Second * 5)
			return
		}
		cmd := exec.Command(bash, args...)
		cmd.Stdout = commandReader
		cmd.Start()
		cmd.Wait()
	}()
	commandReader.Next(10)
}
func (w *window) Close() {
	w.server.Shutdown(context.Background())
	w.closeChannel <- 1
}

func (w *window) Navigation(htmlPath string) {
	initChannel := make(chan int, 1)
	go func() {
		if !w.inited {
			w.inited = true
			w.startChrome(fmt.Sprintf("http://localhost:%s/%s/%s", w.addr, path.Base(w.staticPath), DEFAULT_HTML_NAME))
		}
		if strings.HasPrefix(htmlPath, "http://") || strings.HasPrefix(htmlPath, "https://") {
			w.currentUrl = htmlPath
		} else {
			w.currentUrl = fmt.Sprintf("http://localhost:%s/%s/%s", w.addr, path.Base(w.staticPath), htmlPath)
		}
		for w.wl == nil {
			time.Sleep(time.Second)
		}
		// if w.wl == nil {
		// 	<-w.wlWaitChannel
		// }
		w.wl.navigation(`document.getElementById("iframe").src="`+w.currentUrl+`"`, func() {
			initChannel <- 1
		})

	}()
	<-initChannel
}

func (s *window) BindWithName(key string, obj interface{}) {
	if s.wl != nil {
		s.wl.putRequestHandler(key, obj)
	}
}
func (s *window) Bind(obj interface{}) {
	if s.wl != nil {
		s.wl.putRequestHandler("", obj)
	}
}
func (s *window) RemoveBind(key string) {
	if s.wl != nil {
		s.wl.removeRequestHandler(key)
	}
}

func (s *window) Eval(js string, f interface{}) {
	s.wl.eval(js, f)
}
