package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type USER struct {
	Id     int
	Name   string
	Github *string
}

type Json struct {
	Access_token string `json:access_token`
	Token_type   string `json:token_type`
	Scope        string `json:scope`
}

func main() {
	http.HandleFunc("/", Index)
	http.HandleFunc("/page.html", Redirect)
	http.HandleFunc("/SetSchedule", SetSchedule)
	log.Print(http.ListenAndServe(":8080", nil))
}

func SetSchedule(w http.ResponseWriter, r *http.Request) {

	db, err := sql.Open("mysql", "suzaku:1212@tcp(127.0.0.1:3306)/calendar")
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()
	date := r.FormValue("date")
	content := r.FormValue("content")
	id, err := r.Cookie("id")
	if err != nil {
		fmt.Fprint(w, "err")
		panic(err.Error())
	}
	if _, err = db.Query("insert into schedules values ('" + id.Value + "','" + date + "','" + content + "')"); err != nil {
		panic(err.Error())
	}

}

func Index(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("template/index.html")
	if err != nil {
		log.Fatalf("template error: %v", err)
	}
	if err := t.Execute(w, struct{}{}); err != nil {
		log.Printf("failed to execute template: %v", err)
	}
}

func Get(URL string, access_token string, token_type string) (string, error) {
	client := &http.Client{}
	val := url.Values{}
	//val.Add("access_token", jsonToken.Access_token)
	req, err := http.NewRequest("GET", URL, strings.NewReader(val.Encode()))
	if err != nil {
		return "err", err
	}
	req.Header.Set("Authorization", token_type+" "+access_token)
	resp, err := client.Do(req)
	if err != nil {
		return "err", err
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "err", err
	}
	defer resp.Body.Close()
	return string(b), nil
}

func MakeRandomStr(digit uint32) (string, error) {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	// 乱数を生成
	b := make([]byte, digit)
	if _, err := rand.Read(b); err != nil {
		return "", errors.New("unexpected error...")
	}

	// letters からランダムに取り出して文字列を生成
	var result string
	for _, v := range b {
		// index が letters の長さに収まるように調整
		result += string(letters[int(v)%len(letters)])
	}
	return result, nil
}

