package pkg

import (
	"cli-go/internal/api"
	"cli-go/pkg/model"
	"cli-go/utils"
	"context"
	"encoding/json"
	"fmt"
	"log"

	bolt "go.etcd.io/bbolt"
)

const AccBucket = "accounts"

func (c *ClICtrl) AddAccount(cxt context.Context) {
	var flowErr error
	defer func() {
		if flowErr != nil {
			log.Fatal(flowErr)
		}
	}()
	app := GetAppType()
	cxt = context.WithValue(cxt, "app", string(app))
	email, flowErr := GetUserInput("Enter email address")
	if flowErr != nil {
		return
	}
	var verifyEmail bool

	srpAttr, flowErr := c.Client.GetSRPAttributes(cxt, email)
	if flowErr != nil {
		// if flowErr type is ApiError and status code is 404, then set verifyEmail to true and continue
		// else return
		if apiErr, ok := flowErr.(*api.ApiError); ok && apiErr.StatusCode == 404 {
			verifyEmail = true
		} else {
			return
		}
	}
	var authResponse *api.AuthorizationResponse
	var keyEncKey []byte
	if verifyEmail || srpAttr.IsEmailMFAEnabled {
		authResponse, flowErr = c.validateEmail(cxt, email)
	} else {
		authResponse, keyEncKey, flowErr = c.signInViaPassword(cxt, email, srpAttr)
	}
	if flowErr != nil {
		return
	}
	if authResponse.IsMFARequired() {
		authResponse, flowErr = c.validateTOTP(cxt, authResponse)
	}
	if authResponse.EncryptedToken == "" || authResponse.KeyAttributes == nil {
		panic("no encrypted token or keyAttributes")
	}
	secretInfo, decErr := c.decryptMasterKeyAndToken(cxt, authResponse, keyEncKey)
	if decErr != nil {
		flowErr = decErr
		return
	}
	err := c.storeAccount(cxt, email, authResponse.ID, app, secretInfo)
	if err != nil {
		flowErr = err
		return
	} else {
		fmt.Println("Account added successfully")
		// length of master and secret
		fmt.Printf("Master key length: %d\n", len(secretInfo.MasterKey))
		fmt.Printf("Secret key length: %d\n", len(secretInfo.SecretKey))
		fmt.Printf("Master key: %s", utils.EncodeBase64(secretInfo.MasterKey))
	}
}

func (c *ClICtrl) storeAccount(_ context.Context, email string, userID int64, app api.App, secretInfo *accSecretInfo) error {
	// get password
	secret := c.CliKey
	err := c.DB.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(AccBucket))
		if err != nil {
			return err
		}
		accInfo := model.Account{
			Email:     email,
			UserID:    userID,
			MasterKey: *model.MakeEncString(string(secretInfo.MasterKey), secret),
			Token:     *model.MakeEncString(string(secretInfo.Token), secret),
			SecretKey: *model.MakeEncString(string(secretInfo.SecretKey), secret),
			App:       app,
		}
		accInfoBytes, err := json.Marshal(accInfo)
		if err != nil {
			return err
		}
		accountKey := accInfo.AccountKey()
		return b.Put([]byte(accountKey), accInfoBytes)
	})
	return err
}

func (c *ClICtrl) GetAccounts(cxt context.Context) ([]model.Account, error) {
	var accounts []model.Account
	err := c.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(AccBucket))
		err := b.ForEach(func(k, v []byte) error {
			var info model.Account
			err := json.Unmarshal(v, &info)
			if err != nil {
				return err
			}
			accounts = append(accounts, info)
			return nil
		})
		if err != nil {
			return err
		}
		return nil
	})
	return accounts, err
}

func (c *ClICtrl) ListAccounts(cxt context.Context) error {
	accounts, err := c.GetAccounts(cxt)
	if err != nil {
		return err
	}
	fmt.Printf("Configured accounts: %d\n", len(accounts))
	for _, acc := range accounts {
		fmt.Println("====================================")
		fmt.Println("Email: ", acc.Email)
		fmt.Println("ID:    ", acc.UserID)
		fmt.Println("App:   ", acc.App)
		fmt.Println("====================================")
	}
	return nil
}
