package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/antonholmquist/jason"
	"golang.org/x/oauth2"
	"log"
	"net"
	"net/http"
	"os"
	"html/template"
)

type AccessToken struct {
	Access_token string
	Expires_in   int64
}

type User struct {
	Name string
	Img  string
	ID   string
}

func GetIP() string {

	host, _ := os.Hostname()
	fmt.Println(host)
	addrs, _ := net.LookupIP(host)
	for _, addr := range addrs {
		if ipv4 := addr.To4(); ipv4 != nil {
			fmt.Println("IPv4: ", ipv4)
		}
	}

	ip := string(addrs[0])
	return ip
}

func readHttpBody(response *http.Response) string {
	fmt.Println("Reading body")

	buf := bytes.Buffer{}
	_, err := buf.ReadFrom(response.Body)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(buf.String())

	return buf.String()
}

func GetAccessToken(client_id string, code string, secret string, callbackUri string) AccessToken {
	fmt.Println("GetAccessToken")
	//https://graph.facebook.com/oauth/access_token?client_id=YOUR_APP_ID&redirect_uri=YOUR_REDIRECT_URI&client_secret=YOUR_APP_SECRET&code=CODE_GENERATED_BY_FACEBOOK
	response, err := http.Get("https://graph.facebook.com/oauth/access_token?client_id=" +
		client_id + "&redirect_uri=" + callbackUri +
		"&client_secret=" + secret + "&code=" + code)

	if err == nil {
		auth := readHttpBody(response)

		var token AccessToken

		err := json.Unmarshal([]byte(auth), &token)
		if err != nil {
			fmt.Println(err)
		}

		return token
	}

	var token AccessToken

	return token
}

func Home(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// generate loginURL
	fbConfig := &oauth2.Config{
		// ClientId: FBAppID(string), ClientSecret : FBSecret(string)
		// Example - ClientId: "1234567890", ClientSecret: "red2drdff6e2321e51aedcc94e19c76ee"

		ClientID:     "528188074643591", // change this to yours
		ClientSecret: "32d9ce18352796eb53014cdf17f8ce73",
		RedirectURL:  "https://b742c42f.ngrok.io/chat", // change this to your webserver adddress
		//87c947e7.ngrok.io
		Scopes:       []string{"email", "user_birthday", "user_location", "user_photos", "public_profile"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://www.facebook.com/dialog/oauth",
			TokenURL: "https://graph.facebook.com/oauth/access_token",
		},
	}
	url := fbConfig.AuthCodeURL("")

	// Home page will display a button for login to Facebook

	w.Write([]byte("<html><title>Golang Login Facebook Example</title> <body> <a href='" + url + "'><button>Login with Facebook!</button> </a> </body></html>"))
}

func FBLogin(w http.ResponseWriter, r *http.Request) {
	// grab the code fragment
	var users User

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	code := r.FormValue("code")
	//408565436483487
	//650844db51055f30e7e3721e7396f97c
	// http://192.168.0.125
	ClientId := "528188074643591" // change this to yours
	ClientSecret := "32d9ce18352796eb53014cdf17f8ce73"
	RedirectURL := "https://b742c42f.ngrok.io/chat"
	//87c947e7.ngrok.io
	accessToken := GetAccessToken(ClientId, code, ClientSecret, RedirectURL)

	response, err := http.Get("https://graph.facebook.com/me?fields=id,name,birthday,email&access_token=" + accessToken.Access_token)
	//response, err := http.Get("https://graph.facebook.com/me?access_token=" + accessToken.Access_token)

	if err != nil {
		w.Write([]byte(err.Error()))
	}

	str := readHttpBody(response)

	user, err := jason.NewObjectFromBytes([]byte(str))
	if err != nil {
		fmt.Println(err)
	}

	id, _ := user.GetString("id")
	email, _ := user.GetString("email")
	bday, _ := user.GetString("birthday")
	name, _ := user.GetString("name")
	users.Name = name

	fmt.Println("id:", id, "email", email, "birthday", bday, "name", users.Name)

	//w.Write([]byte(fmt.Sprintf("Username %s ID is %s and birthday is %s and email is %s<br>", users.Name, id, bday, email)))

	img := "https://graph.facebook.com/" + id + "/picture?width=180&height=180"

	fmt.Println(img)
	//w.Write([]byte("Photo is located at " + img + "<br>"))
	// see https://www.socketloop.com/tutorials/golang-download-file-example on how to save FB file to disk

	//w.Write([]byte("<img src='" + img + "'>"))
	log.Println(r.URL)
	if r.URL.Path != "/chat" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	//http.ServeFile(w, r, "home.html")

	tmpl, _ := template.ParseFiles("home.html")
	tmpl.Execute(w, struct {
		ID   string
		Img  string
		Name string
	}{id, img, name,})
}

var addr = flag.String("addr", ":8080", "http service address")

func serveHome(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	if r.URL.Path != "/chat" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "home.html")
}

func main() {

	//mux := http.NewServeMux()
	http.HandleFunc("/", Home)
	http.HandleFunc("/chat", FBLogin)

	//addr := http.ListenAndServe(":8080", mux)
	//if addr != nil {
	//	fmt.Println(addr)
	//}

	GetIP()

	flag.Parse()
	hub := newHub()
	go hub.run()
	//http.HandleFunc("/chat", serveHome)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