func Redirect(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("mysql", "suzaku:1212@tcp(127.0.0.1:3306)/calendar")
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	var access_token string
	var token_type string
	cookie, err := r.Cookie("session_token")
	if err == nil {
		fmt.Println("cookie")
		session_token := cookie.Value
		row, err := db.Query("select * from sessions where sessiontoken='" + session_token + "';")
		if err != nil {
			fmt.Println("err!")
			panic(err.Error())
		}
		defer row.Close()
		var st string
		if row.Next() {
			err = row.Scan(&st, &access_token, &token_type)
			fmt.Println(access_token)
			fmt.Println(token_type)
			if err != nil {
				fmt.Println("err!!")
				panic(err.Error())
			}
		}
	} else {
		code := r.FormValue("code")
		if code == "" {
			fmt.Fprint(w, "err")
			return
		}
		client := &http.Client{}
		URL := "https://github.com/login/oauth/access_token"
		val := url.Values{}
		val.Add("client_id", "699604ddf0252d9f1bac")
		val.Add("client_secret", os.Getenv("CliSecret"))
		val.Add("code", code)
		req, err := http.NewRequest("POST", URL, strings.NewReader(val.Encode()))
		if err != nil {
			fmt.Fprint(w, "err2")
			return
		}
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Add("Accept", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			fmt.Fprint(w, "err3")
			return
		}
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Fprint(w, "err4")
			return
		}
		var jsonToken Json
		err = json.Unmarshal([]byte(string(b)), &jsonToken)
		if err != nil {
			fmt.Fprint(w, "err5")
			return
		}
		defer resp.Body.Close()
		fmt.Println(jsonToken.Access_token, jsonToken.Token_type)
		access_token = jsonToken.Access_token
		token_type = jsonToken.Token_type
		row, err := db.Query("select sessiontoken from sessions;")
		if err != nil {
			panic(err.Error())
		}
		defer row.Close()
		var tokens []string
		for row.Next() {
			var st string
			err = row.Scan(&st)
			if err != nil {
				panic(err.Error())
			}
			tokens = append(tokens, st)
		}

		session_token, _ := MakeRandomStr(30)
		for {
			flg := false
			for _, st := range tokens {
				if st == session_token {
					flg = true
				}
			}
			if flg {
				session_token, _ = MakeRandomStr(30)
			} else {
				break
			}
		}
		_, err = db.Query("insert into sessions values ('" + session_token + "','" + access_token + "','" + token_type + "')")
		if err != nil {
			panic(err.Error())
		}
		cookie := &http.Cookie{
			Name:  "session_token",
			Value: session_token,
		}
		http.SetCookie(w, cookie)
	}

	type GithubUSER struct {
		Login string `json:"login"`
		//Avatar_url string `json:"avatar_url"`
	}
	type COMITER struct {
		Committer struct {
			Name string
			Date time.Time
		}
	}
	type COMIT struct {
		Sha    string
		Commit COMITER
	}

	type REPO struct {
		Name string
	}

	var data []GithubUSER
	//var data []REPO
	var reb string

	var user USER

	var github GithubUSER

	cookie, err = r.Cookie("id")
	if err == nil {
		fmt.Println("cookie!!")
		id := cookie.Value
		row, err := db.Query("select * from users where id=" + id + ";")
		if err != nil {
			fmt.Println("err!")
			panic(err.Error())
		}
		defer row.Close()
		if row.Next() {
			err = row.Scan(&user.Id, &user.Name, &user.Github)
			if err != nil {
				panic(err.Error())
			}
		}
	} else {
		reb, err = Get("https://api.github.com/user", access_token, token_type)
		if err != nil {
			panic(err.Error())
		}
		_ = json.Unmarshal([]byte(reb), &github)

		row, err := db.Query("select * from users where github='" + github.Login + "';")
		if err != nil {
			panic(err.Error())
		}
		defer row.Close()
		if row.Next() {
			if err := row.Scan(&user.Id, &user.Name, &user.Github); err != nil {
				panic(err.Error())
			}
		} else {
			rand.Seed(time.Now().UnixNano())
			id := rand.Intn(2000000000)
			fmt.Println("else")
			row, err := db.Query("select * from users where id=" + strconv.Itoa(id) + ";")
			if err != nil {
				fmt.Println("err!")
				panic(err.Error())
			}
			for row.Next() {
				id := rand.Intn(2000000000)
				row, err = db.Query("select * from users where id=" + strconv.Itoa(id) + ";")
				if err != nil {
					panic(err.Error())
				}
			}
			user.Id = id
			user.Name = github.Login
			user.Github = &github.Login
			_, err = db.Query("insert into users values ('" + strconv.Itoa(id) + "','" + github.Login + "','" + github.Login + "');")
			if err != nil {
				panic(err.Error())
			}
		}
		cookie := &http.Cookie{
			Name:  "id",
			Value: strconv.Itoa(user.Id),
		}
		http.SetCookie(w, cookie)
	}

	row, err := db.Query("select*from schedules where userid='" + strconv.Itoa(user.Id) + "';")
	if err != nil {
		fmt.Println("aaa")
		panic(err.Error())
	}
	defer row.Close()

	type Schedule struct {
		Id      int
		Name    string
		Date    string
		Content string
	}
	var schedules []Schedule
	for row.Next() {
		var sc Schedule
		if err = row.Scan(&sc.Id, &sc.Date, &sc.Content); err != nil {
			fmt.Println("date")
			panic(err.Error())
		}
		row2, err := db.Query("select name from users where id='" + strconv.Itoa(sc.Id) + "';")
		if err != nil {
			fmt.Println("aaa")
			panic(err.Error())
		}
		defer row2.Close()
		if !row2.Next() {
			panic("aa")
		}
		if err := row2.Scan(&sc.Name); err != nil {
			panic(err.Error())
		}
		schedules = append(schedules, sc)
		//str += sc.date + ":" + sc.content + "\n"
	}

	if user.Github == nil {
		fmt.Println("nil!")
	}
	reb, err = Get("https://api.github.com/users/"+*user.Github+"/following", access_token, token_type)
	if err != nil {
		fmt.Println("errrrrr")
		panic(err.Error())
	}
	_ = json.Unmarshal([]byte(reb), &data)
	for _, n := range data {
		row, err := db.Query("select name,date,content from schedules,users where id=userid and github='" + n.Login + "';")
		if err != nil {
			panic(err.Error())
		}
		defer row.Close()
		for row.Next() {
			var sc Schedule
			err = row.Scan(&sc.Name, &sc.Date, &sc.Content)
			if err != nil {
				panic(err.Error())
			}
			schedules = append(schedules, sc)
		}
	}
	//fmt.Fprint(w, "<html>"+string(reb)+"</html>")

	t, err := template.ParseFiles("template/main.html")
	if err != nil {
		log.Fatalf("template error: %v", err)
	}
	if err := t.Execute(w, schedules); err != nil {
		log.Printf("failed to execute template: %v", err)
	}
}
