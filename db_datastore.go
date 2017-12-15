package gscrypto

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"cloud.google.com/go/datastore"
	"golang.org/x/net/context"
)

type datastoreDB struct {
	client *datastore.Client
}

const (
	ErrorNoDatabaseEntryFound = "no entry/file found"
	ErrorNotRequestingUsers   = "this object does not belong to the requesting user"
)

var _ FileDatabase = &datastoreDB{}

func newDatastoreDB(client *datastore.Client) (*datastoreDB, error) {
	ctx := context.Background()
	// Verify that we can communicate and authenticate with the datastore service.
	t, err := client.NewTransaction(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not connect: %v", err)
	}
	if err := t.Rollback(); err != nil {
		return nil, fmt.Errorf("could not connect: %v", err)
	}
	return &datastoreDB{
		client: client,
	}, nil
}

// Close closes the database.
func (db *datastoreDB) Close() {
	// No op.
}

func (db *datastoreDB) GetFile(user string, id int64) (*File, error) {
	ctx := context.Background()
	var encfile File

	key := datastore.IDKey("FileStruct", id, nil)
	err := db.client.Get(ctx, key, &encfile)

	if err != nil {
		return nil, fmt.Errorf("could not list files: %v", err)
	}

	if encfile.Username != user {
		return nil, fmt.Errorf(ErrorNotRequestingUsers)
	}

	return &encfile, nil
}

func (db *datastoreDB) AddFile(f *File) (id int64, err error) {
	ctx := context.Background()
	k := datastore.IncompleteKey("FileStruct", nil)
	k, err = db.client.Put(ctx, k, f)

	if err != nil {
		return 0, fmt.Errorf("could not put file: %v", err)
	}
	return k.ID, nil
}

func (db *datastoreDB) UpdateFile(f *File, id int64) (err error) {
	ctx := context.Background()

	k := datastore.IDKey("FileStruct", id, nil)
	_, err = db.client.Put(ctx, k, f)

	if err != nil {
		return fmt.Errorf("could not put file: %v", err)
	}
	return nil
}

func (db *datastoreDB) ListFiles(user, path string) ([]File, error) {
	ctx := context.Background()

	encfile := make([]File, 0)
	q := datastore.NewQuery("FileStruct")

	if path != "" {
		q = q.Filter("Folder =", path)
	}

	q = q.Filter("Username = ", user)

	if keys, err := db.client.GetAll(ctx, q, &encfile); err != nil {
		return nil, fmt.Errorf("could not list files: %v", err)
	} else {
		for index, key := range keys {
			encfile[index].ID = key.ID
		}
	}

	return encfile, nil
}

func (db *datastoreDB) ListTags() ([]string, error) {
	ctx := context.Background()

	tags := []string{}
	encfile := make([]File, 0)
	q := datastore.NewQuery("FileStruct").Project("Tags").Distinct()

	_, err := db.client.GetAll(ctx, q, &encfile)

	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	for _, f := range encfile {
		for _, tag := range f.Tags {
			tags = append(tags, tag)
		}
	}

	return tags, nil
}

func (db *datastoreDB) ListFilesWithTags(tags []string) ([]File, error) {
	ctx := context.Background()

	encfile := make([]File, 0)
	q := datastore.NewQuery("FileStruct")

	for _, tag := range tags {
		q = q.Filter("Tags =", strings.ToLower(tag))
	}

	_, err := db.client.GetAll(ctx, q, &encfile)

	if err != nil {
		return nil, fmt.Errorf("could not list files: %v", err)
	}

	return encfile, nil
}

