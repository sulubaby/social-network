package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"social/cmd/server/routes"
	"social/cmd/utils"
	"social/pkg/db/sqlite"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	path := os.Getenv("ORBIT_DB_PATH")
	if path == "" {
		path = "db/social_network.db"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		log.Fatal(err)
	}

	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	db, err := sqlite.ConnectToDB("sqlite3", path+separator+"_foreign_keys=on")
	if err != nil {
		log.Println(err)
		return
	}

	err = sqlite.RunMigrations(path)
	if err != nil {
		log.Println(err)
		return
	}

	if err := utils.InitUploadFolders(); err != nil {
		log.Println("error", err)
	}

	if err := utils.CopyDefaultAvatar(); err != nil {
		log.Println(err)
	}

	address := os.Getenv("ORBIT_ADDR")
	if address == "" {
		address = ":4000"
	}
	server := http.Server{
		Handler: routes.StartServer(db),
		Addr:    address,
	}

	err = server.ListenAndServe()
	if err != nil {
		log.Println("ERROR: STARTING SERVER", err)
		return
	}
	log.Println("SERVER STARTED ON POST: " + server.Addr)
}
