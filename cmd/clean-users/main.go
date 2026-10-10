package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "modernc.org/sqlite"
)

func main() {
	paths := []string{"data/mcrflow.db", "tmp/qa-test-data/mcrflow.db", "tmp/e2e-data/mcrflow.db"}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			db, err := sql.Open("sqlite", p)
			if err != nil {
				fmt.Printf("Error opening %s: %v\n", p, err)
				continue
			}
			res, err := db.Exec("DELETE FROM users")
			if err != nil {
				fmt.Printf("Error deleting users from %s: %v\n", p, err)
			} else {
				n, _ := res.RowsAffected()
				fmt.Printf("Successfully purged %d users from %s. User count is now 0.\n", n, p)
			}
			db.Close()
		}
	}
}
