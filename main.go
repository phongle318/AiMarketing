package main

import (
	log "github.com/Sirupsen/logrus"
	"net/http"
	"errors"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"fmt"
	"encoding/json"
	"time"
	"html/template"
	"strconv"
	"os"
	"github.com/gocarina/gocsv"
	"io"
	"github.com/phongle318/AiMarketing/config"
)
const (
	EMP_STRING = ""
	ResultFolder string = "./User/result"
)
var conFptAi *sqlx.DB
var conQna *sqlx.DB

func init() {


	if err := config.LoadFromEnv(); err != nil {
		log.Fatal("load config failed: ", err)
	}
	fptSource := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", config.Bot.DbUsername, config.Bot.DbPassword, config.Bot.DbHost, config.Bot.DbPort, config.Bot.DbNameFptAi)
	qnaSource := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", config.Bot.DbUsername, config.Bot.DbPassword, config.Bot.DbHost, config.Bot.DbPort, config.Bot.DbNameFptAi)
	conFptAi = sqlx.MustConnect("mysql", fptSource)
	conQna = sqlx.MustConnect("mysql", qnaSource)
}

type User struct {
	Email    string  `db:"email" json:"email" csv:"email"`
	Name    string  `db:"name" json:"name" csv:"name"`
	CreatedTime  string `db:"created_time" json:"created_time" csv:"created_time"`
	TotalBot  string `db:"total_bot" json:"total_bot" csv:"total_bot"`
}

// SearchDate : searching date betweeen start date and end date
type SearchDate struct {
	StartDate string `db:startDate`
	EndDate   string `db:endDate`
}


func main() {
	port := 1304
	r := mux.NewRouter()

	r.HandleFunc("/", ViewStatic)
	r.HandleFunc("/user", GetUser).Methods("GET")
	r.HandleFunc("/userobot", GetUserWithoutBot).Methods("GET")

	log.Info("Listening at port:", port)
	//start server
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(port), r))
}

func GetUser(w http.ResponseWriter, r *http.Request) {
	log.Infof("All right!!!")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	searchDate, err := validateParam(r)
	if err != nil {
		ResponseError(w, err, http.StatusBadRequest)
		log.Errorf("Error happen in validateParam : %s", err)
		return
	}
	log.Info("QueryUser")
	user, err := QueryUser(searchDate)
	if err != nil {
		ResponseError(w, err, http.StatusInternalServerError)
		log.Errorf("Error happen in QueryUser : %s", err)
		return
	}
	log.Infof("%+v\n", user)
	//ResponseJSON(w, user)
	//
	ServeFile(user, w, r)
}

func GetUserWithoutBot(w http.ResponseWriter, r *http.Request) {
	log.Info("GetUserWithoutBot")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	searchDate, err := validateParam(r)
	if err != nil {
		ResponseError(w, err, http.StatusBadRequest)
		log.Errorf("Error happen in validateParam : %s", err)
		return
	}

	user, err := QueryUserWithoutBot(searchDate)
	if err != nil {
		ResponseError(w, err, http.StatusInternalServerError)
		return
	}

	//ResponseJSON(w, user)
	ServeFile(user, w, r)
}


func QueryUser(date SearchDate) ([]User, error){
	log.Infof("QueryUser!!!")
	query := `select user.email, user.name, user.created_at as created_time, (bot.user_id) as total_bot from user inner join bot on user.id = bot.user_id  where (DATE_FORMAT(user.created_at, '%Y-%m-%d') between ? and ?) group by bot.user_id ORDER BY bot.created_at DESC;`
	users := []User{}
	log.Info("date.StartDate: ", date.StartDate)
	log.Info("date.EndDate: ", date.EndDate)
	_, err := conQna.Exec(`SET sql_mode =''`)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	rows, err := conQna.Queryx(query, date.StartDate, date.EndDate)
	if err != nil {
		log.Error(err)
		return users, err
	}
	log.Info("rows", rows)
	for rows.Next() {
		user := User{}
		if err = rows.StructScan(&user); err != nil {
			log.Error(err)
			continue
		}
		users = append(users, user)
	}
	return users, err
}

