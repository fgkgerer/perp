package main

import (
	"fmt"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func try(dsn, label string) {
	_, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Println(label, "FAIL:", err)
		return
	}
	fmt.Println(label, "OK")
}

func main() {
	pwd := os.Getenv("PGPWD")
	if pwd == "" {
		pwd = "fRiWDteuoyqZPWu6"
	}
	try("postgresql://postgres:"+pwd+"@db.tnmhbgomjevaizaxmzuy.supabase.co:5432/postgres?sslmode=require", "old-direct")
	try("postgresql://postgres:"+pwd+"@db.niwscthznuliojtmcywu.supabase.co:5432/postgres?sslmode=require", "new-direct")
	try("postgresql://postgres.niwscthznuliojtmcywu:"+pwd+"@aws-0-ap-southeast-1.pooler.supabase.com:6543/postgres?sslmode=require", "new-pooler-ap")
}
