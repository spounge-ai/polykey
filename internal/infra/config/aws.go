package config

// AWSConfig represents the AWS configuration.
type AWSConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	Region    string `mapstructure:"region"     validate:"required_if=Enabled true"`
	S3Bucket  string `mapstructure:"s3_bucket"  validate:"required_if=Enabled true"`
	CacheTTL  string `mapstructure:"cache_ttl"`
}
