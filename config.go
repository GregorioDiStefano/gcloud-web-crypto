package gscrypto

import (
	"log"

	"cloud.google.com/go/datastore"
	"cloud.google.com/go/storage"

	"golang.org/x/net/context"
)

var (
	FileStructDB FileDatabase
	PasswordDB   PasswordDatabase
	UserCreds    UserCredentialsDatabase

	Password          []byte
	PlainTextPassword []byte

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
}

/*
func createUserAccount(passwordHash string) {
	passwordInfo := zxcvbn.PasswordStrength(password, []string{})
	if passwordInfo.Score < 3 {
		panic("The password you picked isn't secure enough.")
	}

	newPasswordHash, err := GeneratePasswordHash([]byte(password))

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
}
*/

func getUserSetup() bool {
	if _, err := PasswordDB.GetCryptoPasswordHash(); err != nil && err.Error() == ErrorNoDatabaseEntryFound {
		return false
	}
	return true
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
