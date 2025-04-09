package main

import (
	"log"

	"system-transparency.org/stprov/internal/sb"
)

var (
	optPKFile  = "pk.esl"
	optKEKFile = "kek.esl"
	optDBFile  = "db.esl"
	optDBXFile = "dbx.esl"
)

func main() {
	//pk, err := sb.ReadOptionalESLFile(optPKFile)
	//if err != nil {
	//	log.Fatalf("invalid pk: %v\n", err)
	//}
	//kek, err := sb.ReadOptionalESLFile(optKEKFile)
	//if err != nil {
	//	log.Fatalf("invalid kek: %v\n", err)
	//}
	//db, err := sb.ReadOptionalESLFile(optDBFile)
	//if err != nil {
	//	log.Fatalf("invalid db: %v\n", err)
	//}
	//dbx, err := sb.ReadOptionalESLFile(optDBXFile)
	//if err != nil {
	//	log.Fatalf("invalid dbx: %v\n", err)
	//}
	//req, _ := api.NewAddSecureBootRequest(pk, kek, db, dbx)

	log.Printf("*** Before provisioning\n")
	status()

	if err := sb.ProvisionKeys(req.PK, req.KEK, req.DB, req.DBX); err != nil {
		log.Fatalf("provision keys: %v\n", err)
	}
	if err := sb.SetSecureBoot(); err != nil {
		log.Fatalf("set SecureBoot: %v\n", err)
	}
	if err := sb.SetDeployedMode(); err != nil {
		log.Fatalf("set DeployedMode: %v\n", err)
	}

	log.Printf("*** After provisioning provisioning\n")
	status()

	log.Printf("OK\n")
}

func status() {
	isSetupMode, err := sb.IsSetupMode()
	if err != nil {
		log.Fatalf("is setup mode: %v\n", err)
	}
	isAuditMode, err := sb.IsAuditMode()
	if err != nil {
		log.Fatalf("is audit mode: %v\n", err)
	}
	isDeployedMode, err := sb.IsDeployedMode()
	if err != nil {
		log.Fatalf("is deployed mode: %v\n", err)
	}
	isSecureBoot, err := sb.IsSetupMode()
	if err != nil {
		log.Fatalf("is setup mode: %v\n", err)
	}

	log.Printf("SetupMode:    %v", isSetupMode)
	log.Printf("AuditMode:    %v", isAuditMode)
	log.Printf("DeployedMode: %v", isDeployedMode)
	log.Printf("SecureBoot:   %v", isSecureBoot)
}
