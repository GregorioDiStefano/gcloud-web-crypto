package gscrypto

import (
	"log"
	"os"

	"cloud.google.com/go/datastore"
	"cloud.google.com/go/storage"

	"golang.org/x/net/context"
)

var (
	FileStructDB FileDatabase
	UserDB       UserDatabase

	Password          []byte
	PlainTextPassword []byte

	StorageBucket     *storage.BucketHandle
	StorageBucketName string

	SecretKey string
)

func init() {
	var err error
	ProjectID := os.Getenv("GOOGLE_CLOUD_PROJECT_ID")
	StorageBucketName := os.Getenv("GOOGLE_CLOUD_STORAGE_BUCKET")
	SecretKey = os.Getenv("JWT_KEY")

	if ProjectID == "" || StorageBucketName == "" || SecretKey == "" {
		panic("did you set GOOGLE_CLOUD_PROJECT_ID, GOOGLE_CLOUD_STORAGE_BUCKET and JWT_KEY?")
	}

	FileStructDB, err = configureDatastoreDB(ProjectID)

	if err != nil {
		log.Fatal(err)
	}

	UserDB, err = configureDatastoreDB(ProjectID)

	if err != nil {
		log.Fatal(err)
	}

	StorageBucket, err = configureStorage(StorageBucketName)

	if err != nil {
		log.Fatal(err)
	}
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
