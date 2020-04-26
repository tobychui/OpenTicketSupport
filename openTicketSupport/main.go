package main

import (
  "log"
  "net/http"
  "os"
  "os/signal"
  "time"
  "syscall"
  "strconv"
  "io/ioutil"
  "reflect"
  "flag"
  "github.com/boltdb/bolt"
  "encoding/json"
  "encoding/csv"
  "github.com/grokify/html-strip-tags-go"
  "github.com/nu7hatch/gouuid"
)
var adminToken = flag.String("token", "admin", "Admin password")
var sysdb *bolt.DB

type ticket struct{
	Email string;
	Fullname string;
	Organization string;
	Title string;
	Content string;
	Uuid string;
	SubmissionTime time.Time;
}

func SetupCloseHandler() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		//Shutdown Database
		log.Println("\r- Shutting down database")
		system_db_closeDatabase(sysdb)

		//Do other things

		os.Exit(0)
	}()
}

func main() {
	//Get admin token from flags
	flag.Parse()

	//Setup close handler on db
	SetupCloseHandler()

	//Initiate the database
	sysdb = system_db_service_init("ticket.db")

	//Setup ticket storage db if not exists
	system_db_newTable(sysdb, "tickets")

	//Create static file server
	fs := http.FileServer(http.Dir("./web"))
	http.Handle("/", fs)

	//Handle admin login functions
    http.HandleFunc("/login", login)
	http.HandleFunc("/logout", logout)
	
	http.HandleFunc("/submitTicket", handleTicketSubmit)
	http.HandleFunc("/adminLogin", handleAdminLogin)
	http.HandleFunc("/adminLogout", handleLogout)
	http.HandleFunc("/list", handleListToken)
	http.HandleFunc("/chklogin", handleLoginCheck)
	http.HandleFunc("/exportcsv", exportCSV)

	log.Println("Listening on :80...")
	err := http.ListenAndServe(":80", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func handleTicketSubmit(w http.ResponseWriter, r *http.Request){
	//Reciving all information from form submit
	email, _ := mv(r, "email", false);
	fullname, _ := mv(r, "fullname", false);
	organization, _ := mv(r, "organization", false);
	title, _ := mv(r, "title", false);
	content, _ := mv(r, "content", false);
	log.Println("Reciving ticket submission: ", email, fullname, organization, title, content)

	//Create a struct to store the tickets
	thisTicket := new(ticket)
	thisTicket.Email = strip.StripTags(email)
	thisTicket.Fullname = strip.StripTags(fullname)
	thisTicket.Organization = strip.StripTags(organization)
	thisTicket.Title = strip.StripTags(title)
	thisTicket.Content = strip.StripTags(content)
	thisTicket.SubmissionTime = time.Now()
	//Generate UUID for this ticket
	ticketKey := strconv.FormatInt(time.Now().Unix(), 10) + getNewUUID();
	thisTicket.Uuid = ticketKey

	//Write the struct into the database
	system_db_write(sysdb, "tickets", ticketKey, thisTicket)

	//Redirect user back to index
	http.Redirect(w, r, "submitComplete.html#" + ticketKey,307)
}

func handleAdminLogin(w http.ResponseWriter, r *http.Request){
	token, _ := mv(r,"token", true)
	//Check if the token is valid
	if (token == *adminToken){
		//Assign login session
		login(w,r);
		sendOK(w);
	}else{
		sendErrorResponse(w, "Invalid token")
	}
}

func handleLogout(w http.ResponseWriter, r *http.Request){
	//Remove login session
	logout(w,r);
	
	//Redirect user to index
	http.Redirect(w, r, "admin/index.html#" + "logout",307)
}

//List all the tickets in the database
func handleListToken(w http.ResponseWriter, r *http.Request){
	if (!checkLogin(w,r)){
		sendErrorResponse(w, "User not logged in");
		return;
	}
	entries := system_db_listTable(sysdb, "tickets")
	//Reverse the entry so the latest one on top
	reverseAny(entries)

	results := []ticket{}

	//Get from paramter see how many tickets to list
	count, _ := mv(r, "c", false)
	if (count == "" || count == "0"){
		//Not defined. List all
		for _, thisTicket := range entries{
			ticketObjectJson := thisTicket[1]
			thisTicketObject := new(ticket)
			err := json.Unmarshal(ticketObjectJson, &thisTicketObject)
			if (err != nil){
				panic(err)
			}
			results = append(results, *thisTicketObject)
		}
	}else{
		//Defined. Only list the top n tickets
		maxNumber, err := StringToInt(count);
		if (err != nil){
			sendErrorResponse(w, "Invalid c number")
			return;
		}
		//Handle case where the number of tickets is less than the request list
		if (len(entries) < maxNumber){
			maxNumber = len(entries);
		}

		for i:=0; i < maxNumber; i++{
			ticketObjectJson := entries[i][1]
			thisTicketObject := new(ticket)
			err := json.Unmarshal(ticketObjectJson, &thisTicketObject)
			if (err != nil){
				panic(err)
			}
			results = append(results, *thisTicketObject)
		}
	}

	//Output the tickets data as json reponse
	jsonResults, _ := json.Marshal(results);
	sendJSONResponse(w, string(jsonResults))
}

func getNewUUID() string{
	u, _ := uuid.NewV4()
	return u.String()
}

//Function to reverse a slice, from https://stackoverflow.com/questions/28058278/how-do-i-reverse-a-slice-in-go
func reverseAny(s interface{}) {
    n := reflect.ValueOf(s).Len()
    swap := reflect.Swapper(s)
    for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
        swap(i, j)
    }
}

func handleLoginCheck(w http.ResponseWriter, r *http.Request){
	if (checkLogin(w,r)){
		sendJSONResponse(w, "true");
	}else{
		sendJSONResponse(w, "false");
	}
}

func exportCSV(w http.ResponseWriter, r *http.Request){
	if (!checkLogin(w,r)){
		sendErrorResponse(w, "User not logged in");
		return;
	}
	file, err := os.Create("export.csv")
    if err != nil {
		sendErrorResponse(w, "Another export is in progress")
		return;
    }
    defer file.Close()

    writer := csv.NewWriter(file)
	
	//Get all tickets from db
	entries := system_db_listTable(sysdb, "tickets")
	results := [][]string{[]string{
		"uuid","title","fullname","email","organization","content","submission time",
	}}
	for _, thisTicket := range entries{
		ticketObjectJson := thisTicket[1]
		thisTicketObject := new(ticket)
		err := json.Unmarshal(ticketObjectJson, &thisTicketObject)
		if (err != nil){
			panic(err)
		}
		thisCSVLine := []string{
			thisTicketObject.Uuid,
			thisTicketObject.Title,
			thisTicketObject.Fullname,
			thisTicketObject.Email,
			thisTicketObject.Organization,
			thisTicketObject.Content,
			thisTicketObject.SubmissionTime.Format("2006-01-02 15:04:05"),
		}
		results = append(results, thisCSVLine)
	}

	//Finish writing
	writer.WriteAll(results)
	writer.Flush()

	//Read and serve the file
	w.Header().Set("Content-Disposition", "attachment; filename=tickets.csv")
	w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
	csvContent, _ := ioutil.ReadFile("export.csv")
    w.Write(csvContent);
}