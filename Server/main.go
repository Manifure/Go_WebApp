package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/gomarkdown/markdown"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/russross/blackfriday/v2"
	"golang.org/x/time/rate"
	"html/template"
	"log"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"
)

var defaultLogin = "admin"
var defaultPassword = "admin"

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "1"
	dbname   = "postgres"
)

type Article struct {
	Id        uint16
	Title     string
	Anons     string
	Full_text template.HTML
}

type Str struct {
	Posts []Article
	Pg    []int
}

var posts = []Article{}
var showPost = Article{}

var limiter = rate.NewLimiter(rate.Limit(100), 100) // 100 запросов в секунду
var lastExceeded time.Time
var mu sync.Mutex

func rateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		if time.Since(lastExceeded) < 10*time.Second {
			mu.Unlock()
			http.Error(w, "429 Too Many Requests", http.StatusTooManyRequests)
			return
		}
		mu.Unlock()

		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()
		if err := limiter.Wait(ctx); err != nil {
			mu.Lock()
			lastExceeded = time.Now()
			mu.Unlock()
			log.Println("Rate limit exceeded")
			http.Error(w, "429 Too Many Requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func index(w http.ResponseWriter, r *http.Request) {
	var page float64
	var pg = []int{}
	num := 1

	tmp, _ := r.URL.Query()["page"]

	if len(tmp) > 0 {
		num, _ = strconv.Atoi(tmp[0])
	}

	//
	t, err := template.ParseFiles("templates/index.html", "templates/header.html", "templates/footer.html")
	if err != nil {
		log.Fatal(err)
	}

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}

	res, err := db.Query("SELECT COUNT(*) FROM articles")
	if err != nil {
		log.Fatal(err)
	}
	defer res.Close()

	for res.Next() {
		err = res.Scan(&page)
		if err != nil {
			fmt.Println(err)
		}
	}

	page = math.Ceil(page / 3)
	for i := 1; i <= int(page); i++ {
		pg = append(pg, i)
	}

	res, err = db.Query(fmt.Sprintf("select * from articles ORDER BY id ASC LIMIT 3 OFFSET %d;", num*3-3))
	if err != nil {
		log.Fatal(err)
	}
	defer res.Close()

	posts = []Article{}

	for res.Next() {
		var post Article
		err = res.Scan(&post.Id, &post.Title, &post.Anons, &post.Full_text)
		if err != nil {
			fmt.Println(err)
			continue
		}
		//fmt.Println(fmt.Sprintf("post: %s with id: %d", post.Title, post.Id))
		posts = append(posts, post)
	}

	t.ExecuteTemplate(w, "index", Str{posts, pg})
}

func admin(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/admin.html", "templates/header.html", "templates/footer.html")

	if err != nil {
		log.Fatal(err)
	}

	t.ExecuteTemplate(w, "admin", nil)
}

func login(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/login.html", "templates/header.html", "templates/footer.html")

	if err != nil {
		log.Fatal(err)
	}

	t.ExecuteTemplate(w, "login", nil)
}

func check_login(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("login") == defaultLogin && r.FormValue("pas") == defaultPassword {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

func save_article(w http.ResponseWriter, r *http.Request) {
	title := r.FormValue("title")
	anons := r.FormValue("anons")
	full_text := r.FormValue("full_text")

	if title == "" || anons == "" || full_text == "" {
		fmt.Fprintf(w, "Error: title or anons or full_text is empty.")
	} else {

		psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
		db, err := sql.Open("postgres", psqlInfo)

		if err != nil {
			log.Fatal(err)
		}

		defer db.Close()

		mdText := markdown.ToHTML([]byte(full_text), nil, nil)

		insert, err := db.Query(fmt.Sprintf("INSERT INTO articles (title, anons, full_text) VALUES ('%s', '%s', '%s')", title, anons, mdText))
		if err != nil {
			log.Fatal(err)
		}

		defer insert.Close()

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func show_post(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	t, err := template.ParseFiles("templates/show.html", "templates/header.html", "templates/footer.html")
	if err != nil {
		log.Fatal(err)
	}

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	res, err := db.Query(fmt.Sprintf("SELECT * FROM articles WHERE id = %s", vars["id"]))
	if err != nil {
		log.Fatal(err)
	}
	defer res.Close()

	showPost = Article{}
	for res.Next() {
		var post Article
		err = res.Scan(&post.Id, &post.Title, &post.Anons, &post.Full_text)
		if err != nil {
			log.Fatal(err)
		}

		// Преобразуем Markdown в HTML
		post.Full_text = template.HTML(blackfriday.Run([]byte(post.Full_text)))
		showPost = post
	}

	// Преобразуем Full_text в template.HTML
	showPost.Full_text = template.HTML(showPost.Full_text)

	// Передаем данные в шаблон
	err = t.ExecuteTemplate(w, "show", showPost)
	if err != nil {
		log.Fatal(err)
	}
}

func handleFunc() {
	rtr := mux.NewRouter()

	rtr.Use(rateLimit)

	rtr.HandleFunc("/", index).Methods("GET")
	rtr.HandleFunc("/admin", admin).Methods("GET")
	rtr.HandleFunc("/login", login).Methods("POST", "GET")
	rtr.HandleFunc("/check_login", check_login).Methods("POST")
	rtr.HandleFunc("/save_article", save_article).Methods("POST")
	rtr.HandleFunc("/post/{id:[0-9]+}", show_post).Methods("GET")

	http.Handle("/", rtr)
	http.Handle("/src/", http.StripPrefix("/src/", http.FileServer(http.Dir("src"))))

	err := http.ListenAndServe(":8888", nil)

	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	handleFunc()
}
