package main

import (
	"encoding/json"
	"fmt"
	"github.com/zmb3/spotify"
	"log"
	"net/http"
	"strings"
)

const redirectURI = "http://10.227.64.187:8080/callback"

var html = `
<br/>
<a href="/player/play">Play</a><br/>
<a href="/player/pause">Pause</a><br/>
<a href="/player/next">Next track</a><br/>
<a href="/player/previous">Previous Track</a><br/>
<a href="/player/shuffle">Shuffle</a><br/>

`

var (
	auth  = spotify.NewAuthenticator(redirectURI, spotify.ScopeUserReadCurrentlyPlaying, spotify.ScopeUserReadPlaybackState, spotify.ScopeUserModifyPlaybackState)
	ch    = make(chan *spotify.Client)
	state = "abc123"
)

type Result struct {
	Val        string
	NeedUpdate bool
}

func main() {
	var client *spotify.Client
	auth.SetAuthInfo("cf33e39b15854c21bd8851e05fa1a1c1", "fe9d073853ef439eaffeacdbf3e63e0b")
	lastTrack := ""
	http.HandleFunc("/callback", completeAuth)
	http.HandleFunc("/player/", func(w http.ResponseWriter, r *http.Request) {
		action := strings.TrimPrefix(r.URL.Path, "/player/")
		fmt.Println("Got request for:", action)
		var err error
		var currentPlaying *spotify.CurrentlyPlaying
		var track string
		result := Result{
			Val:        "",
			NeedUpdate: true,
		}
		w.Header().Set("Content-Type", "text/html")
		switch action {
		case "current":
			currentPlaying, err = client.PlayerCurrentlyPlaying()
			if err != nil || currentPlaying == nil || currentPlaying.Item == nil {
				log.Println("Get current playing failed, get empty result")
				track = ""
				break
			}
			track = fmt.Sprintf("♫ %s-%s", currentPlaying.Item.Name, currentPlaying.Item.Artists[0].Name)
			result.Val = track
		}
		fmt.Printf("lastTrack: %s, currentTrack: %s", lastTrack, track)
		if lastTrack == track {
			result.NeedUpdate = false
		}
		lastTrack = track
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(result)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Got request for:", r.URL.String())
	})

	go func() {
		url := auth.AuthURL(state)
		fmt.Println("Please log in to Spotify by visiting the following page in your browser:", url)
		// wait for auth to complete
		client = <-ch

		user, err := client.CurrentUser()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("You are logged in as:", user.DisplayName)
	}()

	http.ListenAndServe(":8080", nil)
}

func completeAuth(w http.ResponseWriter, r *http.Request) {
	tok, err := auth.Token(state, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}
	if st := r.FormValue("state"); st != state {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", st, state)
	}
	// use the token to get an authenticated client
	client := auth.NewClient(tok)
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "Login Completed!"+html)
	ch <- &client
}
