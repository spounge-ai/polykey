package secrets

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

type ParameterStore struct {
	client *ssm.Client
}

func NewParameterStore(cfg aws.Config) *ParameterStore {
	return &ParameterStore{client: ssm.NewFromConfig(cfg)}
}

func (ps *ParameterStore) GetSecret(ctx context.Context, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("secret name cannot be empty")
	}

	input := &ssm.GetParameterInput{
		Name:           &name,
		WithDecryption: aws.Bool(true),
	}

	result, err := ps.client.GetParameter(ctx, input)
	if err != nil {
		return "", err
	}

	return *result.Parameter.Value, nil
}