func (db *datastoreDB) ListFolders(user, path string) ([]FolderTree, int64, error) {
	ctx := context.Background()

	encfile := make([]FolderTree, 0)
	q := datastore.NewQuery("FolderStruct")
	q = q.Filter("Username = ", user)
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
	nq = nq.Filter("Username = ", user)
	encfile = make([]FolderTree, 0)

	if keys, err := db.client.GetAll(ctx, nq, &encfile); err != nil {
		return nil, 0, fmt.Errorf("could not list files: %v", err)
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

func (db *datastoreDB) DeleteFile(user string, id int64) error {
	var f File
	ctx := context.Background()
	key := datastore.IDKey("FileStruct", id, nil)

	if err := db.client.Get(ctx, key, &f); err != nil {
		return err
	} else if f.Username != user {
		return errors.New(ErrorNotRequestingUsers)
	}

	err := db.client.Delete(ctx, &datastore.Key{ID: id, Kind: "FileStruct"})
	if err != nil {
		return err
	}

	return nil
}

func (db *datastoreDB) DeleteFolder(user string, id int64) error {
	var f FolderTree
	ctx := context.Background()
	key := datastore.IDKey("FolderStruct", id, nil)

	if err := db.client.Get(ctx, key, &f); err == nil {
		if f.Username != user {
			return errors.New(ErrorNoDatabaseEntryFound)
		}
	} else {
		return err
	}

	err := db.client.Delete(ctx, &datastore.Key{ID: id, Kind: "FolderStruct"})
	if err != nil {
		fmt.Println("error deletting folder: ", err.Error())
		return err
	}
	return nil
}

func (db *datastoreDB) FilenameHMACExists(user, searchHMAC string) bool {
	ctx := context.Background()
	fmt.Println("looking for:", searchHMAC)
	q := datastore.NewQuery("FileStruct").Filter("FilenameHMAC = ", searchHMAC)
	q.Filter("Username = ", user)

	encfile := make([]File, 0)
	_, err := db.client.GetAll(ctx, q, &encfile)

	fmt.Println("encfile: ", encfile, err == nil && len(encfile) > 0)
	return err == nil && len(encfile) > 0
}

func (db *datastoreDB) SetUserEntry(userEntry *UserEntry) error {
	ctx := context.Background()
	k := datastore.IncompleteKey("UserEntry", nil)
	_, err := db.client.Put(ctx, k, userEntry)

	if err != nil {
		return err
	}

	return nil
}

func (db *datastoreDB) UpdateUser(id int64, userEntry *UserEntry) error {
	ctx := context.Background()

	k := datastore.IDKey("UserEntry", id, nil)
	_, err := db.client.Put(ctx, k, userEntry)

	if err != nil {
		return fmt.Errorf("could not put file: %v", err)
	}
	return nil
}

func (db *datastoreDB) GetUserEntry(user string) (*UserEntry, int64, error) {
	ctx := context.Background()
	q := datastore.NewQuery("UserEntry").Filter("Username = ", user)
	u := make([]*UserEntry, 0)

	keys, err := db.client.GetAll(ctx, q, &u)

	if err != nil {
		return nil, 0, err
	}

	if len(u) == 0 {
		return nil, 0, errors.New(ErrorNoDatabaseEntryFound)
	}

	return u[0], keys[0].ID, nil
}

func (db *datastoreDB) GetUsers() ([]*UserEntry, error) {
	ctx := context.Background()
	q := datastore.NewQuery("UserEntry")
	users := make([]*UserEntry, 0)

	_, err := db.client.GetAll(ctx, q, &users)

	return users, err
}

func (db *datastoreDB) GetAllFiles(user string) ([]*File, error) {
	ctx := context.Background()

	encfile := make([]*File, 0)
	q := datastore.NewQuery("FileStruct")
	q = q.Filter("Username =", user)

	_, err := db.client.GetAll(ctx, q, &encfile)

	return encfile, err
}

func (db *datastoreDB) GetAllFilesMatchingFolder(user, folder string) ([]*File, []*datastore.Key, error) {
	ctx := context.Background()

	files := make([]*File, 0)
	q := datastore.NewQuery("FileStruct")
	q = q.Filter("Username =", user).Filter("Folder >=", folder)
	keys, err := db.client.GetAll(ctx, q, &files)

	return files, keys, err
}

func (db *datastoreDB) ListAllFolders(user, search string, limit int) ([]string, error) {
	ctx := context.Background()

	matchingFolders := make([]string, 0)
	file := make([]*File, 0)
	q := datastore.NewQuery("FileStruct")
	q = q.Filter("Username =", user).Project("Folder").Distinct().Limit(limit)
	_, err := db.client.GetAll(ctx, q, &file)

	for _, folder := range file {
		if strings.HasPrefix(folder.Folder, search) {
			matchingFolders = append(matchingFolders, folder.Folder)
		}
	}

	return matchingFolders, err
}

func (db *datastoreDB) List(user string) {
	ctx := context.Background()

	//	matchingFolders := make([]string, 0)
	file := make([]*File, 0)

	q := datastore.NewQuery("FileStruct")
	q = q.Filter("Folder >=", "")
	db.client.GetAll(ctx, q, &file)
	fmt.Println("---------------------------")
	for _, f := range file {
		fmt.Println(f)
	}
	fmt.Println("---------------------------")
}

func (db *datastoreDB) RenameFolder(user, oldFolder, newFolder string) (string, error) {
	ctx := context.Background()

	finalFolderPath := newFolder
	file := make([]*File, 0)

	q := datastore.NewQuery("FileStruct")
	q = q.Filter("Username =", user).Filter("Folder >=", oldFolder)

	keys, err := db.client.GetAll(ctx, q, &file)

	if err != nil {
		return "", err
	}

	for idx, f := range file {

		orgKey := keys[idx].ID

		k := datastore.IncompleteKey("FileStruct", nil)

		// replace full directory
		if f.Folder == oldFolder {
			f.Folder = newFolder
		} else {
			/*
				replace sub folder:

					oldFolder=/foo/bar/
					newFolder=/bar/

					but we get files, with full paths:

					/foo/bar/a/1
					/foo/bar/b/2
			*/

			oldFolderPaths := strings.Split(strings.Trim(oldFolder, "/"), "/")
			newFolderPaths := strings.Split(strings.Trim(newFolder, "/"), "/")
			filePaths := strings.Split(strings.Trim(f.Folder, "/"), "/")

			if len(newFolderPaths) == 0 {
				return "", fmt.Errorf("invalid destination folder")
			}

			match := true

			if len(filePaths) < len(oldFolderPaths) {
				fmt.Println(filePaths, "is less than", oldFolderPaths)
				match = false
			} else {
				for idx, dir := range oldFolderPaths {
					if filePaths[idx] != dir {
						match = false
					}
				}
			}

			if match {
				finalFolderPath = filepath.Clean(newFolder + "/" + strings.TrimPrefix(f.Folder, oldFolder))
				finalFolderPath += "/"
				fmt.Println("final folder: ", finalFolderPath)
				f.Folder = finalFolderPath
			}

		}

		if key, err := db.client.Put(ctx, k, f); err == nil {
			fmt.Println(key)
		}

		db.client.Delete(ctx, &datastore.Key{ID: orgKey, Kind: "FileStruct"})
	}
	return finalFolderPath, nil
}
