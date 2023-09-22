package pkg

import (
	enteCrypto "cli-go/internal/crypto"
	"cli-go/pkg/model"
	"cli-go/utils"
	"context"
	"encoding/base64"
	"fmt"
	bolt "go.etcd.io/bbolt"
	"log"
)

var accountMasterKey = map[string][]byte{}

func (c *ClICtrl) SyncAccount(account model.Account) error {
	log.SetPrefix(fmt.Sprintf("[%s] ", account.Email))

	err := createDataBuckets(c.DB, account)
	if err != nil {
		return err
	}
	token := account.Token.MustDecrypt(c.CliKey)
	accountMasterKey[account.AccountKey()] = utils.DecodeBase64(account.MasterKey.MustDecrypt(c.CliKey))
	urlEncodedToken := base64.URLEncoding.EncodeToString(utils.DecodeBase64(token))
	c.Client.AddToken(account.AccountKey(), urlEncodedToken)
	ctx := c.GetRequestContext(context.Background(), account)
	return c.syncRemoteCollections(ctx, account)
}

func (c *ClICtrl) GetRequestContext(ctx context.Context, account model.Account) context.Context {
	ctx = context.WithValue(ctx, "app", string(account.App))
	ctx = context.WithValue(ctx, "account_id", account.AccountKey())
	return ctx
}

var dataCategories = []string{"remote-collections", "local-collections", "remote-files", "local-files", "remote-collection-removed", "remote-files-removed"}

func createDataBuckets(db *bolt.DB, account model.Account) error {
	return db.Update(func(tx *bolt.Tx) error {
		dataBucket, err := tx.CreateBucketIfNotExists([]byte(account.DataBucket()))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		for _, category := range dataCategories {
			_, err := dataBucket.CreateBucketIfNotExists([]byte(fmt.Sprintf(category)))
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (c *ClICtrl) syncRemoteCollections(ctx context.Context, info model.Account) error {
	collections, err := c.Client.GetCollections(ctx, 0)
	if err != nil {
		log.Printf("failed to get collections: %s\n", err)
		return err
	}
	masterKey := accountMasterKey[info.AccountKey()]
	for _, collection := range collections {
		if collection.Owner.ID != info.UserID {
			fmt.Printf("Skipping collection %d\n", collection.ID)
			continue
		}
		collectionKey := collection.GetCollectionKey(masterKey)
		name, nameErr := enteCrypto.SecretBoxOpenBase64(collection.EncryptedName, collection.NameDecryptionNonce, collectionKey)
		if nameErr != nil {
			log.Fatalf("failed to decrypt collection name: %v", nameErr)
		}
		fmt.Printf("Collection Name %s\n", string(name))
	}
	return nil
}
