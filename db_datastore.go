package gscrypto

import (
	"errors"
	"fmt"

	"cloud.google.com/go/datastore"
	"golang.org/x/net/context"
)

// datastoreDB persists books to Cloud Datastore.
// https://cloud.google.com/datastore/docs/concepts/overview
type datastoreDB struct {
	client *datastore.Client
}

const (
	ErrorNoDatabaseEntryFound = "no entry found"
)

// Ensure datastoreDB conforms to the BookDatabase interface.
var _ FileDatabase = &datastoreDB{}

// newDatastoreDB creates a new BookDatabase backed by Cloud Datastore.
// See the datastore and google packages for details on creating a suitable Client:
// https://godoc.org/cloud.google.com/go/datastore
func newDatastoreDB(client *datastore.Client) (*datastoreDB, error) {
	ctx := context.Background()
	// Verify that we can communicate and authenticate with the datastore service.
	t, err := client.NewTransaction(ctx)
	if err != nil {
		return nil, fmt.Errorf("datastoredb: could not connect: %v", err)
	}
	if err := t.Rollback(); err != nil {
		return nil, fmt.Errorf("datastoredb: could not connect: %v", err)
	}
	return &datastoreDB{
		client: client,
	}, nil
}

// Close closes the database.
func (db *datastoreDB) Close() {
	// No op.
}

func (db *datastoreDB) datastoreKey(id int64) *datastore.Key {
	return datastore.IDKey("Book", id, nil)
}

// GetBook retrieves a book by its ID.
func (db *datastoreDB) GetFile(id string) (*File, error) {
	ctx := context.Background()
	encfile := make([]*File, 0)

	q := datastore.NewQuery("FileStruct").Filter("ID =", id)

	keys, err := db.client.GetAll(ctx, q, &encfile)

	if err != nil {
		return nil, fmt.Errorf("datastoredb: could not list books: %v", err)
	}

	fmt.Println("keys: ", keys, err)
	for i, k := range keys {
		fmt.Println(i, k, encfile)
	}

	//	if err := db.client.Get(ctx, k, encfile); err != nil {
	//		return nil, fmt.Errorf("datastoredb: could not get Book: %v", err)
	//	}
	fmt.Println(encfile)
	return encfile[0], nil
}

func (db *datastoreDB) AddFile(b *File) (id int64, err error) {
	ctx := context.Background()
	k := datastore.IncompleteKey("FileStruct", nil)
	//	k := datastore.IncompleteKey("FileStruct", nil)
	k, err = db.client.Put(ctx, k, b)
	if err != nil {
		return 0, fmt.Errorf("datastoredb: could not put Book: %v", err)
	}
	return k.ID, nil
}

func (db *datastoreDB) ListFiles(path string) ([]File, error) {
	ctx := context.Background()

	encfile := make([]File, 0)
	q := datastore.NewQuery("FileStruct")

	if path != "" {
		q = q.Filter("Folder =", path)
	}

	_, err := db.client.GetAll(ctx, q, &encfile)

	if err != nil {
		return nil, fmt.Errorf("datastoredb: could not list books: %v", err)
	}

	return encfile, nil
}

func (db *datastoreDB) ListFolders(path string) ([]FolderTree, int64, error) {
	ctx := context.Background()

	encfile := make([]FolderTree, 0)
	q := datastore.NewQuery("FolderStruct")
	var parentFolderKey int64

	if path == "/" {
		q = q.Filter("ParentFolder = ", "")
	} else {
		for _, ft := range PathToFolderTree(path) {
			q = datastore.NewQuery("FolderStruct").
				Filter("ParentKey = ", parentFolderKey).
				Filter("ParentFolder = ", ft.ParentFolder).
				Filter("Folder = ", ft.Folder).Limit(1)
			keys, err := db.client.GetAll(ctx, q, &encfile)

			if err != nil {
				return nil, 0, err
			}

			if keys == nil || len(keys) == 0 {
				return nil, 0, nil
			}

			parentFolderKey = keys[0].ID
		}
	}

	nq := datastore.NewQuery("FolderStruct").Filter("ParentKey = ", int64(parentFolderKey))
	encfile = make([]FolderTree, 0)

	if keys, err := db.client.GetAll(ctx, nq, &encfile); err != nil {
		return nil, 0, fmt.Errorf("datastoredb: could not list books: %v", err)
	} else {
		for index, key := range keys {
			encfile[index].ID = key.ID
		}
	}

	return encfile, parentFolderKey, nil
}

func (db *datastoreDB) AddFolder(ft *FolderTree) (int64, error) {
	ctx := context.Background()
	k := datastore.IncompleteKey("FolderStruct", nil)
	key, err := db.client.Put(ctx, k, ft)
	if err != nil {
		return 0, err
	}
	return key.ID, err
}

func (db *datastoreDB) ListFilesByFolder(folder string) ([]File, error) {
	ctx := context.Background()
	encfile := make([]File, 0)
	q := datastore.NewQuery("FileStruct").Filter("Folder =", folder)
	_, err := db.client.GetAll(ctx, q, &encfile)

	if err != nil {
		return nil, fmt.Errorf("datastoredb: could not list books: %v", err)
	}

	return encfile, nil
}

func (db *datastoreDB) DeleteFile(uuid string) error {
	ctx := context.Background()
	q := datastore.NewQuery("FileStruct").Filter("ID =", uuid)
	encfile := make([]*File, 1)

	keys, err := db.client.GetAll(ctx, q, &encfile)

	if err != nil {
		return err
	}

	err = db.client.Delete(ctx, keys[0])

	if err != nil {
		return err
	}
	return nil
}

func (db *datastoreDB) DeleteFolder(key int64) error {
	ctx := context.Background()

	err := db.client.Delete(ctx, &datastore.Key{ID: key, Kind: "FolderStruct"})

	if err != nil {
		fmt.Println("error deletting folder: ", err.Error())
		return err
	}
	return nil
}

func (db *datastoreDB) SetCryptoPasswordHash(ph *PasswordHash) error {
	ctx := context.Background()
	k := datastore.IncompleteKey("PasswordConfig", nil)
	_, err := db.client.Put(ctx, k, ph)

	if err != nil {
		return err
	}

	return nil
}

func (db *datastoreDB) GetCryptoPasswordHash() (*PasswordHash, error) {
	ctx := context.Background()
	q := datastore.NewQuery("PasswordConfig")
	ph := make([]*PasswordHash, 0)

	_, err := db.client.GetAll(ctx, q, &ph)

	if err != nil {
		return nil, err
	}

	if len(ph) == 0 {
		return nil, errors.New(ErrorNoDatabaseEntryFound)
	}

	return ph[0], nil
}

func (db *datastoreDB) SetUserCreds(uc *UserCredentials) error {
	return nil
}

func (db *datastoreDB) GetUserCreds(username string) (*UserCredentials, error) {
	ctx := context.Background()
	q := datastore.NewQuery("UserConfigDatabase")
	uc := make([]*UserCredentials, 0)

	_, err := db.client.GetAll(ctx, q, &uc)

	if err != nil {
		return nil, err
	}

	if len(uc) == 0 {
		return nil, errors.New(ErrorNoDatabaseEntryFound)
	}

	return uc[0], nil
}

// hack to get around connection closing issue
func (db *datastoreDB) NOOP() {
	ctx := context.Background()
	db.client.Delete(ctx, db.datastoreKey(0))
}
