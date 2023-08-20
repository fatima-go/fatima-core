package crypt

import (
	"context"
	"encoding/json"
	"errors"
	log "github.com/fatima-go/fatima-log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type AWSSecretManager struct {
	config    aws.Config
	client    *secretsmanager.Client
	secretMap map[string]string
	secretID  string
}

func NewAWSSecretManager() *AWSSecretManager {
	return &AWSSecretManager{}
}

func (sm *AWSSecretManager) SetSecretID(id string) {
	sm.secretID = id
}

func (sm *AWSSecretManager) SetAWSConfig(cfg aws.Config) {
	sm.config = cfg
}

func (sm *AWSSecretManager) GetSecretValue(key string) string {
	val, found := sm.secretMap[key]
	if !found {
		log.Error("key not found in secret cache [key: %s]", key)
	}

	return val
}

func (sm *AWSSecretManager) createAWSClient() error {
	if sm.config.Region == "" {
		cfg, err := config.LoadDefaultConfig(context.Background())
		if err != nil {
			return err
		}

		sm.config = cfg
	}

	sm.client = secretsmanager.NewFromConfig(sm.config)
	return nil
}

func (sm *AWSSecretManager) CacheSecretValues() error {
	if sm.secretID == "" {
		return errors.New("secret ID is not specified")
	}

	err := sm.createAWSClient()
	if err != nil {
		return err
	}

	secret, err := sm.client.GetSecretValue(context.Background(),
		&secretsmanager.GetSecretValueInput{
			SecretId: aws.String(sm.secretID),
		})
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(*secret.SecretString), &sm.secretMap)
}
