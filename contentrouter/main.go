package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"context"

	"io/ioutil"

	"cloud.google.com/go/storage"
	firebase "firebase.google.com/go"
	"github.com/gorilla/mux"
)

var (
	bucket       string
	firebasepath string
	gcspath      string
	redirectpath string
)

func main() {

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	bucket = os.Getenv("BUCKET")
	if bucket == "" {
		log.Printf("BUCKET not set")
		os.Exit(1)
	}
	firebasepath = os.Getenv("FIREBASEPATH")
	if firebasepath == "" {
		log.Printf("FIREBASEPATH default to /")
		firebasepath = "/"
	}
	gcspath = os.Getenv("GCSPATH")
	if gcspath == "" {
		log.Printf("GCSPATH default to /")
		gcspath = "/"
	}
	redirectpath = os.Getenv("REDIRECTPATH")
	if redirectpath == "" {
		redirectpath = "/"
	}

	r := mux.NewRouter()
	r.HandleFunc("/config", ConfigHandler).Methods("GET")
	r.HandleFunc(`/{route:[a-zA-Z0-9\.=\-\/]+}`, ContentRouter).Methods("GET")
	http.Handle("/", r)

	http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
}

// ContentRouter performs an auth check and routes to content
func ContentRouter(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	vars := mux.Vars(r)
	route := vars["route"]
	//log.Printf("requested route: %s", route)

	// TODO make middleware
	// check session
	session, err := r.Cookie("__session")
	if err == nil {
		//log.Printf("session cookie: %s", session)
		app, err := firebase.NewApp(ctx, nil)
		if err != nil {
			log.Println("couldn't get a Firebase client")
			http.Error(w, fmt.Sprintf("this was a problem: %v", err), 500)
			return
		}
		auth, err := app.Auth(ctx)
		if err != nil {
			log.Println("couldn't get a Firebase Auth client")
			http.Error(w, fmt.Sprintf("this was a problem: %v", err), 500)
			return
		}
		_, err = auth.VerifySessionCookieAndCheckRevoked(ctx, session.Value)
		if err != nil {
			log.Printf("session cookie unable to be verified: %v", err)
			w.WriteHeader(http.StatusForbidden)
			http.Redirect(w, r, redirectpath, http.StatusSeeOther)
		}
		//log.Printf("token (from cookie): %s", t.Subject)

		serveContent(ctx, w, route)
		return
	}

	// check token query param
	token := r.URL.Query().Get("token")
	if token == "" { // no token == not authenticated
		//w.WriteHeader(http.StatusForbidden)
		http.Redirect(w, r, redirectpath, http.StatusSeeOther)
	} else {
		// check validity of token
		app, err := firebase.NewApp(ctx, nil)
		if err != nil {
			log.Println("couldn't get a Firebase client")
			http.Error(w, fmt.Sprintf("this was a problem: %v", err), 500)
			return
		}
		auth, err := app.Auth(ctx)
		if err != nil {
			log.Println("couldn't get a Firebase Auth client")
			http.Error(w, fmt.Sprintf("this was a problem: %v", err), 500)
			return
		}
		_, err = auth.VerifyIDToken(ctx, token)
		if err != nil {
			log.Printf("invalid token: %v", err)
			http.Error(w, fmt.Sprintf("this was a problem: %v", err), 500)
			return
		}
		//log.Printf("t: %v", t.Subject)

		// set a session cookie
		maxduration := time.Minute * 30

		sessioncookie, err := auth.SessionCookie(ctx, token, maxduration)
		if err != nil {
			log.Printf("unable to create session cookie: %v", err)
		} else {
			cookie := &http.Cookie{
				Name:   "__session",
				Value:  sessioncookie,
				MaxAge: int(maxduration.Seconds()),
			}
			http.SetCookie(w, cookie)
		}

		serveContent(ctx, w, route)
		return
	}
}

// serveContent returns content
func serveContent(ctx context.Context, w http.ResponseWriter, route string) {
	filebytes, contenttype, err := getFileAtRoute(ctx, route)
	if err != nil {
		w.Header().Add("content-type", "text/plain")
		fmt.Fprintf(w, "couldn't find %s\n", route)
		return
	}
	// set cache control to 5 min for both browser and CDN (max-age)
	// and 10 min for CDN-caching
	w.Header().Add("Cache-Control", "public, max-age=300, s-maxage=600")
	w.Header().Add("content-type", contenttype)
	//log.Printf("content-type: %s", contenttype)
	w.Write(filebytes)
}

// getFileAtRoute retrieves a GCS bucket file given a filepath
func getFileAtRoute(ctx context.Context, filepath string) ([]byte, string, error) {
	var filedata []byte
	var contenttype string

	// need to expose as variable
	filepath = strings.Replace(filepath, firebasepath, gcspath, -1)

	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Printf("err storage client: %v", err)
		return filedata, contenttype, err
	}
	defer client.Close()

	rc, err := client.Bucket(bucket).Object(filepath).NewReader(ctx)
	if err != nil {
		log.Printf("err bucket retrieval: %v", err)
		return filedata, contenttype, err
	}
	defer rc.Close()

	contenttype = rc.Attrs.ContentType

	filebytes, err := ioutil.ReadAll(rc)
	if err != nil {
		return filebytes, contenttype, err
	}
	return filebytes, contenttype, err
}

// ConfigHandler describes the configuration
func ConfigHandler(w http.ResponseWriter, r *http.Request) {
	config := fmt.Sprintf("REDIRECT: %s ; GCSPATH: %s ; FIREBASEPATH: %s ; BUCKET %s",
		redirectpath, gcspath, firebasepath, bucket,
	)
	w.Header().Set("content-type", "text/plain")
	w.Write([]byte(config))
}
