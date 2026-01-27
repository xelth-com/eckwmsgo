package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	db, err := sql.Open("postgres", "host=localhost port=5432 user=openpg password=openpgpwd dbname=eckwms sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT username, email, user_type FROM user_auths LIMIT 5")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("Users in database:")
	fmt.Println("Username\tEmail\tType")
	fmt.Println("----------------------------------------")
	for rows.Next() {
		var username, email, userType string
		err := rows.Scan(&username, &email, &userType)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s\t%s\t%s\n", username, email, userType)
	}
}