func QueryUserWithoutBot(date SearchDate) ([]User, error){
	log.Infof("QueryUserWithoutBot!!!")
	query := `select user.email, user.name, user.created_at as created_time from user left join bot on user.id = bot.user_id where bot.user_id IS NULL and (DATE_FORMAT(bot.created_at, '%Y-%m-%d') between ? and ?)`
	users := []User{}
	rows, err := conQna.Queryx(query, date.StartDate, date.EndDate)
	if err != nil {
		log.Error(err)
		return users, err
	}
	log.Info("rows", rows)
	for rows.Next() {
		user := User{}
		if err = rows.StructScan(&user); err != nil {
			log.Error(err)
			continue
		}
		users = append(users, user)
	}
	return users, err
}

func ViewStatic(w http.ResponseWriter, r *http.Request) {
	log.Info("ViewStatic")
	 w.Header().Set("Access-Control-Allow-Origin", "*")
	t, _ := template.New("marketing.html").Delims("{[{", "}]}").ParseFiles("marketing.html")
	t.Execute(w, nil)
}


func ResponseError(w http.ResponseWriter, err error, code int) {
	log.Error("Error: ", err)
	w.WriteHeader(code)
	ResponseJSON(w, map[string]interface{}{"error": err.Error()})
}

func ResponseSuccess(w http.ResponseWriter, args ...int) {
	if args != nil {
		w.WriteHeader(args[0])
	}
	ResponseJSON(w, map[string]interface{}{"success": 1})
}

func ResponseJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	respBody, err := json.Marshal(v)
	if err != nil {
		log.Error("Error in responseJSON: ", err.Error())
		respBody, _ = json.Marshal(map[string]interface{}{"error": err.Error()})
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(respBody)
	} else {
		w.Write(respBody)
	}
}

func CreateDirIfNotExist(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			panic(err)
		}
	}
}


func validateParam(r *http.Request) (SearchDate, error) {
	startdate, enddate := r.FormValue("startDate"), r.FormValue("endDate")
	searchDate, err := validateDateParams(startdate, enddate)
	if err != nil {
		log.Error("Error validateDateParams: ", err)
		return searchDate, err
	}

	return searchDate, nil
}

func validateDateParams(startdate, enddate string) (SearchDate, error) {
	searchDate := SearchDate{}
	if startdate == EMP_STRING {
		log.Info("OK strart date")
		log.Info("config.StartDateDefault:", config.Bot.StartDateDefault)
		startdate = "2018-04-14"
	}
	if enddate == EMP_STRING {
		log.Info("OK end date")
		enddate = "2018-05-15"
	}

	//Convert Start Date
	start, err := time.Parse("2006-01-02", startdate)
	if err != nil {
		log.Error("Error StartDate request: ", err)
		return searchDate, errors.New("StartDate should be format as YYY-MM-DD")
	}
	//Convert End Date
	end, err := time.Parse("2006-01-02", enddate)
	if err != nil {
		log.Error("Error EndDate request: ", err)
		return searchDate, errors.New("EndDate should be format as YYY-MM-DD")
	}

	if end.Before(start) {
		log.Error("Error Time request: ", err)
		return searchDate, errors.New("EndDate cannot before StartDate")
	}
	//end = end.Add(time.Hour * 24)
	//enddate = end.Format("2006-01-02")
	log.Info("startDate: ", startdate)
	log.Info("endDate: ", enddate)
	searchDate.StartDate = startdate
	searchDate.EndDate = enddate
	return searchDate, err
}
func ServeFile (user []User, w http.ResponseWriter, r *http.Request){
	r.ParseMultipartForm(32 << 20)
	CreateDirIfNotExist(ResultFolder)
	clientFileName := "/result_" + time.Now().Format("20060102150405") + ".csv"
	log.Info("OpenFile")
	clientsFile, err := os.OpenFile(ResultFolder+clientFileName, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		log.Errorf("Error happen OpenFile : %s", err)
		ResponseError(w, err , http.StatusInternalServerError)
		return
	}
	defer clientsFile.Close()
	log.Info("MarshalFile")
	err = gocsv.MarshalFile(&user, clientsFile) // Use this to save the CSV back to the file
	if err != nil {
		log.Errorf("Error happen in MarshalFile : %s", err)
	}
	// w.Header().Set("Content-Type", "application/json")
	// w.Write([]byte(fileBase))
	w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
	w.Header().Set("Content-Disposition", "attachment; filename='attachment.zip'")

	io.Copy(w, clientsFile)
	http.ServeFile(w, r, clientsFile.Name())
}
