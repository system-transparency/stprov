package main

import (
	"log"

	"system-transparency.org/stprov/internal/api"
	"system-transparency.org/stprov/internal/sb"
)

var (
	optPKFile  = "pk.esl"
	optKEKFile = "kek.esl"
	optDBFile  = "db.esl"
	optDBXFile = "dbx.esl"
)

func main() {
	pk, err := sb.ReadOptionalESLFile(optPKFile)
	if err != nil {
		log.Fatalf("invalid pk: %v\n", err)
	}
	kek, err := sb.ReadOptionalESLFile(optKEKFile)
	if err != nil {
		log.Fatalf("invalid kek: %v\n", err)
	}
	db, err := sb.ReadOptionalESLFile(optDBFile)
	if err != nil {
		log.Fatalf("invalid db: %v\n", err)
	}
	dbx, err := sb.ReadOptionalESLFile(optDBXFile)
	if err != nil {
		log.Fatalf("invalid dbx: %v\n", err)
	}

	req, _ := api.NewAddSecureBootRequest(pk, kek, db, dbx)
	if err := sb.ProvisionKeys(req.PK, req.KEK, req.DB, req.DBX); err != nil {
		log.Fatalf("provision keys: %v\n", err)
	}

	log.Printf("OK\n")
}
