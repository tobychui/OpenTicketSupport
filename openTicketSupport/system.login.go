package main

import (
    "net/http"
    "github.com/gorilla/sessions"
)

var (
    // key must be 16, 24 or 32 bytes long (AES-128, AES-192 or AES-256)
    //Please change this before you deploy
    key = []byte("D1170AD54C7753A8422816C9F030DA43AFB8366F1D1745ECCCFA8CE7BE17E5A4")
    store = sessions.NewCookieStore(key)
)

func checkLogin(w http.ResponseWriter, r *http.Request) bool{
    session, _ := store.Get(r, "ots-auth")
    if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
        return false
    }
    return true
}

func login(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "ots-auth")
    session.Values["authenticated"] = true
    session.Save(r, w)
}

func logout(w http.ResponseWriter, r *http.Request) {
    session, _ := store.Get(r, "ots-auth")
    session.Values["authenticated"] = false
    session.Save(r, w)
}
