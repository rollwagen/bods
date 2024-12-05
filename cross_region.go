package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/aws/aws-sdk-go-v2/service/bedrock"
	"github.com/aws/aws-sdk-go-v2/service/bedrock/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/tidwall/buntdb"
)

func cacheDBFilePath() string {
	cacheDir := filepath.Join(xdg.CacheHome, "bods")
	_, err := os.Stat(cacheDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(cacheDir, 0o755)
		if err != nil {
			logger.Printf("could not create directory %s: %v", cacheDir, err)
		}
	}

	dbFilePath := filepath.Join(xdg.CacheHome, "bods/cache.db")
	return dbFilePath
}

func getCache(region string, modelID string) (string, error) {
	path := cacheDBFilePath()
	db, err := buntdb.Open(path)
	if err != nil {
		logger.Println("getfromCache() buntdb.Open - ", err)
		return "", err
	}
	defer db.Close()

	var cacheContent string
	err = db.View(func(tx *buntdb.Tx) error {
		val, err := tx.Get(region + ":" + modelID)
		if err != nil {
			logger.Println("getfromCache() tx.Get - ", err)
			return err
		}
		cacheContent = val
		return nil
	})

	return cacheContent, err
}

func updateCache(region string, modelID string, inferenceProfileID string) error {
	const expiresTTLSeconds = 60 * 60 // 1 hour

	path := cacheDBFilePath()
	db, err := buntdb.Open(path)
	if err != nil {
		logger.Println("updateCache() - ", err)
		return err
	}
	defer db.Close()

	return db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(region+":"+modelID, inferenceProfileID, &buntdb.SetOptions{Expires: true, TTL: time.Second * expiresTTLSeconds})
		return err
	})
}

// given a model id, return the cross region inference profile id, if it exists
func crossRegionInferenceProfileID(client bedrock.Client, modelID string, region string) (string, error) {
	nanoToSec := func(nano time.Duration) float64 {
		return float64(nano) / float64(time.Second)
	}

	// if given an inference profile id as input, just return the same
	startsWithTwoLettersAndDot := func(s string) bool {
		if len(s) < 3 {
			return false
		}
		return strings.HasPrefix(s, s[0:2]+".")
	}
	if startsWithTwoLettersAndDot(modelID) {
		return modelID, nil
	}

	start := time.Now()
	inferenceProfilesIDFromCache, err := getCache(region, modelID)
	if err == nil {
		if inferenceProfilesIDFromCache != "" {
			logger.Printf("cache lookup in %f seconds  inferenceProfilesIDFromCache = %s\n", nanoToSec(time.Since(start)), inferenceProfilesIDFromCache)
			return inferenceProfilesIDFromCache, nil
		}
	}

	start = time.Now()
	listInferenceProfilesInput := bedrock.ListInferenceProfilesInput{MaxResults: aws.Int32(1000), TypeEquals: types.InferenceProfileTypeSystemDefined}
	inferenceProfiles, err := client.ListInferenceProfiles(context.Background(), &listInferenceProfilesInput)
	logger.Printf("bedrock api call ListInferenceProfiles took %f seconds\n", nanoToSec(time.Since(start)))
	if err != nil {
		return "", err
	}
	for _, summary := range inferenceProfiles.InferenceProfileSummaries {
		parts := strings.Split(*summary.InferenceProfileId, ".")
		inferenceModelID := strings.Join(parts[1:], ".")
		if inferenceModelID == modelID {
			err := updateCache(region, modelID, *summary.InferenceProfileId)
			if err != nil {
				logger.Printf("updateCache(%s, %s, %s) error: %s", region, modelID, *summary.InferenceProfileId, err)
			}
			return *summary.InferenceProfileId, nil
		}

	}

	return "", fmt.Errorf("no cross-region inference profile fo model_id=%s", modelID)
}
