package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Json struct {
	Access_token string `json:access_token`
	Token_type   string `json:token_type`
	Scope        string `json:scope`
}

func main() {
	http.HandleFunc("/", Index)
	http.HandleFunc("/page.html", Redirect)
	log.Print(http.ListenAndServe(":8080", nil))
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

func Get(URL string, jsonToken Json) (string, error) {
	client := &http.Client{}
	val := url.Values{}
	//val.Add("access_token", jsonToken.Access_token)
	req, err := http.NewRequest("GET", URL, strings.NewReader(val.Encode()))
	if err != nil {
		return "err", err
	}
	req.Header.Set("Authorization", jsonToken.Token_type+" "+jsonToken.Access_token)
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
func Redirect(w http.ResponseWriter, r *http.Request) {
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

	type USER struct {
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

	//var data []USER
	var data []REPO
	str := ""
	var reb string

	var user USER
	reb, err = Get("https://api.github.com/user", jsonToken)
	if err != nil {
		return
	}
	_ = json.Unmarshal([]byte(reb), &user)

	reb, err = Get("https://api.github.com/users/"+user.Login+"/repos", jsonToken)
	if err != nil {
		return
	}
	_ = json.Unmarshal([]byte(reb), &data)

	var dates []time.Time
	for _, d := range data {
		//str += d.Name + "<br>"
		var commits []COMIT
		reb, err = Get("https://api.github.com/repos/"+user.Login+"/"+d.Name+"/commits", jsonToken)
		if err != nil {
			return
		}
		_ = json.Unmarshal([]byte(reb), &commits)
		for _, c := range commits {
			if c.Commit.Committer.Name == user.Login {
				dates = append(dates, c.Commit.Committer.Date)
				//str += " " + c.Commit.Committer.Date.Format("20060102") + "<br>"
			}
		}
	}
	sort.Slice(dates, func(i, j int) bool {
		return dates[i].Before(dates[j])
	})
	cnt := 0
	for t := time.Date(2019, 12, 1, 0, 0, 0, 0, time.UTC); t.Before(time.Date(2021, 9, 1, 0, 0, 0, 0, time.UTC)); t = t.AddDate(0, 0, 1) {
		//fmt.Println(dates[cnt].Truncate(24 * time.Hour))
		num := 0
		for cnt < len(dates) && dates[cnt].Truncate(24*time.Hour).Equal(t) {
			num++
			cnt++
		}
		if num > 0 {
			str += t.Format("2006/01/02") + ":" + strconv.Itoa(num) + "<br>"
		}
		fmt.Println(t, num)
	}
	fmt.Fprint(w, "<html>"+user.Login+"<br><br>"+str+"</html>")
}
