package gscrypto

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"syscall"
	"time"

	zxcvbn "github.com/nbutton23/zxcvbn-go"

	"cloud.google.com/go/datastore"
	"cloud.google.com/go/storage"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/ssh/terminal"

	"golang.org/x/net/context"
)

var (
	FileStructDB FileDatabase
	PasswordDB   PasswordDatabase
	UserCreds    UserCredentialsDatabase

	PasswordConf PasswordConfig

	StorageBucket     *storage.BucketHandle
	StorageBucketName string

	SecretKey string
)

type PasswordConfig struct {
	PgpPassword []byte
}

const ProjectID = "gscrypto-154621"

func init() {
	var err error

	SecretKey = os.Getenv("SECRET_KEY")
	if SecretKey == "" {
		panic("Set SECRET_KEY environment variable.")
	}

	FileStructDB, err = configureDatastoreDB(ProjectID)

	if err != nil {
		log.Fatal(err)
	}

	PasswordDB, err = configureDatastoreDB(ProjectID)

	if err != nil {
		log.Fatal(err)
	}

	UserCreds, err = configureDatastoreDB(ProjectID)

	if err != nil {
		log.Fatal(err)
	}

	StorageBucketName = "gscrypto-bucket"
	StorageBucket, err = configureStorage(StorageBucketName)

	if err != nil {
		log.Fatal(err)
	}

	//PasswordConf = PasswordConfig{[]byte("abc")}

	var password []byte
	if ph, err := PasswordDB.GetCryptoPasswordHash(); err != nil && err.Error() == ErrorNoDatabaseEntryFound {
		fmt.Println("No password credentials are stored for file encryption/decryption, set them below.")
		fmt.Print("Password: ")
		password1, _ := terminal.ReadPassword(int(syscall.Stdin))
		fmt.Print("\nPassword repeat: ")
		fmt.Print()
		password2, _ := terminal.ReadPassword(int(syscall.Stdin))

		if !bytes.Equal(password1, password2) {
			panic("Passwords don't match")
		}

		passwordInfo := zxcvbn.PasswordStrength(string(password1), []string{})
		if passwordInfo.Score < 3 {
			panic("The password you picked isn't secure enough.")
		}

		password = password1
		newPasswordHash, err := generatePasswordHash(password1)

		if err != nil {
			panic(err)
		}

		salt := make([]byte, 32)
		rand.Read(salt)

		passwordHash := &PasswordHash{
			CreatedDate: time.Now(),
			Hash:        newPasswordHash,
			Iterations:  500000,
			Salt:        salt,
		}
		PasswordDB.SetCryptoPasswordHash(passwordHash)
	} else {
		fmt.Print("Password: ")
		password, _ = terminal.ReadPassword(int(syscall.Stdin))

		if err := bcrypt.CompareHashAndPassword(ph.Hash, password); err != nil {
			panic(err)
		}
	}

	configureCrypto(password)
}

func generatePasswordHash(password []byte) ([]byte, error) {
	if p, err := bcrypt.GenerateFromPassword(password, 1); err != nil {
		return nil, err
	} else {
		return p, err
	}
}

func configureCrypto(password []byte) error {
	passinfo, err := PasswordDB.GetCryptoPasswordHash()
	if err != nil {
		return err
	}
	key := pbkdf2.Key(password, passinfo.Salt, passinfo.Iterations, 256, sha256.New)
	PasswordConf = PasswordConfig{PgpPassword: key}
	return nil
}

func configureDatastoreDB(projectID string) (*datastoreDB, error) {
	ctx := context.Background()

	client, err := datastore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return newDatastoreDB(client)
}

func configureStorage(bucketID string) (*storage.BucketHandle, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return client.Bucket(bucketID), nil
}
