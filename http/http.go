package http

import (
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aeytom/gometer/meter"
)

var addr *string
var resDir *string
var templ *template.Template

// allMeters - available meters
var allMeters = make(map[string]meter.GetSet)

// Flags …
func Flags() {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	addr = flag.String("addr", ":1718", "http service address")
	resDir = flag.String("root", filepath.Join(cwd, "htdocs"), "http resource directory")
}

// Add …
func Add(m meter.GetSet) {
	allMeters[m.ID()] = m
}

// RunServer …
func RunServer() {

	tplIndex, err := ioutil.ReadFile(filepath.Join(*resDir, "index.tpl.html"))
	if err != nil {
		panic(err)
	}
	templ = template.Must(template.New("index").Parse(string(tplIndex)))

	mux := http.NewServeMux()
	mux.Handle("/resource/", logHandler(wrapHandler(http.FileServer(http.Dir(*resDir)))))
	mux.Handle("/favicon.ico", logHandler(http.FileServer(http.Dir(*resDir))))
	mux.Handle("/api/", logHandler(http.StripPrefix("/api/", http.HandlerFunc(api))))
	mux.Handle("/api", logHandler(http.HandlerFunc(list)))
	mux.Handle("/", logHandler(http.HandlerFunc(index)))
	if err := http.ListenAndServe(*addr, mux); err != nil {
		panic(err)
	}
}

func index(w http.ResponseWriter, req *http.Request) {

	type mdata struct {
		M        meter.GetSet
		ID       string
		Label    string
		Selected bool
		Value    float32
	}

	var cm meter.GetSet
	var vMsg string
	fMeter := req.FormValue("meter")
	cm = allMeters[fMeter]

	if req.Method == http.MethodPost {
		if _, err := setMeterValue(req, cm); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			vMsg = err.Error()
		}
	}

	md := make([]mdata, 0, len(allMeters))
	for _, m := range allMeters {
		if cm == nil {
			cm = m
			fMeter = cm.ID()
		}
		d := mdata{
			M:        m,
			ID:       m.ID(),
			Label:    m.Label(),
			Selected: fMeter == m.ID(),
			Value:    m.Get(),
		}
		md = append(md, d)
	}

	data := struct {
		Action string
		M      meter.GetSet
		Meters []mdata
		Meter  string
		Value  string
		Error  string
	}{
		Action: req.URL.String(),
		Meters: md,
		Meter:  cm.ID(),
		M:      cm,
		Value:  fmt.Sprintf("%.3f", cm.Get()),
		Error:  vMsg,
	}
	if err := templ.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

//
func list(w http.ResponseWriter, req *http.Request) {
	cacheHeader(w)

	link := *req.URL
	link.Scheme = "http"
	link.Host = req.Host
	link.RawQuery = ""
	link.Fragment = ""
	var rtext string
	for _, m := range allMeters {
		link.Path = req.URL.Path + "/" + url.PathEscape(m.ID())
		w.Header().Add("Link", fmt.Sprintf("<%s>; rel=alternate", link.String()))
		rtext += fmt.Sprintf("%s\n", link.String())
	}

	_, err := io.WriteString(w, rtext)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

//
func api(w http.ResponseWriter, req *http.Request) {
	rm := strings.Split(req.URL.Path, "/")
	if rm == nil {
		http.NotFound(w, req)
		return
	}

	meter, ok := allMeters[rm[0]]
	if !ok {
		http.NotFound(w, req)
		return
	}

	var rtext string

	if len(rm) < 2 {
		// list options
		link := *req.URL
		link.Scheme = "http"
		link.Host = req.Host
		link.RawQuery = ""
		link.Fragment = ""
		for _, o := range []string{"value", "label", "unit"} {
			link.Path = "/api/" + req.URL.Path + "/" + o
			w.Header().Add("Link", fmt.Sprintf("<%s>; rel=alternate", link.String()))
			rtext += fmt.Sprintf("%s\n", link.String())
		}
	} else {
		switch {
		case req.Method == http.MethodPost:
			if val, err := setMeterValue(req, meter); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
			} else {
				rtext = fmt.Sprintf("%.3f", val)
			}

		// Handle GET
		default:
			switch {
			case rm[1] == "label":
				rtext = meter.Label()
			case rm[1] == "unit":
				rtext = meter.Unit()
			default:
				rtext = fmt.Sprintf("%.3f", meter.Get())
			}
		}
	}
	cacheHeader(w)
	_, err := io.WriteString(w, rtext)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// logHandler …
func logHandler(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := r.URL.String()
		log.Printf("%s \"%s\" %s \"%s\"", r.Method, u, r.Proto, r.UserAgent())
		h.ServeHTTP(w, r)
	}
}

// cacheHeader …
func cacheHeader(w http.ResponseWriter) {
	w.Header().Add("Cache-Control", "must-revalidate, private, max-age=20")
}

// setMeterValue …
func setMeterValue(req *http.Request, meter meter.GetSet) (float32, error) {

	if err := req.ParseForm(); err != nil {
		return 0, err
	}

	rv := req.PostForm.Get("value")
	if rv == "" {
		return 0, errors.New("empty value")
	}

	if val, err := strconv.ParseFloat(rv, 32); err != nil {
		return 0, err
	} else if float32(val) <= meter.Get() {
		return 0, errors.New("value to small")
	} else {
		log.Printf("Set value %.3f to meter '%s'", val, meter.ID())
		meter.Set(float32(val))
		return meter.Get(), nil
	}
}
