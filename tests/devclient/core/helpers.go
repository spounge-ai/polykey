package core

import (
	"context"
)

func HandleResult(tc TestClient, testName string, err error) {
	if err != nil {
		classifiedErr := tc.ErrorClassifier().Classify(err, testName)
		_ = tc.ErrorClassifier().LogAndSanitize(context.Background(), classifiedErr)
	} else {
		tc.Logger().Info("Test passed", "test", testName)
	}
}
