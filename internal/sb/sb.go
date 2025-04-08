package sb

import (
	"fmt"
	"os"

	"github.com/foxboron/go-uefi/efi/signature"
)

func ReadOptionalESLFile(filename string) (*signature.SignatureDatabase, error) {
	if filename == "" {
		return nil, nil
	}
	fp, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fp.Close()

	sd, err := signature.ReadSignatureDatabase(fp)
	return &sd, err
}

func ProvisionSecureBoot(pk, kek, db, dbx *signature.SignatureDatabase) error {
	if pk == nil {
		return fmt.Errorf("required argument: PK")
	}
	if kek == nil {
		return fmt.Errorf("required argument: KEK")
	}
	if db == nil {
		return fmt.Errorf("required argument: db")
	}
	if dbx == nil {
		dbx = db // TODO: set dbx to a valid zero value
	}

	//pkDummyAuth := signature.NewEFIVariableAuthentication2()
	//if err := efi.WriteEFIVariable("PK", pk); err != nil {
	//	return fmt.Errorf("enroll PK: %v", err)
	//}

	return fmt.Errorf("TODO")
}